package community

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/m1k1o/neko-rooms/internal/store"
	"github.com/m1k1o/neko-rooms/internal/types"
)

// Room automation that runs after every stats refresh: now playing, putting
// empty rooms to sleep, starting watch parties and posting to Discord.

const (
	notifyCooldown = 30 * time.Minute
	partyLead      = 2 * time.Minute  // start the room this early
	partyWindow    = 2 * time.Hour    // a party counts as on for this long
	partyAnnounce  = 10 * time.Minute // late "starting now" posts are skipped
	partyAwake     = time.Hour        // no sleeping this long after the start
)

var ErrWaking = errors.New("the room is waking up")

type activity struct {
	mu         sync.Mutex
	primed     bool                 // the first refresh only records state
	ready      map[string]bool      // by room name
	lastActive map[string]time.Time // by room name
	notified   map[string]time.Time // by event and room name
	started    map[string]int64     // room name -> start time of the party it was started for
}

func (a *activity) init() {
	if a.ready == nil {
		a.ready = map[string]bool{}
		a.lastActive = map[string]time.Time{}
		a.notified = map[string]time.Time{}
		a.started = map[string]int64{}
	}
}

//
// now playing
//

var (
	browserSuffix = regexp.MustCompile(`\s+[—–-]\s+(Mozilla Firefox|Firefox|Google Chrome|Chromium|Brave|Microsoft\x{200b}? Edge|Vivaldi|Opera|Tor Browser|Waterfox|LibreWolf)\s*$`)
	countPrefix   = regexp.MustCompile(`^\(\d+\+?\)\s+`)
)

// cleanTitle turns a browser window title into the title of the page.
func cleanTitle(t string) string {
	t = strings.TrimSpace(browserSuffix.ReplaceAllString(strings.TrimSpace(t), ""))
	t = countPrefix.ReplaceAllString(t, "")
	switch strings.ToLower(t) {
	case "", "mozilla firefox", "firefox", "google chrome", "chromium", "brave", "new tab", "new tab page",
		"about:blank", "about:newtab", "about:home", "neko", "loading…", "loading...":
		return ""
	}
	if r := []rune(t); len(r) > 120 {
		t = strings.TrimSpace(string(r[:119])) + "…"
	}
	return t
}

//
// watch parties
//

// partyOn is true from shortly before a scheduled party until a while after.
func partyOn(meta *store.RoomMeta, now time.Time, after time.Duration) bool {
	if meta == nil || meta.ScheduledAt == nil {
		return false
	}
	at := *meta.ScheduledAt
	return now.After(at.Add(-partyLead)) && now.Before(at.Add(after))
}

// canWake is true for rooms that may be started by anyone who can join.
func canWake(meta *store.RoomMeta, now time.Time) bool {
	return meta != nil && (meta.Sleeping || partyOn(meta, now, partyWindow))
}

// wake starts a stopped room that went to sleep.
func (s *Service) wake(ctx context.Context, e *types.RoomEntry) error {
	if e.Running {
		return nil
	}
	if err := s.rooms.Start(ctx, e.ID); err != nil {
		return err
	}
	s.logger.Info().Str("room", e.Name).Msg("woke up room")
	return s.store.SetRoomSleeping(e.Name, false)
}

// Wake starts a sleeping room for someone who wants to join it.
func (s *Service) Wake(ctx context.Context, u *store.User, name, invite string) error {
	r, entry, err := s.Get(ctx, u, name, invite)
	if err != nil {
		return err
	}
	if !r.CanJoin {
		return ErrForbidden
	}
	if entry.Running {
		return nil
	}
	meta, err := s.store.GetRoomMeta(name)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return err
	}
	if !canWake(meta, time.Now()) {
		return errors.New("room is not running")
	}
	return s.wake(ctx, entry)
}

