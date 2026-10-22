// Package og draws the link preview images (Open Graph) that chat apps like
// Discord show when a link to the site or to a room is shared.
package og

import (
	"bytes"
	"embed"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"

	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	// decoders for thumbnails, avatars and logos
	_ "image/gif"
	_ "image/jpeg"

	_ "golang.org/x/image/webp"
)

// Width and Height are the size of the cards, the size chat apps expect.
const (
	Width  = 1200
	Height = 630
)

//go:embed fonts/*.ttf
var fontFiles embed.FS

// Renderer holds the parsed fonts. Faces are made per card, they are not
// safe to share between goroutines.
type Renderer struct {
	serif *opentype.Font
	sans  *opentype.Font
	bold  *opentype.Font
}

func New() (*Renderer, error) {
	load := func(name string) (*opentype.Font, error) {
		data, err := fontFiles.ReadFile("fonts/" + name)
		if err != nil {
			return nil, err
		}
		return opentype.Parse(data)
	}

	var r Renderer
	var err error
	if r.serif, err = load("InstrumentSerif-Regular.ttf"); err != nil {
		return nil, err
	}
	if r.sans, err = load("Geist-Regular.ttf"); err != nil {
		return nil, err
	}
	if r.bold, err = load("Geist-SemiBold.ttf"); err != nil {
		return nil, err
	}
	return &r, nil
}

// Brand is what every card shares.
type Brand struct {
	AppName    string
	Host       string
	Logo       image.Image // optional
	Accent     color.RGBA
	Background color.RGBA
}

type Site struct {
	Brand
	Headline    string
	Description string
	LiveRooms   int
}

type Room struct {
	Brand
	Title       string
	Description string
	Category    string
	Live        bool
	Viewers     int
	Thumbnail   image.Image // optional
	OwnerName   string      // optional
	OwnerAvatar image.Image // optional
	NowPlaying  string      // optional
	Sleeping    bool
	StartsIn    string // optional, e.g. "in 2 hours"
}

var (
	white = color.RGBA{0xfa, 0xfa, 0xfa, 0xff}
	muted = color.RGBA{0xa1, 0xa1, 0xaa, 0xff}
	soft  = color.RGBA{0xd4, 0xd4, 0xd8, 0xff}
	live  = color.RGBA{0x34, 0xd3, 0x99, 0xff}
)

const pad = 72

// SiteCard is the preview of the homepage.
func (r *Renderer) SiteCard(s Site) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, Width, Height))
	bg := darken(s.Background)
	fill(img, bg)
	glow(img, 1060, -60, 760, s.Accent, 0.30)
	glow(img, -120, 720, 620, s.Accent, 0.10)
	dots(img, image.Rect(560, 0, Width, Height), 1100, 40)

	r.header(img, s.Brand, 52, 30)

	headFace := r.face(r.serif, 112)
	descFace := r.face(r.sans, 34)
	head := wrap(headFace, s.Headline, Width-2*pad, 2)
	desc := wrap(descFace, s.Description, 960, 2)

	// laid out from the footer up, so any number of lines fits
	footerTop := Height - pad - 46
	descLast := footerTop - 58
	descFirst := descLast - (len(desc)-1)*46
	headLast := descFirst - 72
	if len(desc) == 0 {
		headLast = footerTop - 64
	}
	headFirst := headLast - (len(head)-1)*104

	for i, line := range head {
		text(img, headFace, pad, headFirst+i*104, white, line)
	}
	for i, line := range desc {
		text(img, descFace, pad, descFirst+i*46, muted, line)
	}

	x := pad
	if s.Host != "" {
		x += pill(img, r.face(r.sans, 24), x, footerTop, 46, s.Host, soft, glass(bg, 0.07)) + 16
	}
	if s.LiveRooms > 0 {
		label := "1 room live now"
		if s.LiveRooms > 1 {
			label = strconv.Itoa(s.LiveRooms) + " rooms live now"
		}
		livePill(img, r.face(r.bold, 22), x, footerTop, 46, label, bg)
	}

	return encode(img)
}

