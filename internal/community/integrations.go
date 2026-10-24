package community

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/m1k1o/neko-rooms/internal/store"
)

const integrationsKey = "integrations"

// Integrations are the admin settings for room automation and Discord.
type Integrations struct {
	// absolute address of the site, used in Discord posts. Empty uses the
	// address the site was last opened on.
	PublicURL string `json:"public_url"`

	DiscordWebhook string `json:"discord_webhook"`
	NotifyLive     bool   `json:"notify_live"`     // a room started
	NotifyWatching bool   `json:"notify_watching"` // people joined an empty room
	NotifySchedule bool   `json:"notify_schedule"` // a watch party was scheduled or starts

	// stop rooms nobody is in after this many minutes, 0 = never. They wake
	// up when someone joins.
	AutoSleepMinutes int `json:"auto_sleep_minutes"`

	// wake sleeping rooms a few minutes before people usually join them
	PredictiveWake bool `json:"predictive_wake"`

	// show the title of the open page on room cards
	NowPlaying bool `json:"now_playing"`

	// minutes of rooms people watch kept for rewinding, 0 = off
	RewindMinutes int `json:"rewind_minutes"`
}

func DefaultIntegrations() Integrations {
	return Integrations{
		NotifyLive:       true,
		NotifyWatching:   true,
		NotifySchedule:   true,
		AutoSleepMinutes: 30,
		PredictiveWake:   true,
		NowPlaying:       true,
		RewindMinutes:    10,
	}
}

func (i *Integrations) Validate() error {
	i.PublicURL = strings.TrimSpace(i.PublicURL)
	if i.PublicURL != "" {
		u, err := url.Parse(i.PublicURL)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return fmt.Errorf("public address must be a http(s) url")
		}
		if !strings.HasSuffix(i.PublicURL, "/") {
			i.PublicURL += "/"
		}
	}

	i.DiscordWebhook = strings.TrimSpace(i.DiscordWebhook)
	if i.DiscordWebhook != "" {
		u, err := url.Parse(i.DiscordWebhook)
		if err != nil || u.Scheme != "https" || !strings.Contains(u.Path, "/api/webhooks/") {
			return fmt.Errorf("discord webhook must look like https://discord.com/api/webhooks/…")
		}
	}

	if i.AutoSleepMinutes < 0 || i.AutoSleepMinutes > 24*60 {
		return fmt.Errorf("sleep timer must be between 0 and 1440 minutes")
	}
	if i.AutoSleepMinutes > 0 && i.AutoSleepMinutes < 5 {
		return fmt.Errorf("sleep timer must be at least 5 minutes")
	}
	if i.RewindMinutes < 0 || i.RewindMinutes > 60 {
		return fmt.Errorf("rewind must be between 0 and 60 minutes")
	}
	return nil
}

func (s *Service) loadIntegrations() {
	i := DefaultIntegrations()
	if err := s.store.GetSetting(integrationsKey, &i); err != nil && !errors.Is(err, store.ErrNotFound) {
		s.logger.Err(err).Msg("unable to load integrations, using defaults")
	}
	s.intMu.Lock()
	s.integrations = i
	s.intMu.Unlock()
}

func (s *Service) Integrations() Integrations {
	s.intMu.RLock()
	defer s.intMu.RUnlock()
	return s.integrations
}

func (s *Service) SetIntegrations(i Integrations) error {
	if err := i.Validate(); err != nil {
		return err
	}
	if err := s.store.SetSetting(integrationsKey, i); err != nil {
		return err
	}
	s.intMu.Lock()
	s.integrations = i
	s.intMu.Unlock()
	return nil
}

// SeenBase records the address visitors open the site on, the fallback for
// links in Discord posts.
func (s *Service) SeenBase(base string) {
	s.intMu.Lock()
	s.seenBase = base
	s.intMu.Unlock()
}

// PublicBase is the absolute site address ending in a slash, or empty.
func (s *Service) PublicBase() string {
	s.intMu.RLock()
	defer s.intMu.RUnlock()
	if s.integrations.PublicURL != "" {
		return s.integrations.PublicURL
	}
	return s.seenBase
}