// WakeByName starts a sleeping room, for visitors of its direct link.
func (s *Service) WakeByName(ctx context.Context, name string) bool {
	meta, err := s.store.GetRoomMeta(name)
	if err != nil || !canWake(meta, time.Now()) {
		return false
	}
	entry, err := s.rooms.GetEntryByName(ctx, name)
	if err != nil {
		return false
	}
	s.recordActivity(name)
	return s.wake(ctx, entry) == nil
}

//
// after each refresh
//

func (s *Service) afterRefresh(entries []types.RoomEntry, prev, next map[string]roomStats) {
	in := s.Integrations()
	metas, err := s.store.ListRoomMeta()
	if err != nil {
		s.logger.Debug().Err(err).Msg("unable to list room meta")
		return
	}

	a := &s.activity
	a.mu.Lock()
	defer a.mu.Unlock()
	a.init()

	first := !a.primed
	a.primed = true
	now := time.Now()

	// rooms that went to sleep, for the warm-up
	sleeping := map[string]func(){}

	for i := range entries {
		e := entries[i]
		meta := metas[e.Name]
		st := next[e.ID]
		was := prev[e.ID]
		ready := e.Running && e.IsReady && !e.Paused

		if st.playing != was.playing && ready {
			s.live.send(e.Name, liveEvent{Type: "playing", Title: st.playing})
		}

		// Discord
		// owner-only control is a neko setting, a started room needs it again
		if ready && !a.ready[e.Name] && meta != nil && meta.OwnerControl {
			go s.ApplyRoomControl(e.Name, true)
		}

		if !first && ready && !a.ready[e.Name] {
			if in.NotifyLive {
				s.notifyLocked(in, "live", e, meta, st)
			}
			if meta != nil && meta.StartURL != "" {
				go s.openStartPage(e, meta.StartURL)
			}
		}
		if !first && ready && a.ready[e.Name] && was.connections == 0 && st.connections > 0 && in.NotifyWatching {
			s.notifyLocked(in, "watching", e, meta, st)
		}
		a.ready[e.Name] = ready

		// sleep when empty
		if ready && in.AutoSleepMinutes > 0 {
			last, seen := a.lastActive[e.Name]
			switch {
			case st.connections > 0 || !seen:
				a.lastActive[e.Name] = now
			case now.Sub(last) > time.Duration(in.AutoSleepMinutes)*time.Minute &&
				(meta == nil || !meta.KeepAwake) && !partyOn(meta, now, partyAwake):
				delete(a.lastActive, e.Name)
				go s.sleep(e)
			}
		} else if !ready {
			delete(a.lastActive, e.Name)
		}

		if in.PredictiveWake && in.AutoSleepMinutes > 0 && !e.Running && meta != nil && meta.Sleeping {
			entry := e
			sleeping[e.Name] = func() {
				ctx, cancel := context.WithTimeout(s.ctx, time.Minute)
				defer cancel()
				if err := s.wake(ctx, &entry); err != nil {
					s.logger.Err(err).Str("room", entry.Name).Msg("unable to warm up room")
				}
			}
		}

		// watch parties start on their own
		if meta != nil && meta.ScheduledAt != nil && partyOn(meta, now, partyWindow) {
			at := *meta.ScheduledAt
			if a.started[e.Name] != at.Unix() {
				a.started[e.Name] = at.Unix()
				if !e.Running {
					go func(e types.RoomEntry) {
						ctx, cancel := context.WithTimeout(s.ctx, time.Minute)
						defer cancel()
						if err := s.wake(ctx, &e); err != nil {
							s.logger.Err(err).Str("room", e.Name).Msg("unable to start watch party")
						}
					}(e)
				}
				if in.NotifySchedule && now.Before(at.Add(partyAnnounce)) {
					s.notifyLocked(in, "party", e, meta, st)
				}
			}
		}
	}

	s.warmUp(sleeping)
}

func (s *Service) openStartPage(e types.RoomEntry, link string) {
	select {
	case <-s.ctx.Done():
		return
	case <-time.After(5 * time.Second):
	}
	ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)
	defer cancel()
	if err := s.rooms.OpenURL(ctx, e.ID, link); err != nil {
		s.logger.Warn().Err(err).Str("room", e.Name).Msg("unable to open start page")
	}
}

