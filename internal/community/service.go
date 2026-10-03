// Package community implements the public side of neko-rooms: the room
// directory on the homepage, room ownership & visibility, friends and joining.
package community

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/m1k1o/neko-rooms/internal/auth"
	"github.com/m1k1o/neko-rooms/internal/store"
	"github.com/m1k1o/neko-rooms/internal/types"
)

const (
	statsInterval     = 10 * time.Second
	statsTimeout      = 5 * time.Second
	statsConcurrency  = 4
	thumbnailTTL      = 30 * time.Second
	thumbnailFailTTL  = 10 * time.Second
	maxThumbnailCache = 200
)

type roomStats struct {
	connections int
	members     []string
	updatedAt   time.Time
}

type thumbnail struct {
	data    []byte
	expires time.Time
}

type Service struct {
	logger zerolog.Logger
	rooms  types.RoomManager
	store  *store.Store
	auth   *auth.Manager

	statsMu sync.RWMutex
	stats   map[string]roomStats // by room id

	thumbMu sync.Mutex
	thumbs  map[string]thumbnail // by room id

	ctx    context.Context
	cancel context.CancelFunc
}

func New(rooms types.RoomManager, s *store.Store, a *auth.Manager) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{
		logger: log.With().Str("module", "community").Logger(),
		rooms:  rooms,
		store:  s,
		auth:   a,
		stats:  map[string]roomStats{},
		thumbs: map[string]thumbnail{},
		ctx:    ctx,
		cancel: cancel,
	}
}

func (s *Service) Start() {
	go func() {
		ticker := time.NewTicker(statsInterval)
		defer ticker.Stop()

		s.refreshStats()
		for {
			select {
			case <-s.ctx.Done():
				return
			case <-ticker.C:
				s.refreshStats()
			}
		}
	}()
}

func (s *Service) Shutdown() {
	s.cancel()
}

// refreshStats polls member counts of all running rooms in the background,
// so that the homepage never waits for docker exec calls.
func (s *Service) refreshStats() {
	entries, err := s.rooms.List(s.ctx, nil)
	if err != nil {
		s.logger.Debug().Err(err).Msg("unable to list rooms for stats")
		return
	}

	next := map[string]roomStats{}
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, statsConcurrency)

	for _, e := range entries {
		if !e.Running || !e.IsReady {
			continue
		}

		wg.Add(1)
		sem <- struct{}{}
		go func(id string) {
			defer wg.Done()
			defer func() { <-sem }()

			ctx, cancel := context.WithTimeout(s.ctx, statsTimeout)
			defer cancel()

			st, err := s.rooms.GetStats(ctx, id)
			if err != nil {
				return
			}

			rs := roomStats{connections: int(st.Connections), updatedAt: time.Now()}
			for _, m := range st.Members {
				if m != nil && m.Name != "" {
					rs.members = append(rs.members, m.Name)
				}
			}
			// v2 does not always report connections, use member count
			if rs.connections == 0 {
				rs.connections = len(st.Members)
			}

			mu.Lock()
			next[id] = rs
			mu.Unlock()
		}(e.ID)
	}
	wg.Wait()

	s.statsMu.Lock()
	s.stats = next
	s.statsMu.Unlock()
}

func (s *Service) getStats(id string) roomStats {
	s.statsMu.RLock()
	defer s.statsMu.RUnlock()
	return s.stats[id]
}

//
// room views
//

