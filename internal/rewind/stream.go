package rewind

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"time"
)

// Stream plays the recording as one WebM, for players that take a single
// progressive file instead of segments, like the Android app. It starts at
// segment from, counts time from zero, closes the gaps when nobody watched
// and keeps adding segments while the room records.
//
// Where a segment lands is up to StreamPositions, so players can map their
// position back to the timeline.

// streamIdle ends a stream when no new segment came for this long
var streamIdle = 15 * time.Second

// gapLimit: segments further apart than this are joined without the gap
const gapLimit = 1000

// StreamPositions is where each segment starts in a stream, in ms, for the
// segments in order. Segments that follow each other keep their spacing,
// gaps are closed.
func StreamPositions(segs []Segment) []int64 {
	pos := make([]int64, len(segs))
	for i := 1; i < len(segs); i++ {
		prev, s := segs[i-1], segs[i]
		if s.Start-prev.End <= gapLimit {
			pos[i] = pos[i-1] + (s.Start - prev.Start)
		} else {
			pos[i] = pos[i-1] + (prev.End - prev.Start)
		}
	}
	return pos
}

// Stream writes the recording of a room from segment from on to w, calling
// flush after each segment, until ctx ends or nothing new comes.
func (m *Manager) Stream(ctx context.Context, room string, from int, w io.Writer, flush func()) error {
	m.mu.Lock()
	e := m.entries[room]
	m.mu.Unlock()
	if e == nil || e.buf == nil {
		return os.ErrNotExist
	}
	buf := e.buf

	var played []Segment
	var pos int64
	next := from
	idleSince := time.Now()
	headerDone := false

	for {
		var seg *Segment
		for _, s := range buf.segments() {
			if s.N >= next {
				s := s
				seg = &s
				break
			}
		}
		if seg == nil {
			if len(played) == 0 {
				return os.ErrNotExist
			}
			if time.Since(idleSince) > streamIdle {
				return nil
			}
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(500 * time.Millisecond):
			}
			continue
		}
		next = seg.N + 1
		idleSince = time.Now()

		f, _, err := buf.open(seg.N)
		if err != nil {
			continue // pruned meanwhile
		}
		data, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			continue
		}
		header, clusters, err := splitSegment(data)
		if err != nil {
			continue
		}

		played = append(played, *seg)
		pos = StreamPositions(played)[len(played)-1]

		var out bytes.Buffer
		if !headerDone {
			out.Write(header)
			headerDone = true
		}
		if err := shiftClusters(&out, clusters, seg.Start-pos); err != nil {
			return err
		}
		if _, err := w.Write(out.Bytes()); err != nil {
			return err
		}
		if flush != nil {
			flush()
		}
	}
}

var errFormat = errors.New("not a rewind segment")

// readVint reads an EBML variable size integer, without its length marker
func readVint(b []byte) (val uint64, n int, err error) {
	if len(b) == 0 || b[0] == 0 {
		return 0, 0, errFormat
	}
	n = 1
	for mask := byte(0x80); b[0]&mask == 0; mask >>= 1 {
		n++
	}
	if len(b) < n {
		return 0, 0, errFormat
	}
	val = uint64(b[0] & (0xFF >> n))
	for i := 1; i < n; i++ {
		val = val<<8 | uint64(b[i])
	}
	return val, n, nil
}

// readID reads an element ID with its marker bits
func readID(b []byte) (id uint32, n int, err error) {
	if len(b) == 0 || b[0] == 0 {
		return 0, 0, errFormat
	}
	n = 1
	for mask := byte(0x80); b[0]&mask == 0; mask >>= 1 {
		n++
	}
	if n > 4 || len(b) < n {
		return 0, 0, errFormat
	}
	for i := 0; i < n; i++ {
		id = id<<8 | uint32(b[i])
	}
	return id, n, nil
}

// splitSegment separates a segment file as written by the buffer into the
// part before the first cluster and the clusters
func splitSegment(data []byte) (header []byte, clusters []byte, err error) {
	// EBML header
	id, n, err := readID(data)
	if err != nil || id != idEBML {
		return nil, nil, errFormat
	}
	size, m, err := readVint(data[n:])
	if err != nil {
		return nil, nil, err
	}
	p := n + m + int(size)
	// Segment of unknown size
	id, n, err = readID(data[p:])
	if err != nil || id != idSegment {
		return nil, nil, errFormat
	}
	_, m, err = readVint(data[p+n:])
	if err != nil {
		return nil, nil, err
	}
	p += n + m
	// its children up to the first cluster
	for p < len(data) {
		id, n, err := readID(data[p:])
		if err != nil {
			return nil, nil, err
		}
		if id == idCluster {
			return data[:p], data[p:], nil
		}
		size, m, err := readVint(data[p+n:])
		if err != nil {
			return nil, nil, err
		}
		p += n + m + int(size)
	}
	return data, nil, nil
}

// shiftClusters copies clusters with their timecodes moved back by shift
func shiftClusters(w *bytes.Buffer, data []byte, shift int64) error {
	for len(data) > 0 {
		id, n, err := readID(data)
		if err != nil {
			return err
		}
		size, m, err := readVint(data[n:])
		if err != nil || len(data) < n+m+int(size) {
			return errFormat
		}
		body := data[n+m : n+m+int(size)]
		data = data[n+m+int(size):]
		if id != idCluster {
			continue
		}

		// the timecode comes first, as the buffer writes it
		tid, tn, err := readID(body)
		if err != nil || tid != idTimecode {
			return errFormat
		}
		tsize, tm, err := readVint(body[tn:])
		if err != nil || len(body) < tn+tm+int(tsize) {
			return errFormat
		}
		var tc uint64
		for _, c := range body[tn+tm : tn+tm+int(tsize)] {
			tc = tc<<8 | uint64(c)
		}
		rest := body[tn+tm+int(tsize):]
		shifted := int64(tc) - shift
		if shifted < 0 {
			shifted = 0
		}

		var inner bytes.Buffer
		elementUint(&inner, idTimecode, uint64(shifted))
		inner.Write(rest)
		element(w, idCluster, inner.Bytes())
	}
	return nil
}