func (s *Service) sleep(e types.RoomEntry) {
	ctx, cancel := context.WithTimeout(s.ctx, time.Minute)
	defer cancel()
	if err := s.rooms.Stop(ctx, e.ID); err != nil {
		s.logger.Err(err).Str("room", e.Name).Msg("unable to put room to sleep")
		return
	}
	if err := s.store.SetRoomSleeping(e.Name, true); err != nil {
		s.logger.Err(err).Str("room", e.Name).Msg("unable to mark room as sleeping")
	}
	s.logger.Info().Str("room", e.Name).Msg("room is empty, put it to sleep")
}

// ScheduleChanged posts a newly scheduled watch party to Discord.
func (s *Service) ScheduleChanged(ctx context.Context, meta *store.RoomMeta) {
	in := s.Integrations()
	if !in.NotifySchedule || meta == nil || meta.ScheduledAt == nil || meta.ScheduledAt.Before(time.Now()) {
		return
	}
	entry, err := s.rooms.GetEntryByName(ctx, meta.Name)
	if err != nil {
		return
	}
	s.activity.mu.Lock()
	defer s.activity.mu.Unlock()
	s.activity.init()
	s.notifyLocked(in, "scheduled", *entry, meta, s.getStats(entry.ID))
}

//
// Discord
//

type discordField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

type discordEmbed struct {
	Title       string            `json:"title,omitempty"`
	URL         string            `json:"url,omitempty"`
	Description string            `json:"description,omitempty"`
	Color       int               `json:"color,omitempty"`
	Fields      []discordField    `json:"fields,omitempty"`
	Image       map[string]string `json:"image,omitempty"`
	Footer      map[string]string `json:"footer,omitempty"`
	Timestamp   string            `json:"timestamp,omitempty"`
}

type discordMessage struct {
	Username        string            `json:"username,omitempty"`
	Content         string            `json:"content,omitempty"`
	Embeds          []discordEmbed    `json:"embeds,omitempty"`
	AllowedMentions map[string][]bool `json:"allowed_mentions"`
}

func colorInt(hex string) int {
	v, err := strconv.ParseInt(strings.TrimPrefix(strings.TrimSpace(hex), "#"), 16, 32)
	if err != nil || len(strings.TrimPrefix(strings.TrimSpace(hex), "#")) != 6 {
		return 0x2dd4bf
	}
	return int(v)
}

// discordEscape keeps names from turning into markdown
func discordEscape(s string) string {
	return strings.NewReplacer("*", `\*`, "_", `\_`, "~", `\~`, "`", "\\`", "|", `\|`, ">", `\>`, "@", "@​").Replace(s)
}

// notifyLocked posts a room event to Discord, at most once in a while per
// room and event. Only public rooms are posted. activity.mu must be held.
func (s *Service) notifyLocked(in Integrations, kind string, e types.RoomEntry, meta *store.RoomMeta, st roomStats) {
	if in.DiscordWebhook == "" || meta == nil || meta.Hidden || meta.Visibility != store.VisibilityPublic {
		return
	}
	base := s.PublicBase()
	if base == "" {
		return
	}
	key := kind + "|" + e.Name
	if kind != "scheduled" && kind != "party" {
		if t, ok := s.activity.notified[key]; ok && time.Since(t) < notifyCooldown {
			return
		}
	}
	s.activity.notified[key] = time.Now()

	msg := s.roomMessage(base, kind, e, meta, st)
	go func() {
		ctx, cancel := context.WithTimeout(s.ctx, 15*time.Second)
		defer cancel()
		if err := postDiscord(ctx, in.DiscordWebhook, msg); err != nil {
			s.logger.Warn().Err(err).Str("room", e.Name).Str("event", kind).Msg("unable to post to discord")
		}
	}()
}