type Room struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Category    string            `json:"category"`
	Visibility  store.Visibility  `json:"visibility"`
	Featured    bool              `json:"featured"`
	Hidden      bool              `json:"hidden,omitempty"`
	Owner       *store.PublicUser `json:"owner"`

	IsOwner   bool `json:"is_owner"`
	IsFriend  bool `json:"is_friend"` // owner is a friend of the viewer
	CanManage bool `json:"can_manage"`
	CanJoin   bool `json:"can_join"`

	Running        bool     `json:"running"`
	Ready          bool     `json:"ready"`
	Paused         bool     `json:"paused"`
	Viewers        int      `json:"viewers"`
	Members        []string `json:"members"`
	FriendsInside  []string `json:"friends_inside"`
	MaxConnections uint16   `json:"max_connections"`
	HasThumbnail   bool     `json:"has_thumbnail"`
	CreatedAt      time.Time `json:"created_at"`

	InviteCode string `json:"invite_code,omitempty"` // only for managers
}

type viewer struct {
	user      *store.User
	friends   map[int64]bool
	usernames map[string]bool // friend usernames & display names, for presence
	policy    auth.Policy
	users     map[int64]*store.User
}

func (s *Service) newViewer(u *store.User) (*viewer, error) {
	v := &viewer{
		user:      u,
		friends:   map[int64]bool{},
		usernames: map[string]bool{},
		policy:    s.auth.Policy(),
		users:     map[int64]*store.User{},
	}

	users, err := s.store.ListUsers()
	if err != nil {
		return nil, err
	}
	for _, x := range users {
		v.users[x.ID] = x
	}

	if u != nil && v.policy.FriendsEnabled {
		v.friends, err = s.store.FriendIDs(u.ID)
		if err != nil {
			return nil, err
		}
		for id := range v.friends {
			if f, ok := v.users[id]; ok {
				v.usernames[strings.ToLower(f.Username)] = true
				if f.DisplayName != "" {
					v.usernames[strings.ToLower(f.DisplayName)] = true
				}
			}
		}
	}

	return v, nil
}

func (v *viewer) isOwner(meta *store.RoomMeta) bool {
	return v.user != nil && meta != nil && meta.OwnerID != nil && *meta.OwnerID == v.user.ID
}

func (v *viewer) isFriendOfOwner(meta *store.RoomMeta) bool {
	return meta != nil && meta.OwnerID != nil && v.friends[*meta.OwnerID]
}

func (v *viewer) canManage(meta *store.RoomMeta) bool {
	return v.user.IsAdmin() || v.isOwner(meta)
}

// canList reports whether the room appears in the viewer's directory.
func (v *viewer) canList(meta *store.RoomMeta) bool {
	if meta == nil || meta.Hidden {
		return v.isOwner(meta)
	}
	if v.isOwner(meta) {
		return true
	}
	switch meta.Visibility {
	case store.VisibilityPublic:
		return v.user != nil || v.policy.GuestsCanBrowse
	case store.VisibilityFriends:
		return v.policy.FriendsEnabled && v.isFriendOfOwner(meta)
	}
	return false
}

func (v *viewer) canJoin(meta *store.RoomMeta, invite string) bool {
	if v.user == nil && v.policy.RoomsRequireLogin {
		return false
	}
	if v.canManage(meta) {
		return true
	}
	if meta == nil {
		return false
	}
	if invite != "" && meta.InviteCode != "" && invite == meta.InviteCode {
		return true
	}
	if meta.Hidden {
		return false
	}
	switch meta.Visibility {
	case store.VisibilityPublic:
		return true
	case store.VisibilityFriends:
		return v.policy.FriendsEnabled && v.isFriendOfOwner(meta)
	}
	return false
}

