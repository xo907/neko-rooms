package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/m1k1o/neko-rooms/internal/auth"
	"github.com/m1k1o/neko-rooms/internal/community"
	"github.com/m1k1o/neko-rooms/internal/store"
)

//
// public directory
//

func (manager *ApiManagerCtx) publicRooms(w http.ResponseWriter, r *http.Request) {
	u := auth.UserFromContext(r.Context())
	p := manager.auth.Policy()
	if !p.HomepageEnabled && !u.IsAdmin() {
		http.Error(w, "homepage is disabled", http.StatusForbidden)
		return
	}
	if u == nil && !p.GuestsCanBrowse {
		writeJSON(w, http.StatusOK, []community.Room{})
		return
	}

	rooms, err := manager.community.Directory(r.Context(), u)
	if err != nil {
		manager.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rooms)
}

func (manager *ApiManagerCtx) publicRoom(w http.ResponseWriter, r *http.Request) {
	room, _, err := manager.community.Get(r.Context(), auth.UserFromContext(r.Context()), chi.URLParam(r, "roomName"), r.URL.Query().Get("invite"))
	if err != nil {
		manager.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, room)
}

func (manager *ApiManagerCtx) publicRoomJoin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Invite      string `json:"invite"`
		DisplayName string `json:"display_name"`
	}
	if !readJSON(w, r, &req) {
		return
	}

	u := auth.UserFromContext(r.Context())
	if u == nil && manager.auth.Policy().RoomsRequireLogin {
		http.Error(w, "sign in to join rooms", http.StatusUnauthorized)
		return
	}

	url, err := manager.community.JoinURL(r.Context(), u, chi.URLParam(r, "roomName"), req.Invite, req.DisplayName)
	if err != nil {
		manager.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"url": url})
}

func (manager *ApiManagerCtx) publicRoomThumbnail(w http.ResponseWriter, r *http.Request) {
	data, err := manager.community.Thumbnail(r.Context(), auth.UserFromContext(r.Context()), chi.URLParam(r, "roomName"), r.URL.Query().Get("invite"))
	if err != nil {
		if errors.Is(err, community.ErrForbidden) {
			http.Error(w, "forbidden", http.StatusForbidden)
		} else {
			http.NotFound(w, r)
		}
		return
	}

	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "private, max-age=15")
	w.Write(data)
}

//
// room metadata (owner or admin)
//

type roomMetaRequest struct {
	Title       *string           `json:"title"`
	Description *string           `json:"description"`
	Category    *string           `json:"category"`
	Visibility  *store.Visibility `json:"visibility"`

	// admin only
	Featured *bool  `json:"featured"`
	Hidden   *bool  `json:"hidden"`
	OwnerID  *int64 `json:"owner_id"`
}

// applyRoomMeta validates and applies requested changes.
func (manager *ApiManagerCtx) applyRoomMeta(u *store.User, meta *store.RoomMeta, req roomMetaRequest) error {
	trim := func(s string, n int) string {
		s = strings.TrimSpace(s)
		if len(s) > n {
			s = s[:n]
		}
		return s
	}

	if req.Title != nil {
		meta.Title = trim(*req.Title, 80)
	}
	if req.Description != nil {
		meta.Description = trim(*req.Description, 500)
	}
	if req.Category != nil {
		meta.Category = trim(*req.Category, 40)
	}
	if req.Visibility != nil && *req.Visibility != meta.Visibility {
		if !req.Visibility.Valid() {
			return errBadRequest("invalid visibility")
		}
		if *req.Visibility == store.VisibilityPublic && !u.IsAdmin() && !manager.auth.Policy().UsersCanMakePublic {
			return errBadRequest("public rooms are not allowed for your account")
		}
		meta.Visibility = *req.Visibility
	}

	if req.Featured != nil || req.Hidden != nil || req.OwnerID != nil {
		if !u.IsAdmin() {
			return errBadRequest("only admins can feature, hide or transfer rooms")
		}
		if req.Featured != nil {
			meta.Featured = *req.Featured
		}
		if req.Hidden != nil {
			meta.Hidden = *req.Hidden
		}
		if req.OwnerID != nil {
			if *req.OwnerID <= 0 {
				meta.OwnerID = nil
			} else {
				if _, err := manager.store.GetUser(*req.OwnerID); err != nil {
					return errBadRequest("owner not found")
				}
				id := *req.OwnerID
				meta.OwnerID = &id
			}
		}
	}

	return nil
}

type badRequest string

func (e badRequest) Error() string { return string(e) }

func errBadRequest(msg string) error { return badRequest(msg) }

