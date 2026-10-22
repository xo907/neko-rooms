package api

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/m1k1o/neko-rooms/internal/auth"
)

// the client crops & resizes avatars to 256px, this leaves plenty of room
const maxAvatarSize = 512 << 10

var avatarMimes = map[string]bool{
	"image/png":  true,
	"image/jpeg": true,
	"image/gif":  true,
	"image/webp": true,
}

func (manager *ApiManagerCtx) publicAvatar(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(r, "userId")
	if !ok {
		http.NotFound(w, r)
		return
	}

	a, err := manager.store.GetAvatar(id)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	etag := `"` + strconv.FormatInt(a.UpdatedAt.Unix(), 10) + `"`
	w.Header().Set("Content-Type", a.Mime)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; sandbox")
	w.Header().Set("ETag", etag)
	if r.URL.Query().Get("v") != "" {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "no-cache")
	}

	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(a.Data)))
	w.Write(a.Data)
}

// publicAvatarsLookup maps names, as seen in rooms, to avatar urls.
// Only names that were asked for are answered.
func (manager *ApiManagerCtx) publicAvatarsLookup(w http.ResponseWriter, r *http.Request) {
	names := []string{}
	for _, n := range r.URL.Query()["name"] {
		if n = strings.TrimSpace(n); n != "" && len(n) <= 64 {
			names = append(names, n)
		}
		if len(names) >= 50 {
			break
		}
	}

	users, err := manager.store.AvatarsByName(names)
	if err != nil {
		manager.writeErr(w, err)
		return
	}

	res := map[string]string{}
	for name, u := range users {
		res[name] = u.AvatarPath()
	}
	w.Header().Set("Cache-Control", "no-cache")
	writeJSON(w, http.StatusOK, res)
}

func (manager *ApiManagerCtx) accountAvatarUpload(w http.ResponseWriter, r *http.Request) {
	u := auth.UserFromContext(r.Context())
	r.Body = http.MaxBytesReader(w, r.Body, maxAvatarSize+1<<16)

	var data []byte
	var err error

	// accept both multipart form uploads and raw bodies
	if file, _, ferr := r.FormFile("file"); ferr == nil {
		defer file.Close()
		data, err = io.ReadAll(io.LimitReader(file, maxAvatarSize+1))
	} else {
		data, err = io.ReadAll(r.Body)
	}
	if err != nil {
		http.Error(w, "unable to read upload: "+err.Error(), http.StatusBadRequest)
		return
	}
	if len(data) == 0 || len(data) > maxAvatarSize {
		http.Error(w, "avatar must be an image of at most 512 KB", http.StatusBadRequest)
		return
	}

	mime := http.DetectContentType(data)
	if !avatarMimes[mime] {
		http.Error(w, "avatar must be a PNG, JPEG, GIF or WebP image", http.StatusBadRequest)
		return
	}

	version, err := manager.store.PutAvatar(u.ID, mime, data)
	if err != nil {
		manager.writeErr(w, err)
		return
	}

	u.Avatar = version
	manager.auth.Audit(r, "account.avatar_upload", u.Username, mime)
	writeJSON(w, http.StatusOK, u)
}

func (manager *ApiManagerCtx) accountAvatarDelete(w http.ResponseWriter, r *http.Request) {
	u := auth.UserFromContext(r.Context())
	if err := manager.store.DeleteAvatar(u.ID); err != nil {
		manager.writeErr(w, err)
		return
	}

	u.Avatar = 0
	writeJSON(w, http.StatusOK, u)
}

func (manager *ApiManagerCtx) adminUserAvatarDelete(w http.ResponseWriter, r *http.Request) {
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
	if err := manager.store.DeleteAvatar(id); err != nil {
		manager.writeErr(w, err)
		return
	}

	manager.auth.Audit(r, "user.avatar_delete", u.Username, "")
	w.WriteHeader(http.StatusNoContent)
}