func (s *Service) toRoom(v *viewer, e types.RoomEntry, meta *store.RoomMeta) Room {
	r := Room{
		ID:             e.ID,
		Name:           e.Name,
		Title:          e.Name,
		Visibility:     store.VisibilityPrivate,
		Running:        e.Running,
		Ready:          e.Running && e.IsReady,
		Paused:         e.Paused,
		MaxConnections: e.MaxConnections,
		Members:        []string{},
		FriendsInside:  []string{},
		CreatedAt:      e.Created,
	}

	if meta != nil {
		if meta.Title != "" {
			r.Title = meta.Title
		}
		r.Description = meta.Description
		r.Category = meta.Category
		r.Visibility = meta.Visibility
		r.Featured = meta.Featured
		if meta.OwnerID != nil {
			if o, ok := v.users[*meta.OwnerID]; ok {
				r.Owner = o.Public()
			}
		}
	}

	r.IsOwner = v.isOwner(meta)
	r.IsFriend = v.isFriendOfOwner(meta)
	r.CanManage = v.canManage(meta)
	r.CanJoin = v.canJoin(meta, "")

	if r.CanManage && meta != nil {
		r.InviteCode = meta.InviteCode
		r.Hidden = meta.Hidden
	}

	if r.Ready {
		st := s.getStats(e.ID)
		r.Viewers = st.connections
		if v.policy.ShowMemberNames || r.CanManage {
			r.Members = append(r.Members, st.members...)
		}
		for _, m := range st.members {
			if v.usernames[strings.ToLower(m)] {
				r.FriendsInside = append(r.FriendsInside, m)
			}
		}
		r.HasThumbnail = v.policy.ThumbnailsEnabled
	}

	return r
}

// Directory returns all rooms the viewer may see, sorted for the homepage.
func (s *Service) Directory(ctx context.Context, u *store.User) ([]Room, error) {
	v, err := s.newViewer(u)
	if err != nil {
		return nil, err
	}

	entries, err := s.rooms.List(ctx, nil)
	if err != nil {
		return nil, err
	}

	metas, err := s.store.ListRoomMeta()
	if err != nil {
		return nil, err
	}

	result := []Room{}
	for _, e := range entries {
		meta := metas[e.Name]
		if !v.canList(meta) {
			continue
		}
		result = append(result, s.toRoom(v, e, meta))
	}

	sort.SliceStable(result, func(i, j int) bool {
		a, b := result[i], result[j]
		if a.Featured != b.Featured {
			return a.Featured
		}
		if a.Ready != b.Ready {
			return a.Ready
		}
		if a.Viewers != b.Viewers {
			return a.Viewers > b.Viewers
		}
		return a.CreatedAt.After(b.CreatedAt)
	})

	return result, nil
}

// AllRooms returns every room with metadata, for admins.
func (s *Service) AllRooms(ctx context.Context, u *store.User) ([]Room, error) {
	v, err := s.newViewer(u)
	if err != nil {
		return nil, err
	}

	entries, err := s.rooms.List(ctx, nil)
	if err != nil {
		return nil, err
	}

	metas, err := s.store.ListRoomMeta()
	if err != nil {
		return nil, err
	}

	result := make([]Room, 0, len(entries))
	for _, e := range entries {
		meta := metas[e.Name]
		r := s.toRoom(v, e, meta)
		if meta != nil {
			r.Hidden = meta.Hidden
		}
		result = append(result, r)
	}
	return result, nil
}

var ErrForbidden = errors.New("forbidden")

// Get returns a single room if the viewer may see it (or holds an invite).
func (s *Service) Get(ctx context.Context, u *store.User, name, invite string) (*Room, *types.RoomEntry, error) {
	entry, err := s.rooms.GetEntryByName(ctx, name)
	if err != nil {
		return nil, nil, err
	}

	meta, err := s.store.GetRoomMeta(name)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return nil, nil, err
	}
	if errors.Is(err, store.ErrNotFound) {
		meta = nil
	}

	v, err := s.newViewer(u)
	if err != nil {
		return nil, nil, err
	}

	if !v.canList(meta) && !v.canJoin(meta, invite) {
		return nil, nil, ErrForbidden
	}

	r := s.toRoom(v, *entry, meta)
	r.CanJoin = v.canJoin(meta, invite)
	return &r, entry, nil
}

