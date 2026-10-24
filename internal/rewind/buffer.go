package rewind

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// segmentLength is how long a segment file gets before the next keyframe
// starts a new one. Shorter is finer for seeking and catching up, longer is
// fewer files.
const segmentLength = 5 * time.Second

// Segment is a closed, playable piece of the buffer.
type Segment struct {
	N     int   `json:"n"`
	Start int64 `json:"start"` // ms on the buffer timeline
	End   int64 `json:"end"`
	Size  int64 `json:"size"`
}

// buffer is the last minutes of a room on disk, as WebM segments on one
// timeline: milliseconds since the buffer started.
type buffer struct {
	mu    sync.Mutex
	dir   string
	epoch time.Time
	next  int
	segs  []Segment
	cur   *segWriter
	video func() // called for each video frame
}

type segWriter struct {
	seg     Segment
	f       *os.File
	w       *bufio.Writer
	width   int
	height  int
	pending []frame // the cluster being collected
	last    int64
}

func newBuffer(dir string) (*buffer, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &buffer{dir: dir, epoch: time.Now(), next: 1}, nil
}

// now is the current time on the buffer timeline
func (b *buffer) now() int64 {
	return time.Since(b.epoch).Milliseconds()
}

func (b *buffer) path(n int) string {
	return filepath.Join(b.dir, fmt.Sprintf("%d.webm", n))
}

// write adds a frame. Segments start at video keyframes, frames before the
// first one are dropped.
func (b *buffer) write(f frame) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if f.video && b.video != nil {
		b.video()
	}
	if f.video {
		key, width, height := vp8Key(f.data)
		f.key = key
		if key {
			c := b.cur
			if c == nil || f.ts-c.seg.Start >= segmentLength.Milliseconds() || width != c.width || height != c.height {
				b.closeLocked()
				if err := b.openLocked(f.ts, width, height); err != nil {
					return err
				}
			}
		}
	}

	c := b.cur
	if c == nil {
		return nil
	}

	// a cluster per keyframe, so players can start from any cluster
	if (f.video && f.key) || (len(c.pending) > 0 && f.ts-c.pending[0].ts > maxClusterSpan) {
		if err := c.flush(); err != nil {
			b.dropLocked()
			return err
		}
	}
	c.pending = append(c.pending, f)
	if f.ts < c.seg.Start {
		c.seg.Start = f.ts
	}
	if f.ts > c.last {
		c.last = f.ts
	}
	return nil
}

func (b *buffer) openLocked(ts int64, width, height int) error {
	n := b.next
	b.next++
	file, err := os.Create(b.path(n))
	if err != nil {
		return err
	}
	w := bufio.NewWriterSize(file, 256<<10)
	if err := writeHeader(w, width, height); err != nil {
		file.Close()
		os.Remove(file.Name())
		return err
	}
	b.cur = &segWriter{
		seg:    Segment{N: n, Start: ts},
		f:      file,
		w:      w,
		width:  width,
		height: height,
		last:   ts,
	}
	return nil
}

func (c *segWriter) flush() error {
	if err := writeCluster(c.w, c.pending); err != nil {
		return err
	}
	c.pending = c.pending[:0]
	return nil
}

// closeLocked finishes the current segment and makes it playable
func (b *buffer) closeLocked() {
	c := b.cur
	if c == nil {
		return
	}
	b.cur = nil
	err := c.flush()
	if err == nil {
		err = c.w.Flush()
	}
	if cerr := c.f.Close(); err == nil {
		err = cerr
	}
	if err != nil {
		os.Remove(c.f.Name())
		return
	}
	if info, err := os.Stat(c.f.Name()); err == nil {
		c.seg.Size = info.Size()
	}
	// video frames last until the next one, about one frame at 25 fps
	c.seg.End = c.last + 40
	b.segs = append(b.segs, c.seg)
}

// dropLocked throws away the current segment after a write error
func (b *buffer) dropLocked() {
	if c := b.cur; c != nil {
		c.f.Close()
		os.Remove(c.f.Name())
		b.cur = nil
	}
}

// onVideo sets what is called for each video frame, nil for nothing
func (b *buffer) onVideo(f func()) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.video = f
}

// cut closes the current segment, when recording stops
func (b *buffer) cut() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.closeLocked()
}

// prune deletes segments that ended longer than keep ago
func (b *buffer) prune(keep time.Duration) {
	b.mu.Lock()
	defer b.mu.Unlock()
	limit := b.now() - keep.Milliseconds()
	i := 0
	for i < len(b.segs) && b.segs[i].End < limit {
		os.Remove(b.path(b.segs[i].N))
		i++
	}
	b.segs = append(b.segs[:0:0], b.segs[i:]...)
}

// segments are the closed segments, oldest first
func (b *buffer) segments() []Segment {
	b.mu.Lock()
	defer b.mu.Unlock()
	return append([]Segment(nil), b.segs...)
}

// open opens a closed segment for reading
func (b *buffer) open(n int) (*os.File, Segment, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, s := range b.segs {
		if s.N == n {
			f, err := os.Open(b.path(n))
			return f, s, err
		}
	}
	return nil, Segment{}, os.ErrNotExist
}

// remove deletes the buffer and its files
func (b *buffer) remove() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.dropLocked()
	b.segs = nil
	os.RemoveAll(b.dir)
}

// clock turns RTP timestamps of a track into ms on the buffer timeline. It
// starts at the arrival of the first packet and follows the RTP clock from
// there, so frames keep their spacing. When the two drift apart, e.g. after
// the room's encoder restarted, it starts over from the arrival time.
type clock struct {
	rate    int64 // RTP clock rate
	started bool
	last    uint32
	ext     int64 // RTP timestamp without wrap-arounds
	baseExt int64
	baseMs  int64
}

const clockDrift = 2000 // ms

func (c *clock) at(rtpTs uint32, now int64) int64 {
	if !c.started {
		c.started = true
		c.last = rtpTs
		c.baseMs = now
		return now
	}
	c.ext += int64(int32(rtpTs - c.last))
	c.last = rtpTs
	ts := c.baseMs + (c.ext-c.baseExt)*1000/c.rate
	if ts-now > clockDrift || now-ts > clockDrift {
		c.baseExt = c.ext
		c.baseMs = now
		ts = now
	}
	return ts
}
