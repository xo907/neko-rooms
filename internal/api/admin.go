package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/m1k1o/neko-rooms/internal/auth"
	"github.com/m1k1o/neko-rooms/internal/store"
)

//
// users
//

type adminUser struct {
	*store.User
	RoomCount int `json:"room_count"`
}

func (manager *ApiManagerCtx) adminUsersList(w http.ResponseWriter, r *http.Request) {
	users, err := manager.store.ListUsers()
	if err != nil {
		manager.writeErr(w, err)
		return
	}

	metas, err := manager.store.ListRoomMeta()
	if err != nil {
		manager.writeErr(w, err)
		return
	}
	counts := map[int64]int{}
	for _, m := range metas {
		if m.OwnerID != nil {
			counts[*m.OwnerID]++
		}
	}

	res := make([]adminUser, 0, len(users))
	for _, u := range users {
		res = append(res, adminUser{User: u, RoomCount: counts[u.ID]})
	}
	writeJSON(w, http.StatusOK, res)
}

type adminUserRequest struct {
	Username    *string     `json:"username"`
	Password    *string     `json:"password"`
	DisplayName *string     `json:"display_name"`
	Email       *string     `json:"email"`
	Role        *store.Role `json:"role"`
	Disabled    *bool       `json:"disabled"`
	RoomLimit   *int        `json:"room_limit"`
}

func (manager *ApiManagerCtx) adminUsersCreate(w http.ResponseWriter, r *http.Request) {
	var req adminUserRequest
	if !readJSON(w, r, &req) {
		return
	}
	if req.Username == nil || req.Password == nil {
		http.Error(w, "username and password are required", http.StatusBadRequest)
		return
	}

	role := store.RoleUser
	if req.Role != nil {
		role = *req.Role
	}
	if !role.Valid() {
		http.Error(w, "invalid role", http.StatusBadRequest)
		return
	}

	creds := credentials{Username: *req.Username, Password: *req.Password}
	if req.DisplayName != nil {
		creds.DisplayName = *req.DisplayName
	}
	if req.Email != nil {
		creds.Email = *req.Email
	}

	u, err := manager.newUserFromRequest(creds, role)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Disabled != nil {
		u.Disabled = *req.Disabled
	}
	if req.RoomLimit != nil {
		u.RoomLimit = max(*req.RoomLimit, -1)
	}

	if err := manager.store.CreateUser(u); err != nil {
		if errors.Is(err, store.ErrConflict) {
			http.Error(w, "username is already taken", http.StatusConflict)
			return
		}
		manager.writeErr(w, err)
		return
	}

	manager.auth.Audit(r, "user.create", u.Username, "role="+string(u.Role))
	writeJSON(w, http.StatusCreated, u)
}

