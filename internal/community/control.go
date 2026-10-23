package community

import (
	"context"
	"encoding/json"
	"net/url"
	"time"
)

// Owner-only control: for rooms signed in to someone's accounts, only the
// owner and admins (who join a room as its admins) may use the browser,
// everyone else watches. It is neko's locked controls setting, applied
// whenever the room starts.

// ApplyRoomControl locks or unlocks a running room's controls.
func (s *Service) ApplyRoomControl(name string, locked bool) {
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()

	entry, err := s.rooms.GetEntryByName(ctx, name)
	if err != nil || !entry.Running {
		return
	}
	settings, err := s.rooms.GetSettings(ctx, entry.ID)
	if err != nil {
		s.logger.Warn().Err(err).Str("room", name).Msg("unable to read room settings")
		return
	}
	if settings.ApiVersion != 3 {
		s.logger.Warn().Str("room", name).Int("api", settings.ApiVersion).Msg("owner-only control needs a neko v3 image")
		return
	}

	body, _ := json.Marshal(map[string]bool{"locked_controls": locked})
	if _, err := s.rooms.NekoPost(ctx, entry.ID, "/api/room/settings?token="+url.QueryEscape(settings.AdminPass), body); err != nil {
		s.logger.Warn().Err(err).Str("room", name).Msg("unable to set room controls")
		return
	}
	s.logger.Info().Str("room", name).Bool("locked", locked).Msg("room controls set")
}
