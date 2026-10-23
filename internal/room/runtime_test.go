package room

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"

	"github.com/m1k1o/neko-rooms/internal/types"
)

func TestAllocateFrom(t *testing.T) {
	got, err := allocateFrom(nil, 59000, 59009, 1)
	if err != nil || got != (EprPorts{59000, 59000}) {
		t.Fatalf("first = %v, %v", got, err)
	}
	got, err = allocateFrom([]EprPorts{{59001, 59001}, {59000, 59000}}, 59000, 59009, 2)
	if err != nil || got != (EprPorts{59002, 59003}) {
		t.Fatalf("after two = %v, %v", got, err)
	}
	if _, err := allocateFrom([]EprPorts{{59000, 59009}}, 59000, 59009, 1); err == nil {
		t.Fatal("a full range allocated a port")
	}
	if _, err := allocateFrom(nil, 59000, 59009, 0); err == nil {
		t.Fatal("zero ports allocated")
	}
}

func TestSettingsRoundTrip(t *testing.T) {
	in := types.RoomSettings{Name: "movie-night", NekoImage: "ghcr.io/m1k1o/neko/firefox:3.1.6", ApiVersion: 3, AdminPass: "secret", DNS: []string{"1.1.1.1"}}
	enc, err := encodeSettings(in)
	if err != nil {
		t.Fatal(err)
	}
	out, err := decodeSettings(enc)
	if err != nil || out.Name != in.Name || out.AdminPass != in.AdminPass || out.ApiVersion != 3 || len(out.DNS) != 1 {
		t.Fatalf("decoded %+v, %v", out, err)
	}
	if _, err := decodeSettings("not base64!"); err == nil {
		t.Fatal("garbage decoded")
	}
}

func TestK8sAPIVersion(t *testing.T) {
	for image, want := range map[string]string{
		"ghcr.io/m1k1o/neko/firefox:3.1.6": "3",
		"ghcr.io/m1k1o/neko/firefox:2.9":   "2",
		"m1k1o/neko:firefox":               "2",
		"ghcr.io/m1k1o/neko-apps/vlc:2":    "2",
	} {
		if got := k8sAPIVersion(image)["net.m1k1o.neko.api-version"]; got != want {
			t.Errorf("%s = %s, want %s", image, got, want)
		}
	}
	if !k8sNameRegex.MatchString("neko-rooms-movie-night") || k8sNameRegex.MatchString("Movie_Night") {
		t.Error("k8s name check is wrong")
	}
	if n := k8sGPUs([]string{"count=2"}); n != 2 {
		t.Errorf("count=2 gave %d", n)
	}
	if n := k8sGPUs([]string{"all"}); n != 1 {
		t.Errorf("all gave %d", n)
	}
}

func TestPollerEvents(t *testing.T) {
	var mu sync.Mutex
	state := []types.RoomEntry{{ID: "a", Name: "a"}}
	set := func(s []types.RoomEntry) {
		mu.Lock()
		state = s
		mu.Unlock()
	}
	p := newPoller(zerolog.Nop(), func(ctx context.Context) ([]types.RoomEntry, error) {
		mu.Lock()
		defer mu.Unlock()
		return append([]types.RoomEntry{}, state...), nil
	})
	p.interval = 10 * time.Millisecond
	p.Start()
	defer p.Shutdown()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events, _ := p.Events(ctx)

	next := func() types.RoomEvent {
		select {
		case e := <-events:
			return e
		case <-time.After(2 * time.Second):
			t.Fatal("no event")
		}
		return types.RoomEvent{}
	}

	// started, then ready, then a new room, then gone
	time.Sleep(30 * time.Millisecond)
	set([]types.RoomEntry{{ID: "a", Running: true}})
	if e := next(); e.ID != "a" || e.Action != types.RoomEventStarted {
		t.Fatalf("got %+v", e)
	}
	set([]types.RoomEntry{{ID: "a", Running: true, IsReady: true}})
	if e := next(); e.Action != types.RoomEventReady {
		t.Fatalf("got %+v", e)
	}
	set([]types.RoomEntry{{ID: "a", Running: true, IsReady: true}, {ID: "b"}})
	if e := next(); e.ID != "b" || e.Action != types.RoomEventCreated {
		t.Fatalf("got %+v", e)
	}
	set([]types.RoomEntry{{ID: "b"}})
	if e := next(); e.ID != "a" || e.Action != types.RoomEventDestroyed {
		t.Fatalf("got %+v", e)
	}
}
