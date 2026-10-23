package room

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"

	"github.com/m1k1o/neko-rooms/internal/types"
)

// poller turns the room list into room events, for runtimes without an
// event stream of their own (swarm services, kubernetes pods). It compares
// the rooms every few seconds and reports what changed.
type poller struct {
	logger   zerolog.Logger
	list     func(ctx context.Context) ([]types.RoomEntry, error)
	interval time.Duration

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	listenersMu sync.Mutex
	listeners   []chan types.RoomEvent
}

func newPoller(logger zerolog.Logger, list func(ctx context.Context) ([]types.RoomEntry, error)) *poller {
	return &poller{logger: logger, list: list, interval: 3 * time.Second}
}

func (p *poller) Start() {
	p.ctx, p.cancel = context.WithCancel(context.Background())

	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		p.logger.Info().Msg("room poller started")
		defer p.logger.Info().Msg("room poller stopped")

		var prev map[string]types.RoomEntry
		ticker := time.NewTicker(p.interval)
		defer ticker.Stop()

		for {
			ctx, cancel := context.WithTimeout(p.ctx, 30*time.Second)
			entries, err := p.list(ctx)
			cancel()
			if err != nil {
				p.logger.Warn().Err(err).Msg("unable to list rooms")
			} else {
				next := make(map[string]types.RoomEntry, len(entries))
				for _, e := range entries {
					next[e.ID] = e
				}
				// the first round only records, listeners start from a list
				if prev != nil {
					p.diff(prev, next)
				}
				prev = next
			}

			select {
			case <-p.ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}

func (p *poller) diff(prev, next map[string]types.RoomEntry) {
	for id, cur := range next {
		old, ok := prev[id]
		if !ok {
			p.emit(cur, types.RoomEventCreated)
			if cur.Running {
				p.emit(cur, types.RoomEventStarted)
			}
			if cur.IsReady {
				p.emit(cur, types.RoomEventReady)
			}
			continue
		}

		switch {
		case !old.Running && cur.Running:
			p.emit(cur, types.RoomEventStarted)
			if cur.IsReady {
				p.emit(cur, types.RoomEventReady)
			}
		case old.Running && !cur.Running:
			p.emit(cur, types.RoomEventStopped)
		case !old.IsReady && cur.IsReady:
			p.emit(cur, types.RoomEventReady)
		}
	}

	for id, old := range prev {
		if _, ok := next[id]; !ok {
			p.emit(old, types.RoomEventDestroyed)
		}
	}
}

func (p *poller) emit(e types.RoomEntry, action types.RoomEventAction) {
	p.logger.Debug().Str("id", e.ID).Str("action", string(action)).Msg("room event")
	event := types.RoomEvent{
		ID:              e.ID,
		Action:          action,
		ContainerLabels: e.ContainerLabels,
	}
	p.listenersMu.Lock()
	for _, listener := range p.listeners {
		// a listener that stopped reading must not hold up the others
		select {
		case listener <- event:
		case <-time.After(5 * time.Second):
			p.logger.Warn().Str("id", e.ID).Msg("room event listener is not reading, dropped an event")
		case <-p.ctx.Done():
		}
	}
	p.listenersMu.Unlock()
}

func (p *poller) Shutdown() error {
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait()
	return nil
}

func (p *poller) Events(ctx context.Context) (<-chan types.RoomEvent, <-chan error) {
	messages := make(chan types.RoomEvent)
	errs := make(chan error, 1)

	p.listenersMu.Lock()
	p.listeners = append(p.listeners, messages)
	p.listenersMu.Unlock()

	go func() {
		defer close(errs)

		select {
		case <-ctx.Done():
			errs <- ctx.Err()
		case <-p.ctx.Done():
			errs <- fmt.Errorf("room events shutdown")
			return
		}

		p.listenersMu.Lock()
		for i, listener := range p.listeners {
			if listener == messages {
				p.listeners = append(p.listeners[:i], p.listeners[i+1:]...)
				break
			}
		}
		p.listenersMu.Unlock()
	}()

	return messages, errs
}
