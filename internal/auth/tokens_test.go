package auth

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/m1k1o/neko-rooms/internal/store"
)

func TestAPITokens(t *testing.T) {
	s, err := store.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m, err := New(s, false, "/")
	if err != nil {
		t.Fatal(err)
	}
	u := &store.User{Username: "alice", Role: store.RoleUser}
	if err := s.CreateUser(u); err != nil {
		t.Fatal(err)
	}

	info, secret, err := m.NewAPIToken(u, "deploy script", 0)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(secret, tokenPrefix) || info.Hint != secret[len(secret)-4:] {
		t.Fatalf("token %q, hint %q", secret, info.Hint)
	}

	ask := func(header string) *store.User {
		r := httptest.NewRequest("GET", "/api/rooms", nil)
		if header != "" {
			r.Header.Set("Authorization", header)
		}
		got, _ := m.UserFromRequest(r)
		return got
	}

	if got := ask("Bearer " + secret); got == nil || got.ID != u.ID {
		t.Fatalf("bearer token resolved to %+v", got)
	}
	if got := ask("bearer " + secret); got == nil {
		t.Fatal("the scheme is case sensitive")
	}
	if got := ask("Bearer " + secret + "x"); got != nil {
		t.Fatal("a wrong token works")
	}
	if got := ask("Bearer nrk_"); got != nil {
		t.Fatal("an empty token works")
	}

	// the hash is stored, not the token
	tokens, err := s.ListAPITokens(u.ID)
	if err != nil || len(tokens) != 1 || tokens[0].LastUsedAt == nil {
		t.Fatalf("tokens = %+v, %v", tokens, err)
	}

	// expired tokens stop working
	_, old, err := m.NewAPIToken(u, "old", time.Nanosecond)
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(time.Millisecond)
	if got := ask("Bearer " + old); got != nil {
		t.Fatal("an expired token works")
	}

	// revoked tokens stop working
	if err := s.DeleteAPIToken(u.ID, info.ID); err != nil {
		t.Fatal(err)
	}
	if got := ask("Bearer " + secret); got != nil {
		t.Fatal("a revoked token works")
	}

	// disabled users lose their tokens' access
	_, live, _ := m.NewAPIToken(u, "live", 0)
	u.Disabled = true
	if err := s.UpdateUser(u); err != nil {
		t.Fatal(err)
	}
	if got := ask("Bearer " + live); got != nil {
		t.Fatal("a disabled user's token works")
	}
}