// RoomCard is the preview of a room.
func (r *Renderer) RoomCard(c Room) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, Width, Height))
	bg := darken(c.Background)
	fill(img, bg)

	// the picture of what is playing fills the right side and fades into the page
	art := image.Rect(600, 0, Width, Height)
	if c.Thumbnail != nil {
		cover(img, art, c.Thumbnail)
		shade(img, art, color.RGBA{0, 0, 0, 0xff}, func(x, y int) float64 { return 0.3 })
	} else {
		glow(img, 980, 300, 560, c.Accent, 0.34)
		dots(img, art, 900, 36)
	}
	shade(img, art, bg, func(x, y int) float64 {
		t := float64(x-art.Min.X) / 380
		return 1 - smoothstep(t)
	})
	shade(img, art, bg, func(x, y int) float64 {
		t := float64(Height-y) / 160
		return 0.6 * (1 - smoothstep(t))
	})
	glow(img, -80, -80, 520, c.Accent, 0.14)

	r.header(img, c.Brand, 40, 26)

	// category and status
	pillTop := 148
	x := pad
	if c.Category != "" {
		x += pill(img, r.face(r.bold, 22), x, pillTop, 42, c.Category, tint(c.Accent), mix(bg, c.Accent, 0.18)) + 12
	}
	if c.Live {
		label := "Live"
		if c.Viewers == 1 {
			label = "Live · 1 watching"
		} else if c.Viewers > 1 {
			label = "Live · " + strconv.Itoa(c.Viewers) + " watching"
		}
		livePill(img, r.face(r.bold, 22), x, pillTop, 42, label, bg)
	} else if c.StartsIn != "" {
		pill(img, r.face(r.bold, 22), x, pillTop, 42, "Watch party "+c.StartsIn, tint(c.Accent), mix(bg, c.Accent, 0.18))
	} else if c.Sleeping {
		pill(img, r.face(r.sans, 22), x, pillTop, 42, "Sleeping · join to wake it", soft, glass(bg, 0.07))
	} else {
		pill(img, r.face(r.sans, 22), x, pillTop, 42, "Offline", muted, glass(bg, 0.07))
	}

	titleFace := r.face(r.serif, 92)
	title := wrap(titleFace, c.Title, 640, 2)
	titleFirst := pillTop + 42 + 100
	for i, line := range title {
		text(img, titleFace, pad, titleFirst+i*88, white, line)
	}

	descFace := r.face(r.sans, 28)
	descLines := 3
	if len(title) > 1 {
		descLines = 2
	}
	descFirst := titleFirst + (len(title)-1)*88 + 58

	// what is on, above the description
	if c.NowPlaying != "" && c.Live {
		label := "Now playing  "
		lf := r.face(r.sans, 26)
		text(img, lf, pad, descFirst, muted, label)
		w := width(lf, label)
		pf := r.face(r.bold, 26)
		text(img, pf, pad+w, descFirst, white, ellipsize(pf, c.NowPlaying, 600-w))
		descFirst += 50
		descLines--
	}

	desc := wrap(descFace, c.Description, 600, descLines)
	for i, line := range desc {
		text(img, descFace, pad, descFirst+i*40, muted, line)
	}

	// who made it
	if c.OwnerName != "" {
		size := 60
		top := Height - pad - size
		avatar(img, r.face(r.bold, 26), pad, top, size, c.OwnerAvatar, c.OwnerName, c.Accent)
		text(img, r.face(r.sans, 20), pad+size+18, top+24, muted, "Hosted by")
		text(img, r.face(r.bold, 28), pad+size+18, top+56, white, ellipsize(r.face(r.bold, 28), c.OwnerName, 420))
	} else if c.Host != "" {
		pill(img, r.face(r.sans, 24), pad, Height-pad-46, 46, c.Host, soft, glass(bg, 0.07))
	}

	return encode(img)
}

// header draws the logo and the site name in the top left corner.
func (r *Renderer) header(img *image.RGBA, b Brand, logoSize int, fontSize float64) {
	face := r.face(r.bold, fontSize)
	x := pad
	top := pad - 8
	if b.Logo != nil {
		logo(img, x, top, logoSize, b.Logo)
		x += logoSize + 18
	} else {
		d := logoSize / 3
		circle(img, x, top+(logoSize-d)/2, d, b.Accent)
		x += d + 14
	}
	text(img, face, x, top+(logoSize+capHeight(face))/2, white, b.AppName)
}

func (r *Renderer) face(f *opentype.Font, size float64) font.Face {
	face, err := opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		// only fails for bad options, which are constants here
		panic(err)
	}
	return face
}

//
// drawing
//

func encode(img image.Image) ([]byte, error) {
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := enc.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func fill(img *image.RGBA, c color.Color) {
	draw.Draw(img, img.Bounds(), image.NewUniform(c), image.Point{}, draw.Src)
}

// shade paints c over r with a per pixel strength between 0 and 1
func shade(img *image.RGBA, r image.Rectangle, c color.RGBA, alpha func(x, y int) float64) {
	r = r.Intersect(img.Bounds())
	mask := image.NewAlpha(r)
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			mask.SetAlpha(x, y, color.Alpha{A: unit(alpha(x, y))})
		}
	}
	draw.DrawMask(img, r, image.NewUniform(c), image.Point{}, mask, r.Min, draw.Over)
}

