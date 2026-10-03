package proxy

import (
	"net/http"
	"strings"
	"testing"

	"github.com/m1k1o/neko-rooms/internal/auth"
	"github.com/m1k1o/neko-rooms/internal/branding"
	"github.com/m1k1o/neko-rooms/internal/store"
)

func TestStripSessionCookie(t *testing.T) {
	r, _ := http.NewRequest("GET", "/", nil)
	r.AddCookie(&http.Cookie{Name: auth.CookieName, Value: "secret"})
	r.AddCookie(&http.Cookie{Name: "neko", Value: "keep"})

	stripSessionCookie(r)

	cookies := r.Header.Get("Cookie")
	if strings.Contains(cookies, "secret") {
		t.Fatal("session cookie must not reach rooms")
	}
	if !strings.Contains(cookies, "neko=keep") {
		t.Fatal("other cookies must be kept")
	}
}

func TestInjectHTML(t *testing.T) {
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	bm, err := branding.New(s, "")
	if err != nil {
		t.Fatal(err)
	}
	b := bm.Get()
	b.AppName = "xo"
	b.Logo = "/api/branding/assets/logo?v=1"
	if err := bm.Set(b); err != nil {
		t.Fatal(err)
	}

	p := &ProxyManagerCtx{branding: bm, loginURL: "/"}
	page := `<!DOCTYPE html><html><head><title>n.eko</title><link rel="icon" href="favicon.ico"></head><body><div id="neko"></div></body></html>`
	out := string(p.injectHTML([]byte(page), "Movie night"))

	for _, want := range []string{
		"<title>Movie night · xo</title>",
		b.Theme.Dark.Background + " !important",
		`"name":"xo"`,
		`"logo":"/api/branding/assets/logo?v=1"`,
		"data-nr-branded",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in output", want)
		}
	}
	if strings.Index(out, "<style>") > strings.Index(out, "</head>") {
		t.Error("styles must be in <head>")
	}
	if strings.Index(out, "<script>") > strings.Index(out, "</body>") {
		t.Error("scripts must be before </body>")
	}

	// disabled injection leaves the page alone
	b.Rooms.ReplaceLogo = false
	b.Rooms.SiteColors = false
	b.Rooms.PageTitle = ""
	bm.Set(b)
	out = string(p.injectHTML([]byte(page), "Movie night"))
	if strings.Contains(out, "data-nr-branded") || strings.Contains(out, "!important") || !strings.Contains(out, "<title>n.eko</title>") {
		t.Error("expected no logo, color or title changes")
	}
}
