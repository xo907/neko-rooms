package rewind

import (
	"bytes"
	"os"
	"testing"
	"time"
)

func TestPutSize(t *testing.T) {
	cases := map[uint64][]byte{
		0:     {0x80},
		5:     {0x85},
		126:   {0xFE},
		127:   {0x40, 0x7F}, // 0xFF would mean unknown size
		300:   {0x41, 0x2C},
		16382: {0x7F, 0xFE},
		16383: {0x20, 0x3F, 0xFF},
	}
	for n, want := range cases {
		var b bytes.Buffer
		putSize(&b, n)
		if !bytes.Equal(b.Bytes(), want) {
			t.Errorf("putSize(%d) = %x, want %x", n, b.Bytes(), want)
		}
	}
}

// keyframe is a VP8 keyframe header of the given size
func keyframe(w, h int) []byte {
	return []byte{0x10, 0x02, 0x00, 0x9d, 0x01, 0x2a, byte(w), byte(w >> 8), byte(h), byte(h >> 8), 0xAA}
}

func TestVP8Key(t *testing.T) {
	key, w, h := vp8Key(keyframe(1280, 720))
	if !key || w != 1280 || h != 720 {
		t.Fatalf("keyframe = %v %dx%d", key, w, h)
	}
	if key, _, _ := vp8Key([]byte{0x11, 0, 0, 0, 0, 0, 0, 0, 0, 0}); key {
		t.Fatal("interframe seen as keyframe")
	}
}

func TestClusterOrderAndTimecode(t *testing.T) {
	var b bytes.Buffer
	err := writeCluster(&b, []frame{
		{video: true, key: true, ts: 5020, data: []byte{1}},
		{video: false, ts: 5000, data: []byte{2}},
	})
	if err != nil {
		t.Fatal(err)
	}
	d := b.Bytes()
	// cluster id, size, then the timecode of the earliest frame
	if !bytes.HasPrefix(d, []byte{0x1F, 0x43, 0xB6, 0x75}) {
		t.Fatalf("no cluster: %x", d)
	}
	tc := []byte{0xE7, 0x82, 0x13, 0x88} // 5000
	if !bytes.Contains(d, tc) {
		t.Fatalf("timecode 5000 missing: %x", d)
	}
	// audio first (track 2 at +0), then video (track 1 at +20, keyframe)
	audio := []byte{0xA3, 0x85, 0x82, 0x00, 0x00, 0x80, 0x02}
	video := []byte{0xA3, 0x85, 0x81, 0x00, 0x14, 0x80, 0x01}
	ia, iv := bytes.Index(d, audio), bytes.Index(d, video)
	if ia < 0 || iv < 0 || ia > iv {
		t.Fatalf("blocks out of order or missing: %x", d)
	}
}

func TestBufferSegments(t *testing.T) {
	b, err := newBuffer(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	inter := []byte{0x11, 0, 0, 0, 0, 0, 0, 0, 0, 0}

	// before the first keyframe nothing is kept
	b.write(frame{video: true, ts: 0, data: inter})
	b.write(frame{ts: 10, data: []byte{1}})
	if b.cur != nil {
		t.Fatal("segment started without a keyframe")
	}

	// keyframes every second, sound in between, for 12 seconds
	for s := int64(1); s <= 12; s++ {
		ts := s * 1000
		b.write(frame{video: true, ts: ts, data: keyframe(640, 360)})
		b.write(frame{ts: ts + 5, data: []byte{1}})
		b.write(frame{video: true, ts: ts + 500, data: inter})
	}
	segs := b.segments()
	if len(segs) != 2 {
		t.Fatalf("segments = %+v, want 2 closed", segs)
	}
	if segs[0].Start != 1000 || segs[1].Start != 6000 {
		t.Fatalf("segment starts = %d, %d", segs[0].Start, segs[1].Start)
	}
	if segs[0].End < 5500 || segs[0].End >= 6000+100 {
		t.Fatalf("segment end = %d", segs[0].End)
	}

	// a size change starts a new segment right away
	b.write(frame{video: true, ts: 12100, data: keyframe(1280, 720)})
	if got := len(b.segments()); got != 3 {
		t.Fatalf("after resize %d segments, want 3", got)
	}

	f, s, err := b.open(segs[0].N)
	if err != nil {
		t.Fatal(err)
	}
	data := make([]byte, 4)
	f.Read(data)
	f.Close()
	if !bytes.Equal(data, []byte{0x1A, 0x45, 0xDF, 0xA3}) || s.Size == 0 {
		t.Fatalf("segment file starts with %x, size %d", data, s.Size)
	}

	// everything older than keep goes
	b.epoch = time.Now().Add(-60 * time.Second)
	b.prune(49 * time.Second)
	left := b.segments()
	if len(left) != 1 || left[0].Start != 11000 {
		t.Fatalf("after prune %+v", left)
	}
	if _, err := os.Stat(b.path(segs[0].N)); !os.IsNotExist(err) {
		t.Fatal("pruned file still there")
	}
}

func TestClock(t *testing.T) {
	c := &clock{rate: 90000}
	var start uint32 = 4294967000
	if ts := c.at(start, 1000); ts != 1000 {
		t.Fatalf("first = %d", ts)
	}
	// across the 32 bit wrap, one second later
	if ts := c.at(start+90000, 2000); ts != 2000 {
		t.Fatalf("after wrap = %d", ts)
	}
	// follows the RTP clock, not arrival jitter
	if ts := c.at(start+90000*2, 3030); ts != 3000 {
		t.Fatalf("jitter = %d", ts)
	}
	// a jump starts over at the arrival time
	if ts := c.at(12345, 4000); ts != 4000 {
		t.Fatalf("after jump = %d", ts)
	}
	if ts := c.at(12345+45000, 4500); ts != 4500 {
		t.Fatalf("after re-anchor = %d", ts)
	}
}

func TestRewriteCandidate(t *testing.T) {
	c := "candidate:1 1 udp 2130706431 100.64.0.5 59000 typ host"
	got := rewriteCandidate(c, []string{"172.18.0.4", "100.64.0.5", "fd00::1"})
	if len(got) != 1 || got[0] != "candidate:1r 1 udp 2130706431 172.18.0.4 59000 typ host" {
		t.Fatalf("got %q", got)
	}
}