// glow is a soft round light
func glow(img *image.RGBA, cx, cy, radius float64, c color.RGBA, strength float64) {
	r := image.Rect(int(cx-radius), int(cy-radius), int(cx+radius)+1, int(cy+radius)+1)
	shade(img, r, c, func(x, y int) float64 {
		d := math.Hypot(float64(x)-cx, float64(y)-cy) / radius
		if d >= 1 {
			return 0
		}
		t := 1 - d
		return t * t * strength
	})
}

// dots is a faint grid that fades away from the top right corner
func dots(img *image.RGBA, r image.Rectangle, reach float64, step int) {
	dot := color.RGBA{0xff, 0xff, 0xff, 0xff}
	for y := r.Min.Y + step/2; y < r.Max.Y; y += step {
		for x := r.Min.X + step/2; x < r.Max.X; x += step {
			d := math.Hypot(float64(Width-x), float64(y)) / reach
			if d >= 1 {
				continue
			}
			a := 0.09 * (1 - d)
			m := image.NewUniform(color.Alpha{A: unit(a)})
			draw.DrawMask(img, image.Rect(x, y, x+2, y+2), image.NewUniform(dot), image.Point{}, m, image.Point{}, draw.Over)
		}
	}
}

// roundMask is an anti-aliased rounded rectangle, a circle when radius is half the size
func roundMask(w, h int, radius float64) *image.Alpha {
	m := image.NewAlpha(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			px, py := float64(x)+0.5, float64(y)+0.5
			dx := math.Max(math.Max(radius-px, px-(float64(w)-radius)), 0)
			dy := math.Max(math.Max(radius-py, py-(float64(h)-radius)), 0)
			m.SetAlpha(x, y, color.Alpha{A: unit(radius - math.Hypot(dx, dy) + 0.5)})
		}
	}
	return m
}

func roundRect(img *image.RGBA, x, y, w, h int, radius float64, c color.Color) {
	draw.DrawMask(img, image.Rect(x, y, x+w, y+h), image.NewUniform(c), image.Point{}, roundMask(w, h, radius), image.Point{}, draw.Over)
}

func circle(img *image.RGBA, x, y, size int, c color.Color) {
	roundRect(img, x, y, size, size, float64(size)/2, c)
}

// cover scales src to fill r, cropping what does not fit
func cover(img *image.RGBA, r image.Rectangle, src image.Image) {
	sb := src.Bounds()
	if sb.Empty() {
		return
	}
	scale := math.Max(float64(r.Dx())/float64(sb.Dx()), float64(r.Dy())/float64(sb.Dy()))
	sw := int(float64(r.Dx()) / scale)
	sh := int(float64(r.Dy()) / scale)
	sx := sb.Min.X + (sb.Dx()-sw)/2
	sy := sb.Min.Y + (sb.Dy()-sh)/2
	xdraw.CatmullRom.Scale(img, r, src, image.Rect(sx, sy, sx+sw, sy+sh), xdraw.Over, nil)
}

// masked draws src scaled to a square, cut by a rounded mask
func masked(img *image.RGBA, x, y, size int, src image.Image, radius float64) {
	tmp := image.NewRGBA(image.Rect(0, 0, size, size))
	cover(tmp, tmp.Bounds(), src)
	draw.DrawMask(img, image.Rect(x, y, x+size, y+size), tmp, image.Point{}, roundMask(size, size, radius), image.Point{}, draw.Over)
}

func logo(img *image.RGBA, x, y, size int, src image.Image) {
	masked(img, x, y, size, src, float64(size)*0.24)
}

// avatar is the photo in a circle, or the first letter of the name
func avatar(img *image.RGBA, face font.Face, x, y, size int, src image.Image, name string, accent color.RGBA) {
	if src != nil {
		masked(img, x, y, size, src, float64(size)/2)
		return
	}
	circle(img, x, y, size, mix(color.RGBA{0x18, 0x18, 0x1b, 0xff}, accent, 0.35))
	r, _ := utf8.DecodeRuneInString(strings.ToUpper(name))
	letter := string(r)
	w := font.MeasureString(face, letter).Round()
	text(img, face, x+(size-w)/2, y+(size+capHeight(face))/2, white, letter)
}

// pill is a rounded label, it returns its width
func pill(img *image.RGBA, face font.Face, x, y, h int, label string, fg, bg color.RGBA) int {
	padX := h * 11 / 24
	w := font.MeasureString(face, label).Round() + 2*padX
	roundRect(img, x, y, w, h, float64(h)/2, bg)
	text(img, face, x+padX, y+(h+capHeight(face))/2, fg, label)
	return w
}