func (manager *ApiManagerCtx) adminUsersUpdate(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(r, "userId")
	if !ok {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	u, err := manager.store.GetUser(id)
	if err != nil {
		manager.writeErr(w, err)
		return
	}

	var req adminUserRequest
	if !readJSON(w, r, &req) {
		return
	}

	me := auth.UserFromContext(r.Context())
	changes := []string{}
	wasActiveAdmin := u.IsAdmin() && !u.Disabled

	if req.Username != nil && *req.Username != u.Username {
		name := strings.TrimSpace(*req.Username)
		if err := auth.ValidateUsername(name); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		u.Username = name
		changes = append(changes, "username")
	}
	if req.DisplayName != nil {
		u.DisplayName = strings.TrimSpace(*req.DisplayName)
		changes = append(changes, "display_name")
	}
	if req.Email != nil {
		u.Email = strings.TrimSpace(*req.Email)
		changes = append(changes, "email")
	}
	if req.Role != nil && *req.Role != u.Role {
		if !req.Role.Valid() {
			http.Error(w, "invalid role", http.StatusBadRequest)
			return
		}
		if u.ID == me.ID {
			http.Error(w, "you can not change your own role", http.StatusBadRequest)
			return
		}
		u.Role = *req.Role
		changes = append(changes, "role="+string(u.Role))
	}
	if req.Disabled != nil && *req.Disabled != u.Disabled {
		if u.ID == me.ID {
			http.Error(w, "you can not disable your own account", http.StatusBadRequest)
			return
		}
		u.Disabled = *req.Disabled
		changes = append(changes, fmt.Sprintf("disabled=%v", u.Disabled))
	}
	if req.RoomLimit != nil {
		u.RoomLimit = max(*req.RoomLimit, -1)
		changes = append(changes, "room_limit="+strconv.Itoa(u.RoomLimit))
	}
	passwordChanged := false
	if req.Password != nil && *req.Password != "" {
		if err := manager.auth.ValidatePassword(*req.Password); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		hash, err := auth.HashPassword(*req.Password)
		if err != nil {
			manager.writeErr(w, err)
			return
		}
		u.PasswordHash = hash
		passwordChanged = true
		changes = append(changes, "password")
	}

	// never lock everyone out
	if wasActiveAdmin && (!u.IsAdmin() || u.Disabled) {
		n, err := manager.store.CountAdmins()
		if err != nil {
			manager.writeErr(w, err)
			return
		}
		if n <= 1 {
			http.Error(w, "this is the last active admin account", http.StatusBadRequest)
			return
		}
	}

	if err := manager.store.UpdateUser(u); err != nil {
		if errors.Is(err, store.ErrConflict) {
			http.Error(w, "username is already taken", http.StatusConflict)
			return
		}
		manager.writeErr(w, err)
		return
	}

	// disabling or resetting password signs the user out
	if u.Disabled || passwordChanged {
		if err := manager.store.DeleteUserSessions(u.ID, ""); err != nil {
			manager.writeErr(w, err)
			return
		}
	}

	manager.auth.Audit(r, "user.update", u.Username, strings.Join(changes, ", "))
	writeJSON(w, http.StatusOK, u)
}

func (manager *ApiManagerCtx) adminUsersDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(r, "userId")
	if !ok {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	me := auth.UserFromContext(r.Context())
	if id == me.ID {
		http.Error(w, "you can not delete your own account", http.StatusBadRequest)
		return
	}

	u, err := manager.store.GetUser(id)
	if err != nil {
		manager.writeErr(w, err)
		return
	}

	if u.IsAdmin() && !u.Disabled {
		n, err := manager.store.CountAdmins()
		if err != nil {
			manager.writeErr(w, err)
			return
		}
		if n <= 1 {
			http.Error(w, "this is the last active admin account", http.StatusBadRequest)
			return
		}
	}

	// rooms of deleted users stay, owned by nobody (admins only)
	if err := manager.store.DeleteUser(id); err != nil {
		manager.writeErr(w, err)
		return
	}

	manager.auth.Audit(r, "user.delete", u.Username, "")
	w.WriteHeader(http.StatusNoContent)
}

func (manager *ApiManagerCtx) adminUserSessions(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(r, "userId")
	if !ok {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}
	list, err := manager.store.ListUserSessions(id)
	if err != nil {
		manager.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sessionViews(list, auth.SessionFromContext(r.Context())))
}

func (manager *ApiManagerCtx) adminUserSessionsRevoke(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(r, "userId")
	if !ok {
		http.Error(w, "invalid user id", http.StatusBadRequest)
		return
	}

	except := ""
	if sess := auth.SessionFromContext(r.Context()); sess != nil && sess.UserID == id {
		except = sess.ID
	}
	if err := manager.store.DeleteUserSessions(id, except); err != nil {
		manager.writeErr(w, err)
		return
	}

	manager.auth.Audit(r, "user.sessions_revoked", strconv.FormatInt(id, 10), "")
	w.WriteHeader(http.StatusNoContent)
}

//
// policy
//

func (manager *ApiManagerCtx) adminPolicyGet(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, manager.auth.Policy())
}

func (manager *ApiManagerCtx) adminPolicySet(w http.ResponseWriter, r *http.Request) {
	p := manager.auth.Policy()
	if !readJSON(w, r, &p) {
		return
	}
	if err := manager.auth.SetPolicy(p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	manager.auth.Audit(r, "settings.update", "policy", "")
	writeJSON(w, http.StatusOK, manager.auth.Policy())
}

//
// audit
//

func (manager *ApiManagerCtx) adminAudit(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	if limit <= 0 || limit > 500 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	entries, total, err := manager.store.ListAudit(limit, offset)
	if err != nil {
		manager.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": entries, "total": total})
}

//
// overview
//

func (manager *ApiManagerCtx) adminOverview(w http.ResponseWriter, r *http.Request) {
	users, err := manager.store.ListUsers()
	if err != nil {
		manager.writeErr(w, err)
		return
	}

	rooms, err := manager.community.AllRooms(r.Context(), auth.UserFromContext(r.Context()))
	if err != nil {
		manager.writeErr(w, err)
		return
	}

	res := map[string]int{
		"users":          len(users),
		"admins":         0,
		"disabled_users": 0,
		"rooms":          len(rooms),
		"rooms_running":  0,
		"rooms_public":   0,
		"viewers":        0,
	}
	for _, u := range users {
		if u.IsAdmin() {
			res["admins"]++
		}
		if u.Disabled {
			res["disabled_users"]++
		}
	}
	for _, room := range rooms {
		if room.Running {
			res["rooms_running"]++
		}
		if room.Visibility == store.VisibilityPublic {
			res["rooms_public"]++
		}
		res["viewers"] += room.Viewers
	}

	writeJSON(w, http.StatusOK, res)
}
