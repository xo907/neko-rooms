package community

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/m1k1o/neko-rooms/internal/auth"
	"github.com/m1k1o/neko-rooms/internal/store"
	"github.com/m1k1o/neko-rooms/internal/types"
)

func TestDiscordPost(t *testing.T) {
	var got discordMessage
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/webhooks/") {
			t.Errorf("posted to %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &got); err != nil {
			t.Errorf("invalid json: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	old := discordClient
	discordClient = srv.Client()
	defer func() { discordClient = old }()

	st, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	a, err := auth.New(st, false, "/")
	if err != nil {
		t.Fatal(err)
	}
	s := &Service{store: st, auth: a, live: newLiveHub(), brand: func() (string, string) { return "WatchParty", "#ff3b3b" }}

	at := time.Date(2030, 5, 1, 20, 0, 0, 0, time.UTC)
	meta := &store.RoomMeta{Name: "movie-night", Title: "Movie *night*", Category: "Movies", Visibility: store.VisibilityPublic, ScheduledAt: &at}
	msg := s.roomMessage("https://watch.example.com/", "scheduled", types.RoomEntry{Name: "movie-night"}, meta, roomStats{playing: "Interstellar"})

	if err := postDiscord(context.Background(), srv.URL+"/api/webhooks/1/abc", msg); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.Content, `Movie \*night\*`) || !strings.Contains(got.Content, fmt.Sprintf("<t:%d:F>", at.Unix())) {
		t.Errorf("content = %q", got.Content)
	}
	if len(got.Embeds) != 1 || got.Embeds[0].URL != "https://watch.example.com/r/movie-night" || got.Embeds[0].Color != 0xff3b3b {
		t.Fatalf("embed = %+v", got.Embeds)
	}
	if !strings.HasPrefix(got.Embeds[0].Image["url"], "https://watch.example.com/api/public/og/rooms/movie-night.png") {
		t.Errorf("image = %q", got.Embeds[0].Image["url"])
	}
	fields := map[string]string{}
	for _, f := range got.Embeds[0].Fields {
		fields[f.Name] = f.Value
	}
	if fields["Now playing"] != "Interstellar" || fields["Category"] != "Movies" {
		t.Errorf("fields = %v", fields)
	}
	if got.AllowedMentions == nil {
		t.Error("mentions are not turned off")
	}
}

func TestIntegrationsValidate(t *testing.T) {
	ok := Integrations{PublicURL: "https://watch.example.com", DiscordWebhook: "https://discord.com/api/webhooks/1/abc", AutoSleepMinutes: 30}
	if err := ok.Validate(); err != nil || ok.PublicURL != "https://watch.example.com/" {
		t.Fatalf("Validate = %v, %q", err, ok.PublicURL)
	}
	for _, bad := range []Integrations{
		{DiscordWebhook: "http://discord.com/api/webhooks/1/abc"},
		{DiscordWebhook: "https://example.com/hook"},
		{PublicURL: "watch.example.com"},
		{AutoSleepMinutes: 2},
		{AutoSleepMinutes: -1},
	} {
		if err := bad.Validate(); err == nil {
			t.Errorf("%+v is valid", bad)
		}
	}
}