func (s *Service) roomMessage(base, kind string, e types.RoomEntry, meta *store.RoomMeta, st roomStats) discordMessage {
	appName, accent := s.brand()
	title := meta.Title
	if title == "" {
		title = e.Name
	}
	link := base + "r/" + url.PathEscape(e.Name)
	bold := "**" + discordEscape(title) + "**"

	var content string
	switch kind {
	case "live":
		content = "🔴 " + bold + " is live"
	case "watching":
		who := "Someone"
		if s.auth.Policy().ShowMemberNames && len(st.members) > 0 {
			names := st.members
			if len(names) > 3 {
				names = names[:3]
			}
			who = discordEscape(strings.Join(names, ", "))
			if len(st.members) > 3 {
				who += " and " + strconv.Itoa(len(st.members)-3) + " more"
			}
		} else if st.connections > 1 {
			who = strconv.Itoa(st.connections) + " people"
		}
		content = "👀 " + who + " started watching " + bold
	case "scheduled":
		at := meta.ScheduledAt.Unix()
		content = fmt.Sprintf("📅 Watch party scheduled: %s <t:%d:F> (<t:%d:R>)", bold, at, at)
	case "party":
		content = "🍿 " + bold + " is starting now"
	}

	embed := discordEmbed{
		Title:       title,
		URL:         link,
		Description: meta.Description,
		Color:       colorInt(accent),
		Image:       map[string]string{"url": base + "api/public/og/rooms/" + url.PathEscape(e.Name) + ".png?v=" + strconv.FormatInt(time.Now().Unix()/60, 36)},
		Footer:      map[string]string{"text": appName},
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}
	if r := []rune(embed.Description); len(r) > 300 {
		embed.Description = string(r[:299]) + "…"
	}
	if st.playing != "" {
		embed.Fields = append(embed.Fields, discordField{Name: "Now playing", Value: discordEscape(st.playing)})
	}
	if meta.ScheduledAt != nil && kind == "scheduled" {
		at := meta.ScheduledAt.Unix()
		embed.Fields = append(embed.Fields, discordField{Name: "Starts", Value: fmt.Sprintf("<t:%d:F>", at), Inline: true})
	}
	if meta.Category != "" {
		embed.Fields = append(embed.Fields, discordField{Name: "Category", Value: discordEscape(meta.Category), Inline: true})
	}
	if meta.OwnerID != nil {
		if o, err := s.store.GetUser(*meta.OwnerID); err == nil {
			name := o.DisplayName
			if name == "" {
				name = o.Username
			}
			embed.Fields = append(embed.Fields, discordField{Name: "Hosted by", Value: discordEscape(name), Inline: true})
		}
	}

	return discordMessage{
		Username:        appName,
		Content:         content,
		Embeds:          []discordEmbed{embed},
		AllowedMentions: map[string][]bool{"parse": {}},
	}
}

var discordClient = &http.Client{Timeout: 15 * time.Second}

func postDiscord(ctx context.Context, webhook string, msg discordMessage) error {
	body, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhook, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := discordClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode >= 300 {
		return fmt.Errorf("discord answered %s", res.Status)
	}
	return nil
}

// TestWebhook posts a test message with the given settings.
func (s *Service) TestWebhook(ctx context.Context, in Integrations) error {
	if err := in.Validate(); err != nil {
		return err
	}
	if in.DiscordWebhook == "" {
		return errors.New("set a discord webhook first")
	}
	appName, accent := s.brand()
	base := in.PublicURL
	if base == "" {
		base = s.PublicBase()
	}
	embed := discordEmbed{
		Title:       appName,
		Description: "Room updates will show up in this channel.",
		Color:       colorInt(accent),
	}
	if base != "" {
		embed.URL = base
		embed.Image = map[string]string{"url": base + "api/public/og/site.png?v=" + strconv.FormatInt(time.Now().Unix(), 36)}
	}
	return postDiscord(ctx, in.DiscordWebhook, discordMessage{
		Username:        appName,
		Content:         "✅ Connected to " + discordEscape(appName),
		Embeds:          []discordEmbed{embed},
		AllowedMentions: map[string][]bool{"parse": {}},
	})
}
