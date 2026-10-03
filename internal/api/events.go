package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/m1k1o/neko-rooms/internal/auth"
	"github.com/m1k1o/neko-rooms/internal/types"
)

func (manager *ApiManagerCtx) events(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	sse := r.URL.Query().Has("sse")
	if sse {
		w.Header().Set("Content-Type", "text/event-stream")
	} else {
		w.Header().Set("Content-Type", "application/json")
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Connection does not support streaming", http.StatusBadRequest)
		return
	}

	var ping <-chan time.Time
	if !sse {
		// dummy channel, never ping
		ping = make(<-chan time.Time)
	} else {
		// ping every 1 minute
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		ping = ticker.C
	}

	// regular users only receive events of their own rooms
	u := auth.UserFromContext(r.Context())
	allowed := func(e types.RoomEvent) bool {
		if u.IsAdmin() {
			return true
		}
		owned, err := manager.community.OwnedNames(u)
		return err == nil && owned[e.ContainerLabels["m1k1o.neko_rooms.name"]]
	}

	// listen for room events
	events, errs := manager.rooms.Events(r.Context())
	for {
		select {
		case <-ping:
			fmt.Fprintf(w, ": ping\n\n")
			flusher.Flush()
		case _, ok := <-errs:
			if !ok {
				manager.logger.Debug().Msg("sse channel closed")
			}
			return
		case e := <-events:
			if !allowed(e) {
				continue
			}

			jsonData, err := json.Marshal(e)
			if err != nil {
				manager.logger.Err(err).Msg("failed to marshal event")
				continue
			}

			if sse {
				fmt.Fprintf(w, "event: rooms\n")
				fmt.Fprintf(w, "data: %s\n\n", jsonData)
			} else {
				fmt.Fprintf(w, "rooms\t%s\n", jsonData)
			}

			flusher.Flush()
		}
	}
}
