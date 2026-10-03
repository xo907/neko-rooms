package branding

import (
	"strings"
	"testing"

	"github.com/m1k1o/neko-rooms/internal/store"
)

func TestDefaultIsValid(t *testing.T) {
	b := Default()
	if err := b.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestValidateRejectsCSSInjection(t *testing.T) {
	cases := map[string]func(b *Branding){
		"color":       func(b *Branding) { b.Theme.Dark.Primary = "red;}body{display:none" },
		"font":        func(b *Branding) { b.Theme.FontFamily = "x;}</style><script>" },
		"url quote":   func(b *Branding) { b.Logo = `x" onerror="alert(1)` },
		"javascript":  func(b *Branding) { b.Favicon = "javascript:alert(1)" },
		"link icon":   func(b *Branding) { b.Header.Links = []Link{{Label: "x", URL: "/", Icon: "mdi-x\" onclick=\""}} },
		"theme mode":  func(b *Branding) { b.Theme.Mode = "neon" },
		"lobby color": func(b *Branding) { b.Lobby.PopupColor = "#fff;}" },
	}
	for name, mutate := range cases {
		b := Default()
		mutate(&b)
		if err := b.Validate(); err == nil {
			t.Errorf("%s: expected validation error", name)
		}
	}
}

func TestDetectMime(t *testing.T) {
	if m := DetectMime([]byte(`<?xml version="1.0"?><svg xmlns="http://www.w3.org/2000/svg"></svg>`)); m != "image/svg+xml" {
		t.Errorf("svg detected as %s", m)
	}
	if m := DetectMime([]byte{0, 0, 1, 0, 1, 0, 16, 16}); m != "image/x-icon" {
		t.Errorf("ico detected as %s", m)
	}
	if m := DetectMime([]byte("hello world")); allowedMimes[m] {
		t.Errorf("text must not be allowed, got %s", m)
	}
}

func TestBootCSSUsesPalette(t *testing.T) {
	b := Default()
	css := bootCSS(b)
	if !strings.Contains(css, b.Theme.Dark.Background) {
		t.Fatalf("boot css missing background: %s", css)
	}
}

func TestUpgradeEnablesRoomBranding(t *testing.T) {
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	// branding saved by the previous version: rooms injection off, no replace_logo key
	old := map[string]any{
		"app_name": "xo",
		"rooms":    map[string]any{"inject": false, "page_title": ""},
	}
	if err := s.SetSetting(settingsKey, old); err != nil {
		t.Fatal(err)
	}

	m, err := New(s, "")
	if err != nil {
		t.Fatal(err)
	}
	b := m.Get()
	if b.AppName != "xo" || !b.Rooms.Inject || !b.Rooms.ReplaceLogo || b.Rooms.PageTitle == "" {
		t.Fatalf("upgrade did not enable room branding: %+v", b.Rooms)
	}

	// an explicit choice made after the upgrade is kept
	b.Rooms.Inject = false
	if err := m.Set(b); err != nil {
		t.Fatal(err)
	}
	m2, _ := New(s, "")
	if m2.Get().Rooms.Inject {
		t.Fatal("explicit opt-out must be kept")
	}
}