// JoinURL builds the room link with credentials for the viewer.
func (s *Service) JoinURL(ctx context.Context, u *store.User, name, invite, displayName string) (string, error) {
	r, entry, err := s.Get(ctx, u, name, invite)
	if err != nil {
		return "", err
	}
	if !r.CanJoin {
		return "", ErrForbidden
	}
	if !entry.Running {
		return "", errors.New("room is not running")
	}

	settings, err := s.rooms.GetSettings(ctx, entry.ID)
	if err != nil {
		return "", err
	}

	pass := settings.UserPass
	if r.CanManage {
		pass = settings.AdminPass
	}

	q := url.Values{}
	q.Set("pwd", pass)
	if u != nil {
		name := u.DisplayName
		if name == "" {
			name = u.Username
		}
		q.Set("usr", name)
	} else if displayName = strings.TrimSpace(displayName); displayName != "" {
		if len(displayName) > 32 {
			displayName = displayName[:32]
		}
		q.Set("usr", displayName)
	}

	return entry.URL + "?" + q.Encode(), nil
}

// Thumbnail returns a cached screenshot of a room visible to the viewer.
func (s *Service) Thumbnail(ctx context.Context, u *store.User, name, invite string) ([]byte, error) {
	if !s.auth.Policy().ThumbnailsEnabled {
		return nil, ErrForbidden
	}

	r, _, err := s.Get(ctx, u, name, invite)
	if err != nil {
		return nil, err
	}
	if !r.Ready {
		return nil, types.ErrRoomNotFound
	}

	s.thumbMu.Lock()
	t, ok := s.thumbs[r.ID]
	s.thumbMu.Unlock()
	if ok && time.Now().Before(t.expires) {
		if t.data == nil {
			return nil, types.ErrRoomNotFound
		}
		return t.data, nil
	}

	ctx, cancel := context.WithTimeout(ctx, statsTimeout)
	defer cancel()

	data, err := s.rooms.GetScreenshot(ctx, r.ID)
	t = thumbnail{data: data, expires: time.Now().Add(thumbnailTTL)}
	if err != nil {
		s.logger.Debug().Err(err).Str("room", name).Msg("unable to get screenshot")
		t = thumbnail{expires: time.Now().Add(thumbnailFailTTL)}
	}

	s.thumbMu.Lock()
	if len(s.thumbs) > maxThumbnailCache {
		s.thumbs = map[string]thumbnail{}
	}
	s.thumbs[r.ID] = t
	s.thumbMu.Unlock()

	if t.data == nil {
		return nil, types.ErrRoomNotFound
	}
	return t.data, nil
}

//
// ownership
//

func NewInviteCode() string {
	b := make([]byte, 12)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// CanManageEntry checks if the user may manage the room behind an entry.
func (s *Service) CanManageEntry(u *store.User, e *types.RoomEntry) bool {
	if u.IsAdmin() {
		return true
	}
	meta, err := s.store.GetRoomMeta(e.Name)
	if err != nil {
		return false
	}
	return meta.OwnerID != nil && *meta.OwnerID == u.ID
}

// VisibleEntries filters raw room entries for the management view.
func (s *Service) VisibleEntries(u *store.User, entries []types.RoomEntry) ([]types.RoomEntry, error) {
	if u.IsAdmin() {
		return entries, nil
	}

	metas, err := s.store.ListRoomMeta()
	if err != nil {
		return nil, err
	}

	result := []types.RoomEntry{}
	for _, e := range entries {
		meta := metas[e.Name]
		if meta != nil && meta.OwnerID != nil && *meta.OwnerID == u.ID {
			result = append(result, e)
		}
	}
	return result, nil
}

func (s *Service) OwnedNames(u *store.User) (map[string]bool, error) {
	metas, err := s.store.ListRoomMeta()
	if err != nil {
		return nil, err
	}
	names := map[string]bool{}
	for name, m := range metas {
		if m.OwnerID != nil && *m.OwnerID == u.ID {
			names[name] = true
		}
	}
	return names, nil
}

func (s *Service) Store() *store.Store {
	return s.store
}
