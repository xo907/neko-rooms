package rewind

import (
	"bytes"
	"context"
	"testing"
	"time"
)

func TestStreamPositions(t *testing.T) {
	segs := []Segment{
		{N: 1, Start: 1000, End: 6040},
		{N: 2, Start: 6000, End: 11040},  // follows on
		{N: 3, Start: 70000, End: 75040}, // after a minute nobody watched
		{N: 4, Start: 75000, End: 80040},
	}
	got := StreamPositions(segs)
	want := []int64{0, 5000, 10040, 15040}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("positions = %v, want %v", got, want)
		}
	}
}

// clusterTimecodes lists the cluster timecodes of a stream
func clusterTimecodes(t *testing.T, data []byte) []uint64 {
	header, clusters, err := splitSegment(data)
	if err != nil || len(header) == 0 {
		t.Fatalf("split: %v", err)
	}
	var tcs []uint64
	for len(clusters) > 0 {
		_, n, _ := readID(clusters)
		size, m, _ := readVint(clusters[n:])
		body := clusters[n+m : n+m+int(size)]
		_, tn, _ := readID(body)
		ts, tm, _ := readVint(body[tn:])
		var tc uint64
		for _, c := range body[tn+tm : tn+tm+int(ts)] {
			tc = tc<<8 | uint64(c)
		}
		tcs = append(tcs, tc)
		clusters = clusters[n+m+int(size):]
	}
	return tcs
}

func TestStream(t *testing.T) {
	old := streamIdle
	streamIdle = 300 * time.Millisecond
	defer func() { streamIdle = old }()

	b, err := newBuffer(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	// 12 seconds, then nothing for a minute, then 12 more
	for _, base := range []int64{1000, 70000} {
		for s := int64(0); s < 12; s++ {
			ts := base + s*1000
			b.write(frame{video: true, ts: ts, data: keyframe(640, 360)})
			b.write(frame{ts: ts + 5, data: []byte{1}})
		}
	}
	b.cut()
	segs := b.segments()
	if len(segs) != 6 {
		t.Fatalf("segments: %+v", segs)
	}

	m := &Manager{entries: map[string]*entry{"r": {buf: b}}}
	var out bytes.Buffer
	flushes := 0
	if err := m.Stream(context.Background(), "r", segs[1].N, &out, func() { flushes++ }); err != nil {
		t.Fatal(err)
	}
	if flushes != 5 {
		t.Fatalf("flushed %d times, want once per segment", flushes)
	}

	// one header, then clusters from zero, the minute gap closed
	if bytes.Count(out.Bytes(), []byte{0x1A, 0x45, 0xDF, 0xA3}) != 1 {
		t.Fatal("more than one header")
	}
	tcs := clusterTimecodes(t, out.Bytes())
	if tcs[0] != 0 {
		t.Fatalf("first timecode %d", tcs[0])
	}
	for i := 1; i < len(tcs); i++ {
		if d := int64(tcs[i]) - int64(tcs[i-1]); d < 0 || d > 1100 {
			t.Fatalf("timecodes jump: %v", tcs)
		}
	}

	// segments that come later are added while the stream runs
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var live bytes.Buffer
	done := make(chan error)
	go func() { done <- m.Stream(ctx, "r", segs[len(segs)-1].N, &live, nil) }()
	time.Sleep(100 * time.Millisecond)
	for s := int64(0); s < 6; s++ {
		ts := 90000 + s*1000
		b.write(frame{video: true, ts: ts, data: keyframe(640, 360)})
	}
	b.cut()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if n := len(clusterTimecodes(t, live.Bytes())); n < 6 {
		t.Fatalf("followed %d clusters", n)
	}

	if err := m.Stream(context.Background(), "nope", 1, &out, nil); err == nil {
		t.Fatal("stream of an unknown room")
	}
}
