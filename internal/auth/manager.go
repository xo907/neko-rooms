package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/bcrypt"

	"github.com/m1k1o/neko-rooms/internal/store"
)

const (
	CookieName      = "neko_rooms_session"
	bcryptCost      = 12
	touchInterval   = 5 * time.Minute
	cleanupInterval = time.Hour
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrAccountDisabled    = errors.New("account is disabled")
	ErrTooManyAttempts    = errors.New("too many failed attempts, try again later")

	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9._-]{1,31}$`)
)

type ctxKey int

const (
	ctxUser ctxKey = iota
	ctxSession
)

type Manager struct {
	logger zerolog.Logger
	store  *store.Store
	trust  bool // trust X-Forwarded-Proto
	path   string

	mu         sync.RWMutex
	policy     Policy
	setupToken string

	loginLimiter *limiter
	basicCache   map[string]basicEntry
	stop         chan struct{}
}

func New(s *store.Store, trustProxy bool, cookiePath string) (*Manager, error) {
	m := &Manager{
		logger:       log.With().Str("module", "auth").Logger(),
		store:        s,
		trust:        trustProxy,
		path:         cookiePath,
		loginLimiter: newLimiter(10, 15*time.Minute),
		basicCache:   map[string]basicEntry{},
		stop:         make(chan struct{}),
	}

	m.policy = DefaultPolicy()
	if err := s.GetSetting(policyKey, &m.policy); err != nil && !errors.Is(err, store.ErrNotFound) {
		return nil, err
	}

	go m.cleanupLoop()
	return m, nil
}

func (m *Manager) Shutdown() {
	close(m.stop)
}

func (m *Manager) cleanupLoop() {
	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()
	for {
		select {
		case <-m.stop:
			return
		case <-ticker.C:
			if err := m.store.DeleteExpiredSessions(); err != nil {
				m.logger.Err(err).Msg("unable to delete expired sessions")
			}
			if err := m.store.PruneAudit(m.Policy().AuditRetention); err != nil {
				m.logger.Err(err).Msg("unable to prune audit log")
			}
		}
	}
}

func (m *Manager) Store() *store.Store {
	return m.store
}

//
// policy
//

func (m *Manager) Policy() Policy {
	m.mu.RLock()
	defer m.mu.RUnlock()
	p := m.policy
	p.UserNekoImages = append([]string{}, m.policy.UserNekoImages...)
	return p
}

func (m *Manager) SetPolicy(p Policy) error {
	if err := p.Validate(); err != nil {
		return err
	}
	if err := m.store.SetSetting(policyKey, p); err != nil {
		return err
	}
	m.mu.Lock()
	m.policy = p
	m.mu.Unlock()
	return nil
}

//
// bootstrap
//

// Bootstrap makes sure there is a way to get an admin account. When the
// database has no users, an admin is created from the given credentials, or
// a one-time setup token is generated and logged.
func (m *Manager) Bootstrap(username, password string) error {
	n, err := m.store.CountUsers()
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}

	if password != "" {
		if username == "" {
			username = "admin"
		}
		hash, err := HashPassword(password)
		if err != nil {
			return err
		}
		u := &store.User{
			Username:     username,
			DisplayName:  username,
			PasswordHash: hash,
			Role:         store.RoleAdmin,
			RoomLimit:    0,
		}
		if err := m.store.CreateUser(u); err != nil {
			return err
		}
		m.logger.Info().Str("username", username).Msg("created initial admin account from configuration")
		return nil
	}

	token, err := randomToken(18)
	if err != nil {
		return err
	}

	m.mu.Lock()
	m.setupToken = token
	m.mu.Unlock()

	m.logger.Warn().Msg("no users exist yet, open the admin panel to create the first admin account")
	m.logger.Warn().Str("setup_token", token).Msg("one-time setup token")
	return nil
}

func (m *Manager) SetupRequired() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.setupToken != ""
}

func (m *Manager) CheckSetupToken(token string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.setupToken != "" && subtle.ConstantTimeCompare([]byte(token), []byte(m.setupToken)) == 1
}

func (m *Manager) FinishSetup() {
	m.mu.Lock()
	m.setupToken = ""
	m.mu.Unlock()
}

//
// passwords
//

func HashPassword(password string) (string, error) {
	// bcrypt only uses the first 72 bytes
	if len(password) > 72 {
		return "", fmt.Errorf("password must be at most 72 bytes")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	return string(hash), err
}

func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// dummyHash is used to keep timing constant for unknown usernames.
var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("neko-rooms-dummy"), bcryptCost)

func (m *Manager) ValidatePassword(password string) error {
	min := m.Policy().PasswordMinLength
	if len(password) < min {
		return fmt.Errorf("password must be at least %d characters", min)
	}
	if len(password) > 72 {
		return fmt.Errorf("password must be at most 72 bytes")
	}
	return nil
}

func ValidateUsername(username string) error {
	if !usernameRegex.MatchString(username) {
		return fmt.Errorf("username must be 2-32 characters: letters, numbers, dot, dash or underscore")
	}
	return nil
}

// Authenticate checks credentials with brute-force protection.
func (m *Manager) Authenticate(username, password, ip string) (*store.User, error) {
	key := ip + "|" + strings.ToLower(username)
	if blocked, _ := m.loginLimiter.Blocked(key); blocked {
		return nil, ErrTooManyAttempts
	}
	u, err := m.store.GetUserByUsername(username)
	if errors.Is(err, store.ErrNotFound) {
		bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		m.loginLimiter.Fail(key)
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, err
	}

	if !CheckPassword(u.PasswordHash, password) {
		m.loginLimiter.Fail(key)
		return nil, ErrInvalidCredentials
	}

	if u.Disabled {
		return nil, ErrAccountDisabled
	}

	m.loginLimiter.Reset(key)
	return u, nil
}

//
// sessions
//

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (m *Manager) sessionTTL() time.Duration {
	return time.Duration(m.Policy().SessionTTLHours) * time.Hour
}

func (m *Manager) isSecure(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return m.trust && strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}

func ClientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (m *Manager) StartSession(w http.ResponseWriter, r *http.Request, u *store.User) (*store.Session, error) {
	token, err := randomToken(32)
	if err != nil {
		return nil, err
	}

	ua := r.UserAgent()
	if len(ua) > 255 {
		ua = ua[:255]
	}

	now := time.Now().UTC()
	sess := &store.Session{
		ID:         hashToken(token),
		UserID:     u.ID,
		CreatedAt:  now,
		LastSeenAt: now,
		ExpiresAt:  now.Add(m.sessionTTL()),
		IP:         ClientIP(r),
		UserAgent:  ua,
	}
	if err := m.store.CreateSession(sess); err != nil {
		return nil, err
	}

	if err := m.store.TouchUserLogin(u.ID); err != nil {
		m.logger.Err(err).Msg("unable to update last login")
	}

	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     m.path,
		Expires:  sess.ExpiresAt,
		HttpOnly: true,
		Secure:   m.isSecure(r),
		SameSite: http.SameSiteLaxMode,
	})

	return sess, nil
}

func (m *Manager) EndSession(w http.ResponseWriter, r *http.Request) {
	if sess := SessionFromContext(r.Context()); sess != nil {
		if err := m.store.DeleteSession(sess.ID); err != nil {
			m.logger.Err(err).Msg("unable to delete session")
		}
	}

	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     m.path,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   m.isSecure(r),
		SameSite: http.SameSiteLaxMode,
	})
}

// UserFromRequest resolves the user from the session cookie, or from HTTP
// basic auth (for scripts using the API). Returns nil when not signed in.
func (m *Manager) UserFromRequest(r *http.Request) (*store.User, *store.Session) {
	if c, err := r.Cookie(CookieName); err == nil && c.Value != "" {
		sess, err := m.store.GetSession(hashToken(c.Value))
		if err == nil && time.Now().Before(sess.ExpiresAt) {
			u, err := m.store.GetUser(sess.UserID)
			if err == nil && !u.Disabled {
				// sliding expiration, throttled to limit writes
				now := time.Now().UTC()
				if now.Sub(sess.LastSeenAt) > touchInterval {
					sess.LastSeenAt = now
					sess.ExpiresAt = now.Add(m.sessionTTL())
					if err := m.store.TouchSession(sess.ID, sess.LastSeenAt, sess.ExpiresAt); err != nil {
						m.logger.Err(err).Msg("unable to touch session")
					}
				}
				return u, sess
			}
		}
	}

	if username, password, ok := r.BasicAuth(); ok {
		// bcrypt is slow on purpose, cache successful checks for a short time
		key := hashToken(username + "\x00" + password)
		m.mu.RLock()
		cached, ok := m.basicCache[key]
		m.mu.RUnlock()
		if ok && time.Now().Before(cached.expires) {
			u, err := m.store.GetUser(cached.userID)
			if err == nil && !u.Disabled && u.PasswordHash == cached.hash {
				return u, nil
			}
		}

		u, err := m.Authenticate(username, password, ClientIP(r))
		if err == nil {
			m.mu.Lock()
			if len(m.basicCache) > 1000 {
				m.basicCache = map[string]basicEntry{}
			}
			m.basicCache[key] = basicEntry{userID: u.ID, hash: u.PasswordHash, expires: time.Now().Add(time.Minute)}
			m.mu.Unlock()
			return u, nil
		}
	}

	return nil, nil
}

type basicEntry struct {
	userID  int64
	hash    string
	expires time.Time
}

func UserFromContext(ctx context.Context) *store.User {
	u, _ := ctx.Value(ctxUser).(*store.User)
	return u
}

func SessionFromContext(ctx context.Context) *store.Session {
	s, _ := ctx.Value(ctxSession).(*store.Session)
	return s
}

//
// middleware
//

// Middleware resolves the current user (if any) into the request context.
func (m *Manager) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u, sess := m.UserFromRequest(r)
		ctx := r.Context()
		if u != nil {
			ctx = context.WithValue(ctx, ctxUser, u)
		}
		if sess != nil {
			ctx = context.WithValue(ctx, ctxSession, sess)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// CSRF requires a custom header on state-changing requests authenticated by
// cookie. Browsers can not send custom headers cross-site without CORS.
func (m *Manager) CSRF(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
		default:
			if _, err := r.Cookie(CookieName); err == nil && r.Header.Get("X-Requested-With") == "" {
				http.Error(w, "missing X-Requested-With header", http.StatusForbidden)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if UserFromContext(r.Context()) == nil {
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func RequireAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := UserFromContext(r.Context())
		if u == nil {
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
		if !u.IsAdmin() {
			http.Error(w, "admin role required", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

//
// audit
//

func (m *Manager) Audit(r *http.Request, action, target, details string) {
	e := &store.AuditEntry{
		Action:  action,
		Target:  target,
		Details: details,
		IP:      ClientIP(r),
	}
	if u := UserFromContext(r.Context()); u != nil {
		e.UserID = &u.ID
		e.Username = u.Username
	}
	m.AuditEntry(e)
}

func (m *Manager) AuditEntry(e *store.AuditEntry) {
	if err := m.store.AddAudit(e); err != nil {
		m.logger.Err(err).Str("action", e.Action).Msg("unable to write audit log")
	}
}
