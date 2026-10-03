package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/m1k1o/neko-rooms/internal/auth"
	"github.com/m1k1o/neko-rooms/internal/store"
)

type publicPolicy struct {
	HomepageEnabled      bool `json:"homepage_enabled"`
	GuestsCanBrowse      bool `json:"guests_can_browse"`
	RoomsRequireLogin    bool `json:"rooms_require_login"`
	RegistrationEnabled  bool `json:"registration_enabled"`
	RegistrationApproval bool `json:"registration_approval"`
	FriendsEnabled       bool `json:"friends_enabled"`
	ThumbnailsEnabled    bool `json:"thumbnails_enabled"`
	UsersCanCreateRooms  bool `json:"users_can_create_rooms"`
	UsersCanMakePublic   bool `json:"users_can_make_public"`
	UsersCanPullImages   bool `json:"users_can_pull_images"`
	UsersCanUseMounts    bool `json:"users_can_use_mounts"`
	PasswordMinLength    int  `json:"password_min_length"`
}

type authStatus struct {
	SetupRequired bool         `json:"setup_required"`
	User          *store.User  `json:"user"`
	Policy        publicPolicy `json:"policy"`
	RoomLimit     int          `json:"room_limit"`
	RoomCount     int          `json:"room_count"`
}

func (manager *ApiManagerCtx) authStatus(w http.ResponseWriter, r *http.Request) {
	p := manager.auth.Policy()
	res := authStatus{
		SetupRequired: manager.auth.SetupRequired(),
		User:          auth.UserFromContext(r.Context()),
		Policy: publicPolicy{
			HomepageEnabled:      p.HomepageEnabled,
			GuestsCanBrowse:      p.GuestsCanBrowse,
			RoomsRequireLogin:    p.RoomsRequireLogin,
			RegistrationEnabled:  p.RegistrationEnabled,
			RegistrationApproval: p.RegistrationApproval,
			FriendsEnabled:       p.FriendsEnabled,
			ThumbnailsEnabled:    p.ThumbnailsEnabled,
			UsersCanCreateRooms:  p.UsersCanCreateRooms,
			UsersCanMakePublic:   p.UsersCanMakePublic,
			UsersCanPullImages:   p.UsersCanPullImages,
			UsersCanUseMounts:    p.UsersCanUseMounts,
			PasswordMinLength:    p.PasswordMinLength,
		},
	}

	if res.User != nil {
		res.RoomLimit = p.RoomLimit(res.User.RoomLimit)
		if res.User.IsAdmin() {
			res.RoomLimit = 0
		}
		res.RoomCount, _ = manager.store.CountRoomsOwnedBy(res.User.ID)
	}

	writeJSON(w, http.StatusOK, res)
}

type credentials struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	SetupToken  string `json:"setup_token"`
}

func (manager *ApiManagerCtx) authLogin(w http.ResponseWriter, r *http.Request) {
	var req credentials
	if !readJSON(w, r, &req) {
		return
	}

	u, err := manager.auth.Authenticate(strings.TrimSpace(req.Username), req.Password, auth.ClientIP(r))
	if err != nil {
		switch {
		case errors.Is(err, auth.ErrTooManyAttempts):
			http.Error(w, err.Error(), http.StatusTooManyRequests)
		case errors.Is(err, auth.ErrInvalidCredentials), errors.Is(err, auth.ErrAccountDisabled):
			manager.auth.AuditEntry(&store.AuditEntry{Action: "auth.login_failed", Target: req.Username, IP: auth.ClientIP(r)})
			http.Error(w, err.Error(), http.StatusUnauthorized)
		default:
			manager.writeErr(w, err)
		}
		return
	}

	if _, err := manager.auth.StartSession(w, r, u); err != nil {
		manager.writeErr(w, err)
		return
	}

	manager.auth.AuditEntry(&store.AuditEntry{UserID: &u.ID, Username: u.Username, Action: "auth.login", IP: auth.ClientIP(r)})
	writeJSON(w, http.StatusOK, u)
}

func (manager *ApiManagerCtx) authLogout(w http.ResponseWriter, r *http.Request) {
	manager.auth.EndSession(w, r)
	w.WriteHeader(http.StatusNoContent)
}

