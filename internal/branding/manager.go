package branding

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/m1k1o/neko-rooms/internal/store"
)

const settingsKey = "branding"

const MaxAssetSize = 5 << 20 // 5 MiB

var assetNameRegex = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,47}$`)

var allowedMimes = map[string]bool{
	"image/png":     true,
	"image/jpeg":    true,
	"image/gif":     true,
	"image/webp":    true,
	"image/avif":    true,
	"image/x-icon":  true,
	"image/svg+xml": true,
	"font/woff":     true,
	"font/woff2":    true,
	"font/ttf":      true,
	"font/otf":      true,
}

var ErrInvalidAsset = errors.New("invalid asset")

type Manager struct {
	mu       sync.RWMutex
	store    *store.Store
	current  Branding
	basePath string // path prefix under which /api is served, without trailing slash
}

func New(s *store.Store, basePath string) (*Manager, error) {
	m := &Manager{
		store:    s,
		basePath: strings.TrimSuffix(basePath, "/"),
	}

	b, err := m.load()
	if err != nil {
		return nil, err
	}
	m.current = b
	return m, nil
}

func (m *Manager) load() (Branding, error) {
	// unmarshal on top of defaults so new fields get default values
	b := Default()
	err := m.store.GetSetting(settingsKey, &b)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return b, err
	}
	return b, nil
}

func (m *Manager) Get() Branding {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.current
}

func (m *Manager) Set(b Branding) error {
	if err := b.Validate(); err != nil {
		return err
	}
	if err := m.store.SetSetting(settingsKey, b); err != nil {
		return err
	}

	m.mu.Lock()
	m.current = b
	m.mu.Unlock()
	return nil
}

func (m *Manager) Reset() (Branding, error) {
	b := Default()
	return b, m.Set(b)
}

// Version changes whenever branding changes, used for cache busting.
func (m *Manager) Version() string {
	b := m.Get()
	raw, _ := json.Marshal(b)
	var h uint32 = 2166136261
	for _, c := range raw {
		h ^= uint32(c)
		h *= 16777619
	}
	return fmt.Sprintf("%08x", h)
}

func (m *Manager) BasePath() string {
	return m.basePath
}

func (m *Manager) AssetURL(a *store.Asset) string {
	return fmt.Sprintf("%s/api/branding/assets/%s?v=%s", m.basePath, a.Name, a.Hash)
}

func (m *Manager) ListAssets() ([]*store.Asset, error) {
	return m.store.ListAssets()
}

func (m *Manager) GetAsset(name string) (*store.Asset, error) {
	return m.store.GetAsset(name)
}

func (m *Manager) PutAsset(name string, data []byte) (*store.Asset, error) {
	if !assetNameRegex.MatchString(name) {
		return nil, fmt.Errorf("%w: name must match %s", ErrInvalidAsset, assetNameRegex)
	}
	if len(data) == 0 || len(data) > MaxAssetSize {
		return nil, fmt.Errorf("%w: size must be between 1 byte and %d bytes", ErrInvalidAsset, MaxAssetSize)
	}

	mime := DetectMime(data)
	if !allowedMimes[mime] {
		return nil, fmt.Errorf("%w: unsupported file type %s", ErrInvalidAsset, mime)
	}

	return m.store.PutAsset(name, mime, data)
}

func (m *Manager) DeleteAsset(name string) error {
	return m.store.DeleteAsset(name)
}

func DetectMime(data []byte) string {
	mime := http.DetectContentType(data)
	if i := strings.Index(mime, ";"); i >= 0 {
		mime = mime[:i]
	}

	// svg is detected as xml or plain text
	if mime == "text/xml" || mime == "text/plain" {
		head := data
		if len(head) > 1024 {
			head = head[:1024]
		}
		if bytes.Contains(bytes.ToLower(head), []byte("<svg")) {
			return "image/svg+xml"
		}
	}

	// ico files are not always detected
	if mime == "application/octet-stream" && len(data) > 4 && bytes.Equal(data[:4], []byte{0, 0, 1, 0}) {
		return "image/x-icon"
	}

	return mime
}