// metaForRoom loads room entry & meta and checks management permission.
func (manager *ApiManagerCtx) metaForRoom(w http.ResponseWriter, r *http.Request) (*store.RoomMeta, bool) {
	u := auth.UserFromContext(r.Context())
	entry, err := manager.rooms.GetEntry(r.Context(), chi.URLParam(r, "roomId"))
	if err != nil {
		manager.writeErr(w, err)
		return nil, false
	}
	if !manager.community.CanManageEntry(u, entry) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return nil, false
	}

	meta, err := manager.store.GetRoomMeta(entry.Name)
	if errors.Is(err, store.ErrNotFound) {
		meta = &store.RoomMeta{Name: entry.Name, Visibility: store.VisibilityPrivate}
	} else if err != nil {
		manager.writeErr(w, err)
		return nil, false
	}
	return meta, true
}

func (manager *ApiManagerCtx) roomMetaGet(w http.ResponseWriter, r *http.Request) {
	meta, ok := manager.metaForRoom(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, meta)
}

func (manager *ApiManagerCtx) roomMetaUpdate(w http.ResponseWriter, r *http.Request) {
	meta, ok := manager.metaForRoom(w, r)
	if !ok {
		return
	}

	var req roomMetaRequest
	if !readJSON(w, r, &req) {
		return
	}

	if err := manager.applyRoomMeta(auth.UserFromContext(r.Context()), meta, req); err != nil {
		var br badRequest
		if errors.As(err, &br) {
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else {
			manager.writeErr(w, err)
		}
		return
	}

	if err := manager.store.PutRoomMeta(meta); err != nil {
		manager.writeErr(w, err)
		return
	}

	manager.auth.Audit(r, "room.update", meta.Name, "visibility="+string(meta.Visibility))
	writeJSON(w, http.StatusOK, meta)
}

func (manager *ApiManagerCtx) roomInviteRegenerate(w http.ResponseWriter, r *http.Request) {
	meta, ok := manager.metaForRoom(w, r)
	if !ok {
		return
	}

	if r.URL.Query().Get("disable") == "true" {
		meta.InviteCode = ""
	} else {
		meta.InviteCode = community.NewInviteCode()
	}

	if err := manager.store.PutRoomMeta(meta); err != nil {
		manager.writeErr(w, err)
		return
	}

	manager.auth.Audit(r, "room.invite", meta.Name, "")
	writeJSON(w, http.StatusOK, meta)
}

//
// admin moderation
//

func (manager *ApiManagerCtx) adminRooms(w http.ResponseWriter, r *http.Request) {
	rooms, err := manager.community.AllRooms(r.Context(), auth.UserFromContext(r.Context()))
	if err != nil {
		manager.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, rooms)
}

//
// friends
//

func (manager *ApiManagerCtx) friendsEnabled(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !manager.auth.Policy().FriendsEnabled {
			http.Error(w, "friends are disabled", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (manager *ApiManagerCtx) friendsList(w http.ResponseWriter, r *http.Request) {
	u := auth.UserFromContext(r.Context())
	friends, err := manager.store.ListFriends(u.ID)
	if err != nil {
		manager.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, friends)
}

func (manager *ApiManagerCtx) friendsRequest(w http.ResponseWriter, r *http.Request) {
	u := auth.UserFromContext(r.Context())

	var req struct {
		Username string `json:"username"`
	}
	if !readJSON(w, r, &req) {
		return
	}

	other, err := manager.store.GetUserByUsername(strings.TrimSpace(req.Username))
	if err != nil || other.Disabled {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	if other.ID == u.ID {
		http.Error(w, "you can not add yourself", http.StatusBadRequest)
		return
	}

	status, err := manager.store.RequestFriend(u.ID, other.ID)
	if err != nil {
		manager.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"status": status, "user": other.Public()})
}

func (manager *ApiManagerCtx) friendsAccept(w http.ResponseWriter, r *http.Request) {
	u := auth.UserFromContext(r.Context())
	id, ok := idParam(r, "userId")
	if !ok {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}
	if err := manager.store.AcceptFriend(u.ID, id); err != nil {
		manager.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (manager *ApiManagerCtx) friendsRemove(w http.ResponseWriter, r *http.Request) {
	u := auth.UserFromContext(r.Context())
	id, ok := idParam(r, "userId")
	if !ok {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}
	if err := manager.store.RemoveFriend(u.ID, id); err != nil {
		manager.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (manager *ApiManagerCtx) usersSearch(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) < 2 {
		writeJSON(w, http.StatusOK, []store.PublicUser{})
		return
	}
	// LIKE wildcards are not allowed in the query
	q = strings.NewReplacer("%", "", "_", "").Replace(q)

	users, err := manager.store.SearchUsers(q, 10)
	if err != nil {
		manager.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, users)
}
