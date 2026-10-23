package api

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/m1k1o/neko-rooms/internal/auth"
	"github.com/m1k1o/neko-rooms/internal/store"
)

// Upgrading a room pulls the newest version of its image and, when there is
// one, recreates the room on it with the same settings. It runs in the
// background, pulls can take minutes; clients poll for the state.

type upgradeState struct {
	State    string    `json:"state"` // pulling | recreating | done | failed
	UpToDate bool      `json:"up_to_date,omitempty"`
	Error    string    `json:"error,omitempty"`
	Room     string    `json:"room"` // the room id, new after a recreate
	Updated  time.Time `json:"updated"`
}

type upgrades struct {
	mu   sync.Mutex
	byID map[string]*upgradeState // by room name, ids change on recreate
}

func (u *upgrades) get(name string) (upgradeState, bool) {
	u.mu.Lock()
	defer u.mu.Unlock()
	s, ok := u.byID[name]
	if !ok {
		return upgradeState{}, false
	}
	return *s, true
}

func (u *upgrades) set(name string, s upgradeState) {
	u.mu.Lock()
	defer u.mu.Unlock()
	if u.byID == nil {
		u.byID = map[string]*upgradeState{}
	}
	s.Updated = time.Now()
	u.byID[name] = &s
}

// begin claims a room for an upgrade, false when one is running
func (u *upgrades) begin(name, id string) bool {
	u.mu.Lock()
	defer u.mu.Unlock()
	if s, ok := u.byID[name]; ok && (s.State == "pulling" || s.State == "recreating") {
		return false
	}
	if u.byID == nil {
		u.byID = map[string]*upgradeState{}
	}
	u.byID[name] = &upgradeState{State: "pulling", Room: id, Updated: time.Now()}
	return true
}

func (manager *ApiManagerCtx) roomUpgrade(w http.ResponseWriter, r *http.Request) {
	roomId := chi.URLParam(r, "roomId")
	entry, err := manager.rooms.GetEntry(r.Context(), roomId)
	if err != nil {
		manager.writeErr(w, err)
		return
	}
	if !manager.upgrades.begin(entry.Name, entry.ID) {
		s, _ := manager.upgrades.get(entry.Name)
		writeJSON(w, http.StatusAccepted, s)
		return
	}

	manager.auth.Audit(r, "room.upgrade", entry.Name, "image="+entry.NekoImage)
	user := auth.UserFromContext(r.Context())

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 45*time.Minute)
		defer cancel()

		fail := func(err error) {
			manager.logger.Err(err).Str("room", entry.Name).Msg("upgrade failed")
			manager.upgrades.set(entry.Name, upgradeState{State: "failed", Error: err.Error(), Room: entry.ID})
		}

		newer, err := manager.rooms.UpdateImage(ctx, entry.NekoImage)
		if err != nil {
			fail(err)
			return
		}

		// the image may also have been pulled before, by someone else
		current, err := manager.rooms.GetEntry(ctx, entry.ID)
		if err != nil {
			fail(err)
			return
		}
		if !newer && !current.IsOutdated {
			manager.upgrades.set(entry.Name, upgradeState{State: "done", UpToDate: true, Room: entry.ID})
			return
		}

		manager.upgrades.set(entry.Name, upgradeState{State: "recreating", Room: entry.ID})
		settings, err := manager.rooms.GetSettings(ctx, entry.ID)
		if err != nil {
			fail(err)
			return
		}
		res, err := manager.recreateRoom(ctx, entry.ID, entry.Name, *settings, current.Running)
		if err != nil {
			fail(err)
			return
		}

		e := &store.AuditEntry{Action: "room.upgraded", Target: res.Name, Details: "image=" + entry.NekoImage}
		if user != nil {
			e.UserID = &user.ID
			e.Username = user.Username
		}
		manager.auth.AuditEntry(e)
		manager.upgrades.set(entry.Name, upgradeState{State: "done", Room: res.ID})
	}()

	s, _ := manager.upgrades.get(entry.Name)
	writeJSON(w, http.StatusAccepted, s)
}

// roomUpgradeStatus answers by room name, since the id changes
func (manager *ApiManagerCtx) roomUpgradeStatus(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "roomName")
	u := auth.UserFromContext(r.Context())
	if !u.IsAdmin() {
		meta, err := manager.store.GetRoomMeta(name)
		if err != nil || meta.OwnerID == nil || *meta.OwnerID != u.ID {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
	}
	s, ok := manager.upgrades.get(name)
	if !ok {
		writeJSON(w, http.StatusOK, upgradeState{State: "idle"})
		return
	}
	writeJSON(w, http.StatusOK, s)
}
