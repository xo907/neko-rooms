package api

import (
	"bytes"
	"context"
	"image"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/m1k1o/neko-rooms/internal/branding"
	"github.com/m1k1o/neko-rooms/internal/community"
	"github.com/m1k1o/neko-rooms/internal/og"
)

// Link previews. Everything here is what a signed-out visitor may see: chat
// apps fetch these without a session, a room that is not public only shows
// up when the link carries its invite code.

const (
	ogSiteTTL   = 5 * time.Minute
	ogRoomTTL   = time.Minute
	ogMaxCached = 100
)

type ogCache struct {
	mu    sync.Mutex
	items map[string]ogItem
}

type ogItem struct {
	data    []byte
	expires time.Time
}

func (c *ogCache) get(key string) ([]byte, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	it, ok := c.items[key]
	if !ok || time.Now().After(it.expires) {
		return nil, false
	}
	return it.data, true
}

func (c *ogCache) put(key string, data []byte, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.items == nil || len(c.items) >= ogMaxCached {
		c.items = map[string]ogItem{}
	}
	c.items[key] = ogItem{data: data, expires: time.Now().Add(ttl)}
}

func requestHost(r *http.Request) string {
	if h := r.Header.Get("X-Forwarded-Host"); h != "" {
		return strings.TrimSpace(strings.Split(h, ",")[0])
	}
	return r.Host
}

// RequestOrigin is the scheme and host the visitor used, e.g. https://example.com
func RequestOrigin(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if p := r.Header.Get("X-Forwarded-Proto"); p != "" {
		scheme = strings.TrimSpace(strings.Split(p, ",")[0])
	}
	return scheme + "://" + requestHost(r)
}

// brand collects what every card shows from the site branding.
func (manager *ApiManagerCtx) brand(r *http.Request) og.Brand {
	b := manager.branding.Get()
	brand := og.Brand{
		AppName: b.AppName,
		Host:    requestHost(r),
		Logo:    manager.brandingImage(b.Logo),
	}
	brand.Accent, _ = og.ParseColor(b.Theme.Dark.Primary)
	if brand.Accent.A == 0 {
		brand.Accent, _ = og.ParseColor("#2dd4bf")
	}
	brand.Background, _ = og.ParseColor(b.Theme.Dark.Background)
	return brand
}

// brandingImage decodes an uploaded branding asset, other urls are skipped
func (manager *ApiManagerCtx) brandingImage(u string) image.Image {
	const marker = "api/branding/assets/"
	i := strings.Index(u, marker)
	if i < 0 {
		return nil
	}
	name := u[i+len(marker):]
	if j := strings.IndexAny(name, "?#"); j >= 0 {
		name = name[:j]
	}
	a, err := manager.branding.GetAsset(name)
	if err != nil {
		return nil
	}
	img, _, err := image.Decode(bytes.NewReader(a.Data))
	if err != nil {
		return nil
	}
	return img
}

func (manager *ApiManagerCtx) liveRooms(ctx context.Context) int {
	p := manager.auth.Policy()
	if !p.HomepageEnabled || !p.GuestsCanBrowse {
		return 0
	}
	rooms, err := manager.community.Directory(ctx, nil)
	if err != nil {
		return 0
	}
	n := 0
	for _, room := range rooms {
		if room.Ready {
			n++
		}
	}
	return n
}

func (manager *ApiManagerCtx) siteCard(r *http.Request) ([]byte, error) {
	live := manager.liveRooms(r.Context())
	key := "site|" + manager.branding.Version() + "|" + requestHost(r) + "|" + strconv.Itoa(live)
	if data, ok := manager.ogCache.get(key); ok {
		return data, nil
	}

	b := manager.branding.Get()
	year := time.Now().Year()
	headline := b.Expand(b.Home.HeroTitle, year)
	if headline == "" {
		headline = b.AppName
	}
	desc := b.Expand(b.Home.HeroSubtitle, year)
	if desc == "" {
		desc = b.Description
	}

	data, err := manager.og.SiteCard(og.Site{
		Brand:       manager.brand(r),
		Headline:    headline,
		Description: desc,
		LiveRooms:   live,
	})
	if err != nil {
		return nil, err
	}
	manager.ogCache.put(key, data, ogSiteTTL)
	return data, nil
}

