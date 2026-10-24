package api

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/m1k1o/neko-rooms/internal/auth"
	"github.com/m1k1o/neko-rooms/internal/captions"
	"github.com/m1k1o/neko-rooms/internal/community"
	"github.com/m1k1o/neko-rooms/internal/store"
)

//
// live channel of a room
//

type liveRequest struct {
	Ticket string `json:"t"`
	Name   string `json:"name"`

	Emoji  string `json:"emoji"`
	Via    string `json:"via"`
	Up     bool   `json:"up"`
	Target string `json:"target"`
	URL    string `json:"url"`
	Title  string `json:"title"`
	Lang   string `json:"lang"`
}

func liveTicket(r *http.Request, body string) string {
	if t := r.Header.Get("X-Room-Ticket"); t != "" {
		return t
	}
	if body != "" {
		return body
	}
	return r.URL.Query().Get("t")
}

func (manager *ApiManagerCtx) liveError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, community.ErrBadTicket):
		http.Error(w, err.Error(), http.StatusUnauthorized)
	case errors.Is(err, community.ErrForbidden):
		http.Error(w, "forbidden", http.StatusForbidden)
	case errors.Is(err, store.ErrNotFound):
		http.Error(w, "not found", http.StatusNotFound)
	default:
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
}

func (manager *ApiManagerCtx) liveEvents(w http.ResponseWriter, r *http.Request) {
	v, err := manager.community.Viewer(chi.URLParam(r, "roomName"), liveTicket(r, ""), r.URL.Query().Get("name"))
	if err != nil {
		manager.liveError(w, err)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming is not supported", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()

	events, leave := manager.community.Subscribe(v)
	defer leave()

	ping := time.NewTicker(25 * time.Second)
	defer ping.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ping.C:
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		case data := <-events:
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func (manager *ApiManagerCtx) liveAction(w http.ResponseWriter, r *http.Request) {
	var req liveRequest
	if !readJSON(w, r, &req) {
		return
	}
	v, err := manager.community.Viewer(chi.URLParam(r, "roomName"), liveTicket(r, req.Ticket), req.Name)
	if err != nil {
		manager.liveError(w, err)
		return
	}

	id, _ := strconv.Atoi(chi.URLParam(r, "itemId"))
	switch chi.URLParam(r, "action") {
	case "react":
		err = manager.community.React(v, req.Emoji, req.Via)
	case "hand":
		err = manager.community.Hand(v, req.Up, req.Target)
	case "queue":
		err = manager.community.QueueAdd(v, req.URL, req.Title)
	case "vote":
		err = manager.community.QueueVote(v, id)
	case "play":
		err = manager.community.QueuePlay(r.Context(), v, id)
	case "remove":
		err = manager.community.QueueRemove(v, id)
	case "captions":
		err = manager.community.SetCaptions(v, req.Lang)
	default:
		http.NotFound(w, r)
		return
	}
	if err != nil {
		manager.liveError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

//
// sleeping rooms
//

func (manager *ApiManagerCtx) publicRoomWake(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Invite string `json:"invite"`
	}
	if !readJSON(w, r, &req) {
		return
	}
	u := auth.UserFromContext(r.Context())
	if u == nil && manager.auth.Policy().RoomsRequireLogin {
		http.Error(w, "sign in to join rooms", http.StatusUnauthorized)
		return
	}
	if err := manager.community.Wake(r.Context(), u, chi.URLParam(r, "roomName"), req.Invite); err != nil {
		manager.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusAccepted, map[string]bool{"waking": true})
}

//
// watch parties
//

func (manager *ApiManagerCtx) publicRoomRSVP(w http.ResponseWriter, r *http.Request) {
	u := auth.UserFromContext(r.Context())
	if u == nil {
		http.Error(w, "sign in to RSVP", http.StatusUnauthorized)
		return
	}
	name := chi.URLParam(r, "roomName")
	room, _, err := manager.community.Get(r.Context(), u, name, r.URL.Query().Get("invite"))
	if err != nil {
		manager.writeErr(w, err)
		return
	}
	if room.ScheduledAt == nil {
		http.Error(w, "this room has no watch party planned", http.StatusBadRequest)
		return
	}
	if err := manager.store.SetRSVP(room.Name, u.ID, r.Method == http.MethodPost); err != nil {
		manager.writeErr(w, err)
		return
	}
	room, _, err = manager.community.Get(r.Context(), u, name, r.URL.Query().Get("invite"))
	if err != nil {
		manager.writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, room)
}

// siteBase is the absolute address of the site root, from an api request
func siteBase(r *http.Request) string {
	if i := strings.Index(r.URL.Path, "/api/"); i >= 0 {
		return RequestOrigin(r) + r.URL.Path[:i+1]
	}
	return RequestOrigin(r) + "/"
}

// icsEscape escapes text values in a calendar file
func icsEscape(s string) string {
	return strings.NewReplacer(`\`, `\\`, ";", `\;`, ",", `\,`, "\r\n", `\n`, "\n", `\n`).Replace(s)
}

// publicRoomCalendar is the watch party as a calendar event (.ics)
func (manager *ApiManagerCtx) publicRoomCalendar(w http.ResponseWriter, r *http.Request) {
	invite := r.URL.Query().Get("invite")
	name := strings.TrimSuffix(chi.URLParam(r, "roomName"), ".ics")
	room, _, err := manager.community.Get(r.Context(), auth.UserFromContext(r.Context()), name, invite)
	if err != nil {
		manager.writeErr(w, err)
		return
	}
	if room.ScheduledAt == nil {
		http.NotFound(w, r)
		return
	}

	b := manager.branding.Get()
	link := siteBase(r) + "r/" + url.PathEscape(room.Name)
	if invite != "" {
		link += "?invite=" + url.QueryEscape(invite)
	}

	const layout = "20060102T150405Z"
	start := room.ScheduledAt.UTC()
	desc := room.Description
	if desc != "" {
		desc += "\n\n"
	}
	desc += link

	var sb strings.Builder
	line := func(s string) { sb.WriteString(s + "\r\n") }
	line("BEGIN:VCALENDAR")
	line("VERSION:2.0")
	line("PRODID:-//" + icsEscape(b.AppName) + "//watch party//EN")
	line("CALSCALE:GREGORIAN")
	line("METHOD:PUBLISH")
	line("BEGIN:VEVENT")
	line("UID:" + room.Name + "-" + strconv.FormatInt(start.Unix(), 10) + "@" + requestHost(r))
	line("DTSTAMP:" + time.Now().UTC().Format(layout))
	line("DTSTART:" + start.Format(layout))
	line("DTEND:" + start.Add(2*time.Hour).Format(layout))
	line("SUMMARY:" + icsEscape(room.Title))
	line("DESCRIPTION:" + icsEscape(desc))
	line("URL:" + link)
	line("LOCATION:" + link)
	line("END:VEVENT")
	line("END:VCALENDAR")

	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+room.Name+`.ics"`)
	w.Write([]byte(sb.String()))
}

//
// admin integrations
//

func (manager *ApiManagerCtx) adminIntegrationsGet(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, manager.community.Integrations())
}

func (manager *ApiManagerCtx) adminIntegrationsSet(w http.ResponseWriter, r *http.Request) {
	in := manager.community.Integrations()
	if !readJSON(w, r, &in) {
		return
	}
	if err := manager.community.SetIntegrations(in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	manager.auth.Audit(r, "settings.update", "integrations", "")
	writeJSON(w, http.StatusOK, manager.community.Integrations())
}

func (manager *ApiManagerCtx) adminIntegrationsTest(w http.ResponseWriter, r *http.Request) {
	in := manager.community.Integrations()
	if !readJSON(w, r, &in) {
		return
	}
	if in.PublicURL == "" {
		manager.community.SeenBase(siteBase(r))
	}
	if err := manager.community.TestWebhook(r.Context(), in); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

//
// rewind
//

func (manager *ApiManagerCtx) rewindIndex(w http.ResponseWriter, r *http.Request) {
	idx, err := manager.community.RewindIndex(chi.URLParam(r, "roomName"), liveTicket(r, ""))
	if err != nil {
		manager.liveError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, http.StatusOK, idx)
}

func (manager *ApiManagerCtx) rewindSegment(w http.ResponseWriter, r *http.Request) {
	n, err := strconv.Atoi(chi.URLParam(r, "n"))
	if err != nil {
		http.Error(w, "bad segment", http.StatusBadRequest)
		return
	}
	f, seg, err := manager.community.RewindSegment(chi.URLParam(r, "roomName"), liveTicket(r, ""), n)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			http.Error(w, "gone", http.StatusNotFound)
			return
		}
		manager.liveError(w, err)
		return
	}
	defer f.Close()
	// segments never change once they are listed
	w.Header().Set("Content-Type", "video/webm")
	w.Header().Set("Cache-Control", "private, max-age=3600, immutable")
	http.ServeContent(w, r, strconv.Itoa(seg.N)+".webm", time.Time{}, f)
}

//
// subtitles
//

func (manager *ApiManagerCtx) captionLanguages(w http.ResponseWriter, r *http.Request) {
	langs, err := manager.community.CaptionLanguages(r.Context(), chi.URLParam(r, "roomName"), liveTicket(r, ""))
	if err != nil {
		manager.liveError(w, err)
		return
	}
	if langs == nil {
		langs = []captions.Language{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"languages": langs})
}

func (manager *ApiManagerCtx) adminCaptionsTest(w http.ResponseWriter, r *http.Request) {
	var req struct {
		community.Integrations
		Lang string `json:"lang"`
	}
	req.Integrations = manager.community.Integrations()
	if !readJSON(w, r, &req) {
		return
	}
	text, translated, err := manager.community.TestCaptions(r.Context(), req.Integrations, req.Lang)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"text": text, "translated": translated})
}
