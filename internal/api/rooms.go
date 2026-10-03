package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"github.com/m1k1o/neko-rooms/internal/auth"
	"github.com/m1k1o/neko-rooms/internal/community"
	"github.com/m1k1o/neko-rooms/internal/room"
	"github.com/m1k1o/neko-rooms/internal/store"
	"github.com/m1k1o/neko-rooms/internal/types"
)

// requireRoomManager allows only the room owner or admins.
func (manager *ApiManagerCtx) requireRoomManager(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := auth.UserFromContext(r.Context())
		if u.IsAdmin() {
			next.ServeHTTP(w, r)
			return
		}

		entry, err := manager.rooms.GetEntry(r.Context(), chi.URLParam(r, "roomId"))
		if err != nil {
			manager.writeErr(w, err)
			return
		}
		if !manager.community.CanManageEntry(u, entry) {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (manager *ApiManagerCtx) requirePullPermission(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		u := auth.UserFromContext(r.Context())
		if !u.IsAdmin() && !manager.auth.Policy().UsersCanPullImages {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// checkUserRoomSettings enforces the policy on room settings sent by non-admins.
func (manager *ApiManagerCtx) checkUserRoomSettings(u *store.User, settings *types.RoomSettings) error {
	if u.IsAdmin() {
		return nil
	}

	p := manager.auth.Policy()

	if len(p.UserNekoImages) > 0 && !slices.Contains(p.UserNekoImages, settings.NekoImage) {
		return fmt.Errorf("neko image %q is not allowed for your account", settings.NekoImage)
	}

	for _, m := range settings.Mounts {
		if (m.Type == types.MountPublic || m.Type == types.MountProtected) && !p.UsersCanUseMounts {
			return fmt.Errorf("%s mounts are not allowed for your account", m.Type)
		}
	}

	if !p.UsersCanSetDevices && (len(settings.Resources.Devices) > 0 || len(settings.Resources.Gpus) > 0) {
		return fmt.Errorf("devices and gpus are not allowed for your account")
	}

	if p.UserMaxConnections > 0 && settings.MaxConnections > uint16(p.UserMaxConnections) {
		return fmt.Errorf("max connections is limited to %d for your account", p.UserMaxConnections)
	}

	return nil
}

func (manager *ApiManagerCtx) roomsList(w http.ResponseWriter, r *http.Request) {
	labelsMap := map[string]string{}
	for key, value := range r.URL.Query() {
		key = strings.ToLower(key)

		if !room.CheckLabelKey(key) {
			http.Error(w, "invalid label name, allowed characters: [a-z0-9.-]", 400)
			return
		}

		labelsMap[key] = value[0]
	}

	response, err := manager.rooms.List(r.Context(), labelsMap)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	response, err = manager.community.VisibleEntries(auth.UserFromContext(r.Context()), response)
	if err != nil {
		manager.writeErr(w, err)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

type roomCreateRequest struct {
	types.RoomSettings
	Meta *roomMetaRequest `json:"meta,omitempty"`
}

func (manager *ApiManagerCtx) roomCreate(w http.ResponseWriter, r *http.Request) {
	u := auth.UserFromContext(r.Context())
	p := manager.auth.Policy()

	var start = true // default value
	if s := r.URL.Query().Get("start"); s != "" {
		var err error
		start, err = strconv.ParseBool(s)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
	}

	// Default values
	request := roomCreateRequest{
		RoomSettings: types.RoomSettings{
			MaxConnections: 10,
			Resources: types.RoomResources{
				ShmSize: 2 * 1e9,
			},
		},
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	if !u.IsAdmin() {
		if !p.UsersCanCreateRooms {
			http.Error(w, "creating rooms is not allowed for your account", http.StatusForbidden)
			return
		}

		if limit := p.RoomLimit(u.RoomLimit); limit > 0 {
			count, err := manager.store.CountRoomsOwnedBy(u.ID)
			if err != nil {
				manager.writeErr(w, err)
				return
			}
			if count >= limit {
				http.Error(w, fmt.Sprintf("room limit reached (%d)", limit), http.StatusForbidden)
				return
			}
		}

		if err := manager.checkUserRoomSettings(u, &request.RoomSettings); err != nil {
			http.Error(w, err.Error(), http.StatusForbidden)
			return
		}
	}

	// room names identify metadata, do not allow taking over a stale entry
	if request.Name != "" {
		if meta, err := manager.store.GetRoomMeta(request.Name); err == nil && meta.OwnerID != nil && *meta.OwnerID != u.ID && !u.IsAdmin() {
			http.Error(w, "room name is already taken", http.StatusConflict)
			return
		}
	}

	ID, err := manager.rooms.Create(r.Context(), request.RoomSettings)
	if err != nil {
		manager.logger.Error().Err(err).Msg("create: failed to create room")
		http.Error(w, err.Error(), 500)
		return
	}

	response, err := manager.rooms.GetEntry(r.Context(), ID)
	if err != nil {
		manager.logger.Error().Err(err).Msg("create: failed to get room entry")
		http.Error(w, err.Error(), 500)
		return
	}

	// store ownership & community metadata
	meta := &store.RoomMeta{
		Name:       response.Name,
		OwnerID:    &u.ID,
		Visibility: store.VisibilityPrivate,
		InviteCode: community.NewInviteCode(),
	}
	if request.Meta != nil {
		if err := manager.applyRoomMeta(u, meta, *request.Meta); err != nil {
			// room exists already, keep it private rather than failing
			manager.logger.Warn().Err(err).Str("room", meta.Name).Msg("create: invalid room metadata")
		}
	}
	if err := manager.store.PutRoomMeta(meta); err != nil {
		manager.logger.Error().Err(err).Msg("create: failed to store room metadata")
	}

	manager.auth.Audit(r, "room.create", response.Name, "image="+request.NekoImage)

	if start {
		if err := manager.rooms.Start(r.Context(), ID); err != nil {
			manager.logger.Error().Err(err).Msg("create: failed to start room")
			http.Error(w, err.Error(), 500)
			return
		}

		response, err = manager.rooms.GetEntry(r.Context(), ID)
		if err != nil {
			manager.logger.Error().Err(err).Msg("create: failed to get room entry")
			http.Error(w, err.Error(), 500)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (manager *ApiManagerCtx) roomRecreate(w http.ResponseWriter, r *http.Request) {
	u := auth.UserFromContext(r.Context())
	roomId := chi.URLParam(r, "roomId")

	entry, err := manager.rooms.GetEntry(r.Context(), roomId)
	if err != nil {
		if errors.Is(err, types.ErrRoomNotFound) {
			http.Error(w, err.Error(), 404)
		} else {
			manager.logger.Error().Err(err).Msg("recreate: failed to get room entry")
			http.Error(w, err.Error(), 500)
		}
		return
	}

	start := entry.Running
	if s := r.URL.Query().Get("start"); s != "" {
		start, err = strconv.ParseBool(s)
		if err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
	}

	settings, err := manager.rooms.GetSettings(r.Context(), roomId)
	if err != nil {
		if errors.Is(err, types.ErrRoomNotFound) {
			http.Error(w, err.Error(), 404)
		} else {
			manager.logger.Error().Err(err).Msg("recreate: failed to get room settings")
			http.Error(w, err.Error(), 500)
		}
		return
	}

	// optional settings payload
	if err := json.NewDecoder(r.Body).Decode(&settings); err != nil && !errors.Is(err, io.EOF) {
		http.Error(w, err.Error(), 400)
		return
	}

	if err := manager.checkUserRoomSettings(u, settings); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	oldName := entry.Name
	if settings.Name != "" && settings.Name != oldName {
		if _, err := manager.store.GetRoomMeta(settings.Name); err == nil {
			http.Error(w, "room name is already taken", http.StatusConflict)
			return
		}
	}

	if err := manager.rooms.Remove(r.Context(), roomId); err != nil {
		manager.logger.Error().Err(err).Msg("recreate: failed to remove room")
		http.Error(w, err.Error(), 500)
		return
	}

	ID, err := manager.rooms.Create(r.Context(), *settings)
	if err != nil {
		manager.logger.Error().Err(err).Msg("recreate: failed to create room")
		http.Error(w, err.Error(), 500)
		return
	}

	if start {
		if err := manager.rooms.Start(r.Context(), ID); err != nil {
			manager.logger.Error().Err(err).Msg("recreate: failed to start room")
			http.Error(w, err.Error(), 500)
			return
		}
	}

	response, err := manager.rooms.GetEntry(r.Context(), ID)
	if err != nil {
		manager.logger.Error().Err(err).Msg("recreate: failed to get room entry")
		http.Error(w, err.Error(), 500)
		return
	}

	// keep metadata when the room was renamed
	if response.Name != oldName {
		if meta, err := manager.store.GetRoomMeta(oldName); err == nil {
			meta.Name = response.Name
			if err := manager.store.PutRoomMeta(meta); err == nil {
				manager.store.DeleteRoomMeta(oldName)
			}
		}
	}

	manager.auth.Audit(r, "room.recreate", response.Name, "")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (manager *ApiManagerCtx) roomRemove(w http.ResponseWriter, r *http.Request) {
	roomId := chi.URLParam(r, "roomId")

	entry, err := manager.rooms.GetEntry(r.Context(), roomId)
	if err != nil {
		manager.writeErr(w, err)
		return
	}

	if err := manager.rooms.Remove(r.Context(), roomId); err != nil {
		manager.writeErr(w, err)
		return
	}

	if err := manager.store.DeleteRoomMeta(entry.Name); err != nil {
		manager.logger.Err(err).Msg("remove: failed to delete room metadata")
	}

	manager.auth.Audit(r, "room.remove", entry.Name, "")
	w.WriteHeader(http.StatusNoContent)
}

func (manager *ApiManagerCtx) roomGetEntry(w http.ResponseWriter, r *http.Request) {
	roomId := chi.URLParam(r, "roomId")

	response, err := manager.rooms.GetEntry(r.Context(), roomId)
	if err != nil {
		if errors.Is(err, types.ErrRoomNotFound) {
			http.Error(w, err.Error(), 404)
		} else {
			http.Error(w, err.Error(), 500)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (manager *ApiManagerCtx) roomGetEntryByName(w http.ResponseWriter, r *http.Request) {
	// roomId is actually room name here
	roomName := chi.URLParam(r, "roomId")

	response, err := manager.rooms.GetEntryByName(r.Context(), roomName)
	if err != nil {
		if errors.Is(err, types.ErrRoomNotFound) {
			http.Error(w, err.Error(), 404)
		} else {
			http.Error(w, err.Error(), 500)
		}
		return
	}

	if !manager.community.CanManageEntry(auth.UserFromContext(r.Context()), response) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (manager *ApiManagerCtx) roomGetSettings(w http.ResponseWriter, r *http.Request) {
	roomId := chi.URLParam(r, "roomId")

	response, err := manager.rooms.GetSettings(r.Context(), roomId)
	if err != nil {
		if errors.Is(err, types.ErrRoomNotFound) {
			http.Error(w, err.Error(), 404)
		} else {
			http.Error(w, err.Error(), 500)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (manager *ApiManagerCtx) roomGetStats(w http.ResponseWriter, r *http.Request) {
	roomId := chi.URLParam(r, "roomId")

	response, err := manager.rooms.GetStats(r.Context(), roomId)
	if err != nil {
		if errors.Is(err, types.ErrRoomNotFound) {
			http.Error(w, err.Error(), 404)
		} else {
			http.Error(w, err.Error(), 500)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (manager *ApiManagerCtx) roomGenericAction(action string, fn func(ctx context.Context, id string) error) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		roomId := chi.URLParam(r, "roomId")

		err := fn(r.Context(), roomId)
		if err != nil {
			if errors.Is(err, types.ErrRoomNotFound) {
				http.Error(w, err.Error(), 404)
			} else {
				http.Error(w, err.Error(), 500)
			}
			return
		}

		manager.auth.Audit(r, action, roomId, "")
		w.WriteHeader(http.StatusNoContent)
	}
}

func (manager *ApiManagerCtx) dockerCompose(w http.ResponseWriter, r *http.Request) {
	response, err := manager.rooms.ExportAsDockerCompose(r.Context())
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	w.Header().Set("Content-Type", "text/yaml")
	w.Write(response)
}
