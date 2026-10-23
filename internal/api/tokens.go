package api

import (
	"net/http"
	"time"

	"github.com/m1k1o/neko-rooms/internal/auth"
)

//
// API tokens, for scripts: Authorization: Bearer nrk_...
//

func (manager *ApiManagerCtx) accountTokens(w http.ResponseWriter, r *http.Request) {
	tokens, err := manager.store.ListAPITokens(auth.UserFromContext(r.Context()).ID)
	if err != nil {
		manager.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, tokens)
}

func (manager *ApiManagerCtx) accountTokenCreate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name    string `json:"name"`
		Expires int    `json:"expires_days"` // 0 = never
	}
	if !readJSON(w, r, &req) {
		return
	}
	if req.Expires < 0 || req.Expires > 3650 {
		http.Error(w, "expiry must be between 0 and 3650 days", http.StatusBadRequest)
		return
	}

	u := auth.UserFromContext(r.Context())
	t, secret, err := manager.auth.NewAPIToken(u, req.Name, time.Duration(req.Expires)*24*time.Hour)
	if err != nil {
		manager.writeErr(w, err)
		return
	}

	manager.auth.Audit(r, "token.create", t.Name, "")
	writeJSON(w, http.StatusCreated, map[string]any{"token": secret, "info": t})
}

func (manager *ApiManagerCtx) accountTokenDelete(w http.ResponseWriter, r *http.Request) {
	id, ok := idParam(r, "tokenId")
	if !ok {
		http.NotFound(w, r)
		return
	}
	if err := manager.store.DeleteAPIToken(auth.UserFromContext(r.Context()).ID, id); err != nil {
		manager.writeErr(w, err)
		return
	}
	manager.auth.Audit(r, "token.delete", "", "")
	w.WriteHeader(http.StatusNoContent)
}
