// Package rewind keeps the last minutes of rooms people watch, so someone
// who joins late or comes back can see what they missed and catch up.
//
// A hidden viewer records what neko already sends to everyone, as it is: no
// decoding, no encoding, about the cost of one more viewer. It only records
// while people watch, and the recording is gone when the room stops.
package rewind

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/m1k1o/neko-rooms/internal/types"
)

const (
	// keep recording this long after the last person left
	idleStop = 2 * time.Minute
	// a failed recorder tries again after this, doubling up to retryMax
	retryMin = 30 * time.Second
	retryMax = 10 * time.Minute
	// rooms that can not be recorded are left alone this long
	unsupportedWait = 30 * time.Minute
)

// Room is what the manager needs to know about a room, on every refresh.
type Room struct {
	Name    string
	ID      string
	Ready   bool
	Viewers int
}

// Index describes what can be played back for a room.
type Index struct {
	Epoch     int64     `json:"epoch"` // unix ms of timeline zero
	Now       int64     `json:"now"`   // timeline ms right now
	Keep      int64     `json:"keep"`  // ms kept
	Recording bool      `json:"recording"`
	Segments  []Segment `json:"segments"`
}

type entry struct {
	id          string
	buf         *buffer
	cancel      context.CancelFunc
	lastViewers time.Time
	failures    int
	retryAt     time.Time
}

type Manager struct {
	logger zerolog.Logger
	rooms  types.RoomManager
	dir    string

	mu      sync.Mutex
	keep    time.Duration
	entries map[string]*entry // by room name
	closed  bool
}

// New starts with an empty directory, recordings do not outlive a restart.
func New(rooms types.RoomManager, dir string) (*Manager, error) {
	if err := os.RemoveAll(dir); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Manager{
		logger:  log.With().Str("module", "rewind").Logger(),
		rooms:   rooms,
		dir:     dir,
		entries: map[string]*entry{},
	}, nil
}

// Sync starts and stops recorders for the rooms, keep is how much to keep, 0
// turns rewinding off.
func (m *Manager) Sync(rooms []Room, keep time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return
	}
	m.keep = keep
	now := time.Now()

	seen := map[string]bool{}
	for _, r := range rooms {
		if !r.Ready || keep <= 0 {
			continue
		}
		seen[r.Name] = true

		e := m.entries[r.Name]
		if e != nil && e.id != r.ID {
			// a new container with the same name
			m.dropLocked(r.Name)
			e = nil
		}
		if e == nil {
			e = &entry{id: r.ID}
			m.entries[r.Name] = e
		}
		if r.Viewers > 0 {
			e.lastViewers = now
		}
		if e.buf != nil {
			e.buf.prune(keep)
		}

		want := !e.lastViewers.IsZero() && now.Sub(e.lastViewers) < idleStop
		switch {
		case want && e.cancel == nil && !now.Before(e.retryAt):
			m.startLocked(r.Name, e)
		case !want && e.cancel != nil:
			e.cancel()
			e.cancel = nil
		}
	}

	for name := range m.entries {
		if !seen[name] {
			m.dropLocked(name)
		}
	}
}

func (m *Manager) startLocked(name string, e *entry) {
	if e.buf == nil {
		dir, err := os.MkdirTemp(m.dir, "room-")
		if err == nil {
			e.buf, err = newBuffer(dir)
		}
		if err != nil {
			m.logger.Err(err).Str("room", name).Msg("unable to create rewind buffer")
			e.retryAt = time.Now().Add(retryMax)
			return
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	e.cancel = cancel
	buf := e.buf
	logger := m.logger.With().Str("room", name).Logger()

	go func() {
		started := time.Now()
		err := m.record(ctx, e.id, buf, logger)
		buf.cut()

		m.mu.Lock()
		defer m.mu.Unlock()
		if m.entries[name] != e {
			return
		}
		if ctx.Err() != nil {
			return // stopped on purpose
		}
		cancel()
		e.cancel = nil

		var unsupported errUnsupported
		switch {
		case errors.As(err, &unsupported):
			logger.Info().Err(err).Msg("room can not be rewound")
			e.retryAt = time.Now().Add(unsupportedWait)
			return
		case time.Since(started) > retryMax:
			e.failures = 0
		}
		e.failures++
		wait := retryMin << min(e.failures-1, 5)
		if wait > retryMax {
			wait = retryMax
		}
		e.retryAt = time.Now().Add(wait)
		logger.Warn().Err(err).Dur("retry", wait).Msg("rewind recorder stopped")
	}()
}

func (m *Manager) record(ctx context.Context, id string, buf *buffer, logger zerolog.Logger) error {
	entry, err := m.rooms.GetEntry(ctx, id)
	if err != nil {
		return err
	}
	settings, err := m.rooms.GetSettings(ctx, id)
	if err != nil {
		return err
	}
	if settings.ApiVersion != 3 {
		return errUnsupported{"neko v2"}
	}
	host, ok := roomHost(entry)
	if !ok {
		return errUnsupported{"room is not reachable"}
	}

	logger.Info().Msg("rewind recorder started")
	r := &recorder{logger: logger, host: host, pass: settings.UserPass, buf: buf}
	return r.run(ctx)
}

// roomHost is where a room's web port is reached, like the proxy does
func roomHost(e *types.RoomEntry) (string, bool) {
	l := e.ContainerLabels
	if h := l["m1k1o.neko_rooms.proxy.host"]; h != "" {
		return h, true
	}
	port := l["m1k1o.neko_rooms.proxy.port"]
	if port == "" {
		return "", false
	}
	return net.JoinHostPort(e.ID, port), true
}

func (m *Manager) dropLocked(name string) {
	e := m.entries[name]
	if e == nil {
		return
	}
	delete(m.entries, name)
	if e.cancel != nil {
		e.cancel()
	}
	if e.buf != nil {
		// let the recorder finish writing first
		buf := e.buf
		time.AfterFunc(5*time.Second, buf.remove)
	}
}

// Index is what there is to play back for a room.
func (m *Manager) Index(name string) (Index, bool) {
	m.mu.Lock()
	e := m.entries[name]
	keep := m.keep
	var buf *buffer
	recording := false
	if e != nil {
		buf = e.buf
		recording = e.cancel != nil
	}
	m.mu.Unlock()
	if buf == nil {
		return Index{}, false
	}

	idx := Index{
		Epoch:     buf.epoch.UnixMilli(),
		Now:       buf.now(),
		Keep:      keep.Milliseconds(),
		Recording: recording,
		Segments:  []Segment{},
	}
	limit := idx.Now - idx.Keep
	for _, s := range buf.segments() {
		if s.End >= limit {
			idx.Segments = append(idx.Segments, s)
		}
	}
	return idx, true
}

// Open opens segment n of a room.
func (m *Manager) Open(name string, n int) (*os.File, Segment, error) {
	m.mu.Lock()
	e := m.entries[name]
	m.mu.Unlock()
	if e == nil || e.buf == nil {
		return nil, Segment{}, os.ErrNotExist
	}
	return e.buf.open(n)
}

// Shutdown stops all recorders and deletes the recordings.
func (m *Manager) Shutdown() {
	m.mu.Lock()
	m.closed = true
	for name, e := range m.entries {
		delete(m.entries, name)
		if e.cancel != nil {
			e.cancel()
		}
	}
	m.mu.Unlock()
	// recorders log out of their rooms on the way
	time.Sleep(500 * time.Millisecond)
	os.RemoveAll(filepath.Clean(m.dir))
}
