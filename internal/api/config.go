package api

import (
	"encoding/json"
	"net/http"
	"slices"

	"github.com/m1k1o/neko-rooms/internal/auth"
)

func (manager *ApiManagerCtx) configRooms(w http.ResponseWriter, r *http.Request) {
	response := manager.rooms.Config()

	// regular users may be limited to a subset of images
	if u := auth.UserFromContext(r.Context()); !u.IsAdmin() {
		if allowed := manager.auth.Policy().UserNekoImages; len(allowed) > 0 {
			images := []string{}
			for _, img := range response.NekoImages {
				if slices.Contains(allowed, img) {
					images = append(images, img)
				}
			}
			response.NekoImages = images
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
