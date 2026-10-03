package proxy

import (
	"net/http"
	"strings"
	"testing"

	"github.com/m1k1o/neko-rooms/internal/auth"
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
