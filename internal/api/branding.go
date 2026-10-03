package api

import (
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/m1k1o/neko-rooms/internal/branding"
)

func (manager *ApiManagerCtx) brandingGet(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache")
	writeJSON(w, http.StatusOK, manager.branding.Get())
}

func (manager *ApiManagerCtx) brandingDefaults(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, branding.Default())
}

func (manager *ApiManagerCtx) brandingAsset(w http.ResponseWriter, r *http.Request) {
	a, err := manager.branding.GetAsset(chi.URLParam(r, "name"))
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", a.Mime)
	w.Header().Set("Content-Length", strconv.Itoa(len(a.Data)))
	w.Header().Set("X-Content-Type-Options", "nosniff")
	// neutralize scripts in uploaded svg files
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; img-src data:; sandbox")
	w.Header().Set("ETag", `"`+a.Hash+`"`)
	if r.URL.Query().Get("v") != "" {
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		w.Header().Set("Cache-Control", "no-cache")
	}

	if r.Header.Get("If-None-Match") == `"`+a.Hash+`"` {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Write(a.Data)
}

func (manager *ApiManagerCtx) adminBrandingSet(w http.ResponseWriter, r *http.Request) {
	b := manager.branding.Get()
	if !readJSON(w, r, &b) {
		return
	}
	if err := manager.branding.Set(b); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	manager.auth.Audit(r, "branding.update", "", "")
	writeJSON(w, http.StatusOK, manager.branding.Get())
}

func (manager *ApiManagerCtx) adminBrandingReset(w http.ResponseWriter, r *http.Request) {
	b, err := manager.branding.Reset()
	if err != nil {
		manager.writeErr(w, err)
		return
	}
	manager.auth.Audit(r, "branding.reset", "", "")
	writeJSON(w, http.StatusOK, b)
}

type assetView struct {
	Name      string `json:"name"`
	Mime      string `json:"mime"`
	Size      int    `json:"size"`
	URL       string `json:"url"`
	UpdatedAt string `json:"updated_at"`
}

func (manager *ApiManagerCtx) adminAssetsList(w http.ResponseWriter, r *http.Request) {
	assets, err := manager.branding.ListAssets()
	if err != nil {
		manager.writeErr(w, err)
		return
	}

	res := make([]assetView, 0, len(assets))
	for _, a := range assets {
		res = append(res, assetView{
			Name:      a.Name,
			Mime:      a.Mime,
			Size:      a.Size,
			URL:       manager.branding.AssetURL(a),
			UpdatedAt: a.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	writeJSON(w, http.StatusOK, res)
}

func (manager *ApiManagerCtx) adminAssetUpload(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	r.Body = http.MaxBytesReader(w, r.Body, branding.MaxAssetSize+1<<16)

	var data []byte
	var err error

	// accept both multipart form uploads and raw bodies
	if file, _, ferr := r.FormFile("file"); ferr == nil {
		defer file.Close()
		data, err = io.ReadAll(file)
	} else {
		data, err = io.ReadAll(r.Body)
	}
	if err != nil {
		http.Error(w, "unable to read upload: "+err.Error(), http.StatusBadRequest)
		return
	}

	a, err := manager.branding.PutAsset(name, data)
	if err != nil {
		if errors.Is(err, branding.ErrInvalidAsset) {
			http.Error(w, err.Error(), http.StatusBadRequest)
		} else {
			manager.writeErr(w, err)
		}
		return
	}

	manager.auth.Audit(r, "branding.asset_upload", name, a.Mime)
	writeJSON(w, http.StatusOK, assetView{
		Name:      a.Name,
		Mime:      a.Mime,
		Size:      a.Size,
		URL:       manager.branding.AssetURL(a),
		UpdatedAt: a.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	})
}

func (manager *ApiManagerCtx) adminAssetDelete(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if err := manager.branding.DeleteAsset(name); err != nil {
		manager.writeErr(w, err)
		return
	}
	manager.auth.Audit(r, "branding.asset_delete", name, "")
	w.WriteHeader(http.StatusNoContent)
}
