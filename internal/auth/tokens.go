package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/m1k1o/neko-rooms/internal/store"
)

// API tokens start with this, so they are easy to spot in logs and configs
const tokenPrefix = "nrk_"

// NewAPIToken creates a token for a user, the token itself is only returned here.
func (m *Manager) NewAPIToken(u *store.User, name string, ttl time.Duration) (*store.APIToken, string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		name = "API token"
	}
	if len(name) > 60 {
		name = name[:60]
	}

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, "", err
	}
	secret := tokenPrefix + base64.RawURLEncoding.EncodeToString(b)

	t := &store.APIToken{UserID: u.ID, Name: name, Hint: secret[len(secret)-4:]}
	if ttl > 0 {
		exp := time.Now().Add(ttl).UTC()
		t.ExpiresAt = &exp
	}
	if err := m.store.CreateAPIToken(t, hashToken(secret)); err != nil {
		return nil, "", err
	}
	return t, secret, nil
}

// bearerToken reads an API token from the Authorization header
func bearerToken(r *http.Request) (string, bool) {
	h := r.Header.Get("Authorization")
	if len(h) < 7 || !strings.EqualFold(h[:7], "bearer ") {
		return "", false
	}
	t := strings.TrimSpace(h[7:])
	return t, strings.HasPrefix(t, tokenPrefix)
}

var errBadToken = errors.New("invalid or expired API token")

// userFromToken resolves the user of an API token
func (m *Manager) userFromToken(secret string) (*store.User, error) {
	t, err := m.store.GetAPITokenByHash(hashToken(secret))
	if err != nil {
		return nil, errBadToken
	}
	now := time.Now().UTC()
	if t.ExpiresAt != nil && now.After(*t.ExpiresAt) {
		return nil, errBadToken
	}
	u, err := m.store.GetUser(t.UserID)
	if err != nil || u.Disabled {
		return nil, errBadToken
	}
	// remember when it was used, throttled to limit writes
	if t.LastUsedAt == nil || now.Sub(*t.LastUsedAt) > touchInterval {
		if err := m.store.TouchAPIToken(t.ID, now); err != nil {
			m.logger.Err(err).Msg("unable to touch api token")
		}
	}
	return u, nil
}