func (manager *ApiManagerCtx) roomCard(r *http.Request, room *community.Room, invite string) ([]byte, error) {
	startsIn := ""
	if room.ScheduledAt != nil && !room.Ready {
		startsIn = StartsIn(*room.ScheduledAt, time.Now())
	}
	key := "room|" + room.Name + "|" + invite + "|" + manager.branding.Version() + "|" + requestHost(r) +
		"|" + room.NowPlaying + "|" + startsIn + "|" + strconv.FormatBool(room.Sleeping)
	if data, ok := manager.ogCache.get(key); ok {
		return data, nil
	}

	card := og.Room{
		Brand:       manager.brand(r),
		Title:       room.Title,
		Description: room.Description,
		Category:    room.Category,
		Live:        room.Ready,
		NowPlaying:  room.NowPlaying,
		Sleeping:    room.Sleeping,
		StartsIn:    startsIn,
	}
	if room.Ready {
		if data, err := manager.community.Thumbnail(r.Context(), nil, room.Name, invite); err == nil {
			card.Thumbnail, _, _ = image.Decode(bytes.NewReader(data))
		}
	}
	if o := room.Owner; o != nil {
		card.OwnerName = o.DisplayName
		if card.OwnerName == "" {
			card.OwnerName = o.Username
		}
		if o.Avatar > 0 {
			if a, err := manager.store.GetAvatar(o.ID); err == nil {
				card.OwnerAvatar, _, _ = image.Decode(bytes.NewReader(a.Data))
			}
		}
	}

	data, err := manager.og.RoomCard(card)
	if err != nil {
		return nil, err
	}
	manager.ogCache.put(key, data, ogRoomTTL)
	return data, nil
}

// shareRoom is the room as a signed-out visitor with the link sees it
func (manager *ApiManagerCtx) shareRoom(ctx context.Context, name, invite string) *community.Room {
	room, _, err := manager.community.Get(ctx, nil, name, invite)
	if err != nil {
		return nil
	}
	return room
}

func writePNG(w http.ResponseWriter, data []byte, maxAge int) {
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age="+strconv.Itoa(maxAge))
	w.Header().Set("Content-Length", strconv.Itoa(len(data)))
	w.Write(data)
}

func (manager *ApiManagerCtx) ogSiteImage(w http.ResponseWriter, r *http.Request) {
	if manager.og == nil {
		http.NotFound(w, r)
		return
	}
	data, err := manager.siteCard(r)
	if err != nil {
		manager.logger.Err(err).Msg("unable to draw site card")
		http.Error(w, "unable to draw image", http.StatusInternalServerError)
		return
	}
	writePNG(w, data, 300)
}

func (manager *ApiManagerCtx) ogRoomImage(w http.ResponseWriter, r *http.Request) {
	if manager.og == nil {
		http.NotFound(w, r)
		return
	}
	name := strings.TrimSuffix(chi.URLParam(r, "roomName"), ".png")
	invite := r.URL.Query().Get("invite")

	var data []byte
	var err error
	if room := manager.shareRoom(r.Context(), name, invite); room != nil {
		data, err = manager.roomCard(r, room, invite)
	} else {
		// nothing about a room that is not shared
		data, err = manager.siteCard(r)
	}
	if err != nil {
		manager.logger.Err(err).Msg("unable to draw room card")
		http.Error(w, "unable to draw image", http.StatusInternalServerError)
		return
	}
	writePNG(w, data, 60)
}

