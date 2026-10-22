package og

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// set OG_OUT to a directory to look at the cards
func save(t *testing.T, name string, data []byte) {
	t.Helper()
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("%s is not a png: %v", name, err)
	}
	if cfg.Width != Width || cfg.Height != Height {
		t.Fatalf("%s is %dx%d", name, cfg.Width, cfg.Height)
	}
	if dir := os.Getenv("OG_OUT"); dir != "" {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// sample stands in for a screenshot or a photo
func sample(w, h int, a, b color.RGBA) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, mix(a, b, float64(x+y)/float64(w+h)))
		}
	}
	return img
}

func TestCards(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatal(err)
	}
	brand := Brand{
		AppName:    "WatchParty",
		Host:       "watch.example.com",
		Accent:     color.RGBA{0x2d, 0xd4, 0xbf, 0xff},
		Background: color.RGBA{0x06, 0x06, 0x07, 0xff},
	}

	data, err := r.SiteCard(Site{
		Brand:       brand,
		Headline:    "Watch anything, together.",
		Description: "Hop into a room, share a browser and hang out with friends in real time.",
		LiveRooms:   3,
	})
	if err != nil {
		t.Fatal(err)
	}
	save(t, "site.png", data)

	data, err = r.RoomCard(Room{
		Brand:       brand,
		Title:       "Friday movie night",
		Description: "Classic sci-fi double feature, bring snacks. We start at 9 and take requests in the chat between films.",
		Category:    "Movies",
		Live:        true,
		Thumbnail:   sample(1280, 720, color.RGBA{0x1e, 0x3a, 0x8a, 0xff}, color.RGBA{0xbe, 0x18, 0x5d, 0xff}),
		OwnerName:   "xo",
		OwnerAvatar: sample(256, 256, color.RGBA{0xf5, 0x9e, 0x0b, 0xff}, color.RGBA{0xef, 0x44, 0x44, 0xff}),
	})
	if err != nil {
		t.Fatal(err)
	}
	save(t, "room-live.png", data)

	brand.Logo = sample(128, 128, color.RGBA{0x10, 0x10, 0x10, 0xff}, color.RGBA{0x40, 0x40, 0x40, 0xff})
	data, err = r.RoomCard(Room{
		Brand:     brand,
		Title:     "A room with a really long name that has to wrap onto a second line somewhere",
		Category:  "Gaming",
		OwnerName: "Somebody",
	})
	if err != nil {
		t.Fatal(err)
	}
	save(t, "room-offline.png", data)
}

func TestWrap(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatal(err)
	}
	face := r.face(r.sans, 28)
	lines := wrap(face, "one two three four five six seven eight nine ten eleven twelve", 200, 2)
	if len(lines) != 2 {
		t.Fatalf("got %d lines", len(lines))
	}
	if last := []rune(lines[1]); last[len(last)-1] != '…' {
		t.Fatalf("last line %q has no ellipsis", lines[1])
	}
	for _, l := range lines {
		if width(face, l) > 200 {
			t.Fatalf("line %q is too wide", l)
		}
	}
	if got := wrap(face, "", 200, 2); len(got) != 0 {
		t.Fatalf("empty text gave %q", got)
	}
}