// livePill is a pill with a green dot
func livePill(img *image.RGBA, face font.Face, x, y, h int, label string, bg color.RGBA) int {
	padX := h * 11 / 24
	d := h / 4
	w := font.MeasureString(face, label).Round() + 2*padX + d + 10
	roundRect(img, x, y, w, h, float64(h)/2, mix(bg, live, 0.16))
	circle(img, x+padX, y+(h-d)/2, d, live)
	text(img, face, x+padX+d+10, y+(h+capHeight(face))/2, live, label)
	return w
}

//
// text
//

func text(img draw.Image, face font.Face, x, y int, c color.Color, s string) {
	d := &font.Drawer{Dst: img, Src: image.NewUniform(c), Face: face, Dot: fixed.P(x, y)}
	d.DrawString(s)
}

func capHeight(face font.Face) int {
	m := face.Metrics()
	if m.CapHeight > 0 {
		return m.CapHeight.Round()
	}
	return (m.Ascent * 7 / 10).Round()
}

func width(face font.Face, s string) int {
	return font.MeasureString(face, s).Round()
}

// ellipsize shortens s to fit max pixels
func ellipsize(face font.Face, s string, max int) string {
	if width(face, s) <= max {
		return s
	}
	r := []rune(s)
	for len(r) > 0 && width(face, string(r)+"…") > max {
		r = r[:len(r)-1]
	}
	return strings.TrimRight(string(r), " ") + "…"
}

// wrap breaks s into at most n lines of max pixels, the last one ellipsized
func wrap(face font.Face, s string, max, n int) []string {
	lines := []string{}
	cur := ""
	for _, word := range strings.Fields(s) {
		next := word
		if cur != "" {
			next = cur + " " + word
		}
		if width(face, next) <= max {
			cur = next
			continue
		}
		if cur != "" {
			lines = append(lines, cur)
		}
		cur = word
		// a word longer than a line is cut
		for width(face, cur) > max {
			r := []rune(cur)
			i := len(r)
			for i > 1 && width(face, string(r[:i])) > max {
				i--
			}
			lines = append(lines, string(r[:i]))
			cur = string(r[i:])
		}
	}
	if cur != "" {
		lines = append(lines, cur)
	}
	if len(lines) > n {
		// there is more text, the last line ends in an ellipsis
		lines = lines[:n]
		r := []rune(lines[n-1])
		for len(r) > 0 && width(face, string(r)+"…") > max {
			r = r[:len(r)-1]
		}
		last := string(r)
		// end on a whole word when that does not waste much of the line
		if i := strings.LastIndex(last, " "); i > len(last)*2/3 {
			last = last[:i]
		}
		lines[n-1] = strings.TrimRight(last, " ,.;:-") + "…"
	}
	return lines
}

//
// color
//

func unit(a float64) uint8 {
	if a <= 0 {
		return 0
	}
	if a >= 1 {
		return 0xff
	}
	return uint8(a*255 + 0.5)
}

func smoothstep(t float64) float64 {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	return t * t * (3 - 2*t)
}

func mix(a, b color.RGBA, t float64) color.RGBA {
	l := func(x, y uint8) uint8 { return uint8(float64(x)*(1-t) + float64(y)*t + 0.5) }
	return color.RGBA{l(a.R, b.R), l(a.G, b.G), l(a.B, b.B), 0xff}
}

// glass is a faint white layer over the background
func glass(bg color.RGBA, t float64) color.RGBA {
	return mix(bg, color.RGBA{0xff, 0xff, 0xff, 0xff}, t)
}

// tint is a lighter accent that reads on a dark background
func tint(c color.RGBA) color.RGBA {
	return mix(c, color.RGBA{0xff, 0xff, 0xff, 0xff}, 0.25)
}

func luminance(c color.RGBA) float64 {
	return (0.2126*float64(c.R) + 0.7152*float64(c.G) + 0.0722*float64(c.B)) / 255
}

// darken keeps cards dark even when the site uses a light background
func darken(c color.RGBA) color.RGBA {
	if c.A == 0 || luminance(c) > 0.2 {
		return color.RGBA{0x09, 0x09, 0x0b, 0xff}
	}
	return c
}

// ParseColor reads #rgb or #rrggbb, ok is false for anything else.
func ParseColor(s string) (color.RGBA, bool) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) == 3 {
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	}
	if len(s) != 6 {
		return color.RGBA{}, false
	}
	var v [3]uint8
	for i := 0; i < 3; i++ {
		var b uint8
		for _, ch := range s[i*2 : i*2+2] {
			b <<= 4
			switch {
			case ch >= '0' && ch <= '9':
				b |= uint8(ch - '0')
			case ch >= 'a' && ch <= 'f':
				b |= uint8(ch-'a') + 10
			case ch >= 'A' && ch <= 'F':
				b |= uint8(ch-'A') + 10
			default:
				return color.RGBA{}, false
			}
		}
		v[i] = b
	}
	return color.RGBA{v[0], v[1], v[2], 0xff}, true
}