// ogOEmbed adds the host and category line above the title in Discord
func (manager *ApiManagerCtx) ogOEmbed(w http.ResponseWriter, r *http.Request) {
	b := manager.branding.Get()
	base := r.URL.Query().Get("base")
	doc := map[string]string{
		"version":       "1.0",
		"type":          "link",
		"provider_name": b.AppName,
	}
	if u, err := url.Parse(base); err == nil && (u.Scheme == "http" || u.Scheme == "https") {
		doc["provider_url"] = base
	}

	if name := r.URL.Query().Get("room"); name != "" {
		if room := manager.shareRoom(r.Context(), name, r.URL.Query().Get("invite")); room != nil {
			parts := []string{}
			if room.Owner != nil {
				owner := room.Owner.DisplayName
				if owner == "" {
					owner = room.Owner.Username
				}
				parts = append(parts, "Hosted by "+owner)
			}
			if room.Category != "" {
				parts = append(parts, room.Category)
			}
			if len(parts) > 0 {
				doc["author_name"] = strings.Join(parts, " · ")
			}
			doc["title"] = room.Title
		}
	}

	writeJSON(w, http.StatusOK, doc)
}

// PageMeta is the link preview of the homepage, or of a room when name is
// set. base is the absolute url of the site root, ending in a slash.
func (manager *ApiManagerCtx) PageMeta(r *http.Request, base, name, invite string) *branding.PageMeta {
	if manager.og == nil {
		return nil
	}
	b := manager.branding.Get()
	year := time.Now().Year()
	api := base + "api/public/og/"

	// links in Discord posts use the address admins open the site on
	if u, _ := manager.auth.UserFromRequest(r); u.IsAdmin() {
		manager.community.SeenBase(base)
	}

	site := &branding.PageMeta{
		Title:       b.Expand(b.PageTitle, year),
		Description: b.Description,
		URL:         base,
		Image:       api + "site.png?v=" + manager.branding.Version(),
		ImageAlt:    b.AppName,
		SiteName:    b.AppName,
		OEmbed:      api + "oembed.json?" + url.Values{"base": {base}}.Encode(),
	}
	if name == "" {
		return site
	}

	room := manager.shareRoom(r.Context(), name, invite)
	if room == nil {
		return site
	}

	q := url.Values{}
	if invite != "" {
		q.Set("invite", invite)
	}
	link := base + "r/" + url.PathEscape(room.Name)
	if len(q) > 0 {
		link += "?" + q.Encode()
	}

	desc := room.Description
	if desc == "" {
		desc = "Watch together in " + room.Title + " on " + b.AppName + "."
	}
	if room.Ready {
		if room.NowPlaying != "" {
			desc = "▶ " + room.NowPlaying + "\n" + desc
		}
	} else if room.ScheduledAt != nil {
		status := "📅 Watch party " + StartsIn(*room.ScheduledAt, time.Now())
		if room.RSVPs > 0 {
			status += " · " + strconv.Itoa(room.RSVPs) + " going"
		}
		desc = status + "\n" + desc
	} else if room.Sleeping {
		desc = "💤 Sleeping, joining wakes it up\n" + desc
	}

	// a new image url every few minutes, chat apps keep images they have seen
	img := url.Values{"v": {strconv.FormatInt(time.Now().Unix()/300, 36)}}
	if invite != "" {
		img.Set("invite", invite)
	}
	oembed := url.Values{"base": {base}, "room": {room.Name}}
	if invite != "" {
		oembed.Set("invite", invite)
	}

	return &branding.PageMeta{
		DocTitle:    room.Title + " · " + b.AppName,
		Title:       room.Title,
		Description: desc,
		URL:         link,
		Image:       api + "rooms/" + url.PathEscape(room.Name) + ".png?" + img.Encode(),
		ImageAlt:    room.Title,
		SiteName:    b.AppName,
		OEmbed:      api + "oembed.json?" + oembed.Encode(),
	}
}

// StartsIn says when a watch party starts, e.g. "in 2 hours".
func StartsIn(at, now time.Time) string {
	d := at.Sub(now)
	plural := func(n int, unit string) string {
		if n == 1 {
			return "in 1 " + unit
		}
		return "in " + strconv.Itoa(n) + " " + unit + "s"
	}
	switch {
	case d < time.Minute:
		return "starting now"
	case d < time.Hour:
		return plural(int(d.Minutes()+0.5), "minute")
	case d < 36*time.Hour:
		return plural(int(d.Hours()+0.5), "hour")
	default:
		return plural(int(d.Hours()/24+0.5), "day")
	}
}