func (manager *ApiManagerCtx) newUserFromRequest(req credentials, role store.Role) (*store.User, error) {
	req.Username = strings.TrimSpace(req.Username)
	if err := auth.ValidateUsername(req.Username); err != nil {
		return nil, err
	}
	if err := manager.auth.ValidatePassword(req.Password); err != nil {
		return nil, err
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	displayName := strings.TrimSpace(req.DisplayName)
	if displayName == "" {
		displayName = req.Username
	}
	if len(displayName) > 64 {
		displayName = displayName[:64]
	}

	return &store.User{
		Username:     req.Username,
		DisplayName:  displayName,
		Email:        strings.TrimSpace(req.Email),
		PasswordHash: hash,
		Role:         role,
		RoomLimit:    -1,
	}, nil
}

// authSetup creates the first admin account using the one-time setup token.
func (manager *ApiManagerCtx) authSetup(w http.ResponseWriter, r *http.Request) {
	var req credentials
	if !readJSON(w, r, &req) {
		return
	}

	if !manager.auth.SetupRequired() {
		http.Error(w, "setup already completed", http.StatusConflict)
		return
	}
	if !manager.auth.CheckSetupToken(req.SetupToken) {
		http.Error(w, "invalid setup token, check the server logs", http.StatusForbidden)
		return
	}

	u, err := manager.newUserFromRequest(req, store.RoleAdmin)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	u.RoomLimit = 0

	if err := manager.store.CreateUser(u); err != nil {
		manager.writeErr(w, err)
		return
	}
	manager.auth.FinishSetup()

	if _, err := manager.auth.StartSession(w, r, u); err != nil {
		manager.writeErr(w, err)
		return
	}

	manager.auth.AuditEntry(&store.AuditEntry{UserID: &u.ID, Username: u.Username, Action: "auth.setup", IP: auth.ClientIP(r)})
	writeJSON(w, http.StatusCreated, u)
}

func (manager *ApiManagerCtx) authRegister(w http.ResponseWriter, r *http.Request) {
	p := manager.auth.Policy()
	if !p.RegistrationEnabled {
		http.Error(w, "registration is disabled", http.StatusForbidden)
		return
	}

	var req credentials
	if !readJSON(w, r, &req) {
		return
	}

	u, err := manager.newUserFromRequest(req, store.RoleUser)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	u.Disabled = p.RegistrationApproval

	if err := manager.store.CreateUser(u); err != nil {
		if errors.Is(err, store.ErrConflict) {
			http.Error(w, "username is already taken", http.StatusConflict)
			return
		}
		manager.writeErr(w, err)
		return
	}

	manager.auth.AuditEntry(&store.AuditEntry{UserID: &u.ID, Username: u.Username, Action: "auth.register", IP: auth.ClientIP(r)})

	if u.Disabled {
		writeJSON(w, http.StatusAccepted, map[string]any{"pending_approval": true})
		return
	}

	if _, err := manager.auth.StartSession(w, r, u); err != nil {
		manager.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, u)
}

//
// account (self service)
//

func (manager *ApiManagerCtx) accountUpdate(w http.ResponseWriter, r *http.Request) {
	u := auth.UserFromContext(r.Context())

	var req struct {
		DisplayName *string `json:"display_name"`
		Email       *string `json:"email"`
	}
	if !readJSON(w, r, &req) {
		return
	}

	if req.DisplayName != nil {
		name := strings.TrimSpace(*req.DisplayName)
		if name == "" || len(name) > 64 {
			http.Error(w, "display name must be 1-64 characters", http.StatusBadRequest)
			return
		}
		u.DisplayName = name
	}
	if req.Email != nil {
		u.Email = strings.TrimSpace(*req.Email)
	}

	if err := manager.store.UpdateUser(u); err != nil {
		manager.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (manager *ApiManagerCtx) accountPassword(w http.ResponseWriter, r *http.Request) {
	u := auth.UserFromContext(r.Context())

	var req struct {
		Current string `json:"current_password"`
		New     string `json:"new_password"`
	}
	if !readJSON(w, r, &req) {
		return
	}

	if !auth.CheckPassword(u.PasswordHash, req.Current) {
		http.Error(w, "current password is incorrect", http.StatusBadRequest)
		return
	}
	if err := manager.auth.ValidatePassword(req.New); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	hash, err := auth.HashPassword(req.New)
	if err != nil {
		manager.writeErr(w, err)
		return
	}
	u.PasswordHash = hash
	if err := manager.store.UpdateUser(u); err != nil {
		manager.writeErr(w, err)
		return
	}

	// sign out everywhere else
	except := ""
	if sess := auth.SessionFromContext(r.Context()); sess != nil {
		except = sess.ID
	}
	if err := manager.store.DeleteUserSessions(u.ID, except); err != nil {
		manager.writeErr(w, err)
		return
	}

	manager.auth.Audit(r, "account.password_changed", u.Username, "")
	w.WriteHeader(http.StatusNoContent)
}

type sessionView struct {
	*store.Session
	ID      string `json:"id"`
	Current bool   `json:"current"`
}

func sessionViews(list []*store.Session, current *store.Session) []sessionView {
	views := make([]sessionView, 0, len(list))
	for _, s := range list {
		views = append(views, sessionView{
			Session: s,
			ID:      s.ID[:16], // never expose the full id
			Current: current != nil && current.ID == s.ID,
		})
	}
	return views
}

func (manager *ApiManagerCtx) accountSessions(w http.ResponseWriter, r *http.Request) {
	u := auth.UserFromContext(r.Context())
	list, err := manager.store.ListUserSessions(u.ID)
	if err != nil {
		manager.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, sessionViews(list, auth.SessionFromContext(r.Context())))
}

func (manager *ApiManagerCtx) accountSessionsRevoke(w http.ResponseWriter, r *http.Request) {
	u := auth.UserFromContext(r.Context())
	except := ""
	if sess := auth.SessionFromContext(r.Context()); sess != nil {
		except = sess.ID
	}
	if err := manager.store.DeleteUserSessions(u.ID, except); err != nil {
		manager.writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
