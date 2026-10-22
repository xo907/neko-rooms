package community

import (
	"strings"
	"testing"
	"time"

	"github.com/m1k1o/neko-rooms/internal/store"
)

func TestCleanTitle(t *testing.T) {
	tests := map[string]string{
		"Never Gonna Give You Up - YouTube — Mozilla Firefox": "Never Gonna Give You Up - YouTube",
		"(3) Home / X - Google Chrome":                        "Home / X",
		"Mozilla Firefox":                                     "",
		"New Tab - Chromium":                                  "",
		"  Twitch  ":                                          "Twitch",
	}
	for in, want := range tests {
		if got := cleanTitle(in); got != want {
			t.Errorf("cleanTitle(%q) = %q, want %q", in, got, want)
		}
	}
	if got := []rune(cleanTitle(strings.Repeat("a", 300))); len(got) != 120 {
		t.Errorf("long titles are cut to 120 runes, got %d", len(got))
	}
}

func TestTickets(t *testing.T) {
	s := &Service{live: newLiveHub()}
	s.live.secret = []byte("0123456789abcdef0123456789abcdef")

	tk := s.issueTicket("movie-night", "xo", true)
	got, err := s.ParseTicket("movie-night", tk)
	if err != nil || got.Name != "xo" || !got.Manager || got.ID == "" {
		t.Fatalf("ParseTicket = %+v, %v", got, err)
	}
	if _, err := s.ParseTicket("other-room", tk); err == nil {
		t.Error("a ticket works for another room")
	}
	if _, err := s.ParseTicket("movie-night", tk[:len(tk)-2]+"xx"); err == nil {
		t.Error("a forged ticket works")
	}
	if v := viewerName(&Ticket{}, "  Bob\n"); v != "Bob" {
		t.Errorf("guest name = %q", v)
	}
	if v := viewerName(&Ticket{Name: "xo"}, "someone else"); v != "xo" {
		t.Errorf("ticket name is not kept, got %q", v)
	}
}

func TestIsEmoji(t *testing.T) {
	for _, e := range append([]string{"❤️", "👍🏽"}, Reactions...) {
		if !isEmoji(e) {
			t.Errorf("%q is not an emoji", e)
		}
	}
	for _, e := range []string{"", "hi", "👍 lol", "<script>", "👍👍👍👍👍👍👍👍👍"} {
		if isEmoji(e) {
			t.Errorf("%q is an emoji", e)
		}
	}
}

func TestCleanURL(t *testing.T) {
	good := map[string]string{
		"youtube.com/watch?v=1":        "https://youtube.com/watch?v=1",
		"http://example.com/a b":       "",
		"javascript:alert(1)":          "",
		"https://twitch.tv/somebody":   "https://twitch.tv/somebody",
		"ftp://example.com/file.mp4":   "",
		"https://example.com/\x00evil": "",
	}
	for in, want := range good {
		got, err := CleanURL(in)
		if want == "" {
			if err == nil {
				t.Errorf("CleanURL(%q) = %q, want an error", in, got)
			}
			continue
		}
		if err != nil || got != want {
			t.Errorf("CleanURL(%q) = %q, %v, want %q", in, got, err, want)
		}
	}
}

func TestPartyWindow(t *testing.T) {
	now := time.Now()
	at := now.Add(time.Minute)
	meta := &store.RoomMeta{ScheduledAt: &at}
	if !partyOn(meta, now, partyWindow) {
		t.Error("a party a minute away is on")
	}
	later := now.Add(time.Hour)
	meta.ScheduledAt = &later
	if partyOn(meta, now, partyWindow) {
		t.Error("a party an hour away is on")
	}
	if canWake(&store.RoomMeta{}, now) {
		t.Error("an awake room without a party can be woken by anyone")
	}
	if !canWake(&store.RoomMeta{Sleeping: true}, now) {
		t.Error("a sleeping room cannot be woken")
	}
}
