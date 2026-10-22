package community

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/url"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/m1k1o/neko-rooms/internal/store"
)

// The live channel of a room: reactions, raised hands and the up next queue,
// shared by everyone watching. Viewers get a ticket when they join a room,
// it is their key to the channel.

const (
	liveSecretKey = "live_secret"
	ticketTTL     = 12 * time.Hour
	maxQueue      = 50
	maxURL        = 2048
	maxTitle      = 120
)

// Reactions the app offers, all of them are neko emotes too, so they show
// up the same in the browser.
var Reactions = []string{"👍", "🔥", "👏", "🎉", "😍", "😮", "😭", "💯"}

// isEmoji is true for a short run of emoji, what neko emotes map to
func isEmoji(s string) bool {
	n := 0
	for _, r := range s {
		n++
		if n > 8 {
			return false
		}
		if r == 0x200d || r == 0xfe0f || (r >= 0x2190 && r <= 0x2bff) || (r >= 0x1f000 && r <= 0x1faff) {
			continue
		}
		return false
	}
	return n > 0
}

var ErrBadTicket = errors.New("invalid or expired room ticket")

type Ticket struct {
	Room    string `json:"r"`
	ID      string `json:"i"`
	Name    string `json:"n,omitempty"`
	Manager bool   `json:"m,omitempty"`
	Exp     int64  `json:"e"`
}

func (s *Service) loadLiveSecret() {
	var secret string
	if err := s.store.GetSetting(liveSecretKey, &secret); err == nil && len(secret) >= 32 {
		s.live.secret = []byte(secret)
		return
	}
	b := make([]byte, 32)
	rand.Read(b)
	secret = base64.RawURLEncoding.EncodeToString(b)
	if err := s.store.SetSetting(liveSecretKey, secret); err != nil {
		s.logger.Err(err).Msg("unable to save live channel secret, tickets end on restart")
	}
	s.live.secret = []byte(secret)
}

func (h *liveHub) sign(payload string) string {
	m := hmac.New(sha256.New, h.secret)
	m.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(m.Sum(nil))
}

func (s *Service) issueTicket(room, name string, manager bool) string {
	id := make([]byte, 9)
	rand.Read(id)
	t := Ticket{
		Room:    room,
		ID:      base64.RawURLEncoding.EncodeToString(id),
		Name:    name,
		Manager: manager,
		Exp:     time.Now().Add(ticketTTL).Unix(),
	}
	data, _ := json.Marshal(t)
	payload := base64.RawURLEncoding.EncodeToString(data)
	return payload + "." + s.live.sign(payload)
}

// ParseTicket checks a ticket for a room.
func (s *Service) ParseTicket(room, ticket string) (*Ticket, error) {
	payload, sig, ok := strings.Cut(ticket, ".")
	if !ok || !hmac.Equal([]byte(sig), []byte(s.live.sign(payload))) {
		return nil, ErrBadTicket
	}
	data, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return nil, ErrBadTicket
	}
	var t Ticket
	if err := json.Unmarshal(data, &t); err != nil || t.Room != room || time.Now().Unix() > t.Exp {
		return nil, ErrBadTicket
	}
	return &t, nil
}

//
// hub
//

type QueueItem struct {
	ID     int       `json:"id"`
	URL    string    `json:"url"`
	Title  string    `json:"title"`
	By     string    `json:"by"`
	ByID   string    `json:"by_id"`
	Voters []string  `json:"voters"` // ticket ids
	Added  time.Time `json:"added"`
}

type Hand struct {
	ID   string    `json:"id"`
	Name string    `json:"name"`
	At   time.Time `json:"at"`
}

type Me struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Manager bool   `json:"manager"`
}

type liveEvent struct {
	Type    string       `json:"type"`
	You     *Me          `json:"you,omitempty"`
	Queue   *[]QueueItem `json:"queue,omitempty"`
	Hands   *[]Hand      `json:"hands,omitempty"`
	Title   string       `json:"title,omitempty"` // playing
	Emoji   string       `json:"emoji,omitempty"`
	Name    string       `json:"name,omitempty"`
	ID      string       `json:"id,omitempty"`
	Item    *QueueItem   `json:"item,omitempty"`
	Playing *string      `json:"playing,omitempty"`
	Via     string       `json:"via,omitempty"` // where a reaction was sent from
}

type liveSub struct {
	id string
	ch chan []byte
}

type liveRoom struct {
	subs   map[*liveSub]struct{}
	queue  []*QueueItem
	hands  []Hand
	nextID int
	sent   map[string][]time.Time // reactions per ticket, for rate limits
}

type liveHub struct {
	mu     sync.Mutex
	rooms  map[string]*liveRoom
	secret []byte
}

func newLiveHub() *liveHub {
	return &liveHub{rooms: map[string]*liveRoom{}}
}

func (h *liveHub) room(name string) *liveRoom {
	r, ok := h.rooms[name]
	if !ok {
		r = &liveRoom{subs: map[*liveSub]struct{}{}, sent: map[string][]time.Time{}}
		h.rooms[name] = r
	}
	return r
}

// sendLocked delivers an event to everyone in a room, slow listeners miss it
func (h *liveHub) sendLocked(name string, ev liveEvent) {
	r, ok := h.rooms[name]
	if !ok || len(r.subs) == 0 {
		return
	}
	data, _ := json.Marshal(ev)
	for sub := range r.subs {
		select {
		case sub.ch <- data:
		default:
		}
	}
}

func (h *liveHub) send(name string, ev liveEvent) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.sendLocked(name, ev)
}

func (r *liveRoom) sortedQueue() []QueueItem {
	sort.SliceStable(r.queue, func(i, j int) bool {
		a, b := r.queue[i], r.queue[j]
		if len(a.Voters) != len(b.Voters) {
			return len(a.Voters) > len(b.Voters)
		}
		return a.Added.Before(b.Added)
	})
	out := make([]QueueItem, 0, len(r.queue))
	for _, it := range r.queue {
		c := *it
		c.Voters = append([]string{}, it.Voters...)
		out = append(out, c)
	}
	return out
}

func (r *liveRoom) handsCopy() []Hand {
	return append([]Hand{}, r.hands...)
}

func (h *liveHub) sendQueueLocked(name string, r *liveRoom) {
	q := r.sortedQueue()
	h.sendLocked(name, liveEvent{Type: "queue", Queue: &q})
}

func (h *liveHub) sendHandsLocked(name string, r *liveRoom) {
	hs := r.handsCopy()
	h.sendLocked(name, liveEvent{Type: "hands", Hands: &hs})
}

//
// service side
//

// LiveViewer is someone on the channel of a room.
type LiveViewer struct {
	Ticket *Ticket
	Name   string
}

// viewerName is the name a viewer shows up with
func viewerName(t *Ticket, given string) string {
	if t.Name != "" {
		return t.Name
	}
	given = strings.TrimFunc(given, func(r rune) bool { return unicode.IsSpace(r) || unicode.IsControl(r) })
	if r := []rune(given); len(r) > 32 {
		given = string(r[:32])
	}
	if given == "" {
		return "Guest"
	}
	return given
}

func (s *Service) Viewer(room, ticket, name string) (*LiveViewer, error) {
	t, err := s.ParseTicket(room, ticket)
	if err != nil {
		return nil, err
	}
	return &LiveViewer{Ticket: t, Name: viewerName(t, name)}, nil
}

// Subscribe joins the channel of a room, the first message tells the
// viewer about the room. Call the returned function to leave.
func (s *Service) Subscribe(v *LiveViewer) (<-chan []byte, func()) {
	room := v.Ticket.Room
	sub := &liveSub{id: v.Ticket.ID, ch: make(chan []byte, 32)}

	s.live.mu.Lock()
	r := s.live.room(room)
	r.subs[sub] = struct{}{}
	q := r.sortedQueue()
	hs := r.handsCopy()
	s.live.mu.Unlock()

	playing := ""
	if entry, err := s.rooms.GetEntryByName(s.ctx, room); err == nil {
		playing = s.getStats(entry.ID).playing
	}
	init, _ := json.Marshal(liveEvent{
		Type:    "init",
		You:     &Me{ID: v.Ticket.ID, Name: v.Name, Manager: v.Ticket.Manager},
		Queue:   &q,
		Hands:   &hs,
		Playing: &playing,
	})
	sub.ch <- init

	return sub.ch, func() {
		s.live.mu.Lock()
		defer s.live.mu.Unlock()
		delete(r.subs, sub)

		// a hand goes down when its owner leaves
		for other := range r.subs {
			if other.id == sub.id {
				return
			}
		}
		for i, h := range r.hands {
			if h.ID == sub.id {
				r.hands = append(r.hands[:i], r.hands[i+1:]...)
				s.live.sendHandsLocked(room, r)
				break
			}
		}
		if len(r.subs) == 0 && len(r.queue) == 0 {
			delete(s.live.rooms, room)
		}
	}
}

// React shares a reaction. via is "neko" for emotes sent with the neko
// client, browsers already show those.
func (s *Service) React(v *LiveViewer, emoji, via string) error {
	if !isEmoji(emoji) {
		return errors.New("unknown reaction")
	}
	if via != "neko" {
		via = ""
	}

	s.live.mu.Lock()
	defer s.live.mu.Unlock()
	r := s.live.room(v.Ticket.Room)

	// at most 8 reactions in 4 seconds each
	now := time.Now()
	recent := r.sent[v.Ticket.ID][:0]
	for _, t := range r.sent[v.Ticket.ID] {
		if now.Sub(t) < 4*time.Second {
			recent = append(recent, t)
		}
	}
	if len(recent) >= 8 {
		r.sent[v.Ticket.ID] = recent
		return nil
	}
	r.sent[v.Ticket.ID] = append(recent, now)

	s.live.sendLocked(v.Ticket.Room, liveEvent{Type: "reaction", Emoji: emoji, Name: v.Name, ID: v.Ticket.ID, Via: via})
	return nil
}

// hostName is the member in control of a room right now
func (s *Service) hostName(room string) string {
	entry, err := s.rooms.GetEntryByName(s.ctx, room)
	if err != nil {
		return ""
	}
	return s.getStats(entry.ID).host
}

// Hand raises or lowers a hand. Lowering someone else's hand is for the
// person in control (handing over control) and room managers.
func (s *Service) Hand(v *LiveViewer, up bool, target string) error {
	if target == "" {
		target = v.Ticket.ID
	}
	if target != v.Ticket.ID {
		if up {
			return ErrForbidden
		}
		if !v.Ticket.Manager && s.hostName(v.Ticket.Room) != v.Name {
			return ErrForbidden
		}
	}

	s.live.mu.Lock()
	defer s.live.mu.Unlock()
	r := s.live.room(v.Ticket.Room)

	at := -1
	for i, h := range r.hands {
		if h.ID == target {
			at = i
			break
		}
	}
	switch {
	case up && at < 0:
		r.hands = append(r.hands, Hand{ID: v.Ticket.ID, Name: v.Name, At: time.Now()})
	case !up && at >= 0:
		r.hands = append(r.hands[:at], r.hands[at+1:]...)
	default:
		return nil
	}
	s.live.sendHandsLocked(v.Ticket.Room, r)
	return nil
}

// CleanURL checks a link for the queue.
func CleanURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw != "" && !strings.Contains(raw, "://") {
		raw = "https://" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || len(raw) > maxURL {
		return "", errors.New("enter a http(s) link")
	}
	for _, r := range raw {
		if unicode.IsControl(r) || unicode.IsSpace(r) {
			return "", errors.New("enter a http(s) link")
		}
	}
	return u.String(), nil
}

func (s *Service) QueueAdd(v *LiveViewer, link, title string) error {
	link, err := CleanURL(link)
	if err != nil {
		return err
	}
	title = strings.TrimSpace(title)
	if r := []rune(title); len(r) > maxTitle {
		title = string(r[:maxTitle])
	}

	s.live.mu.Lock()
	defer s.live.mu.Unlock()
	r := s.live.room(v.Ticket.Room)
	if len(r.queue) >= maxQueue {
		return errors.New("the queue is full")
	}
	for _, it := range r.queue {
		if it.URL == link {
			return errors.New("that link is already in the queue")
		}
	}
	r.nextID++
	r.queue = append(r.queue, &QueueItem{
		ID:     r.nextID,
		URL:    link,
		Title:  title,
		By:     v.Name,
		ByID:   v.Ticket.ID,
		Voters: []string{v.Ticket.ID},
		Added:  time.Now(),
	})
	s.live.sendQueueLocked(v.Ticket.Room, r)
	return nil
}

func (s *Service) QueueVote(v *LiveViewer, id int) error {
	s.live.mu.Lock()
	defer s.live.mu.Unlock()
	r := s.live.room(v.Ticket.Room)
	for _, it := range r.queue {
		if it.ID != id {
			continue
		}
		for i, voter := range it.Voters {
			if voter == v.Ticket.ID {
				// a second vote takes it back
				it.Voters = append(it.Voters[:i], it.Voters[i+1:]...)
				s.live.sendQueueLocked(v.Ticket.Room, r)
				return nil
			}
		}
		it.Voters = append(it.Voters, v.Ticket.ID)
		s.live.sendQueueLocked(v.Ticket.Room, r)
		return nil
	}
	return store.ErrNotFound
}

// take removes an item the viewer may act on: their own, or any for the
// person in control and managers
func (s *Service) take(v *LiveViewer, id int, host bool) (*QueueItem, error) {
	s.live.mu.Lock()
	defer s.live.mu.Unlock()
	r := s.live.room(v.Ticket.Room)
	for i, it := range r.queue {
		if it.ID != id {
			continue
		}
		if it.ByID != v.Ticket.ID && !v.Ticket.Manager && !host {
			return nil, ErrForbidden
		}
		r.queue = append(r.queue[:i], r.queue[i+1:]...)
		s.live.sendQueueLocked(v.Ticket.Room, r)
		return it, nil
	}
	return nil, store.ErrNotFound
}

func (s *Service) QueueRemove(v *LiveViewer, id int) error {
	_, err := s.take(v, id, s.hostName(v.Ticket.Room) == v.Name)
	return err
}

// QueuePlay opens a queued link in the room's browser. Anyone may play when
// nobody is in control, otherwise the person in control and managers.
func (s *Service) QueuePlay(ctx context.Context, v *LiveViewer, id int) error {
	entry, err := s.rooms.GetEntryByName(ctx, v.Ticket.Room)
	if err != nil {
		return err
	}
	host := s.getStats(entry.ID).host
	if host != "" && host != v.Name && !v.Ticket.Manager {
		return errors.New(host + " is in control, ask them to play it")
	}

	it, err := s.take(v, id, true)
	if err != nil {
		return err
	}
	if err := s.rooms.OpenURL(ctx, entry.ID, it.URL); err != nil {
		// put it back
		s.live.mu.Lock()
		r := s.live.room(v.Ticket.Room)
		r.queue = append(r.queue, it)
		s.live.sendQueueLocked(v.Ticket.Room, r)
		s.live.mu.Unlock()
		return err
	}
	s.live.send(v.Ticket.Room, liveEvent{Type: "played", Item: it, Name: v.Name})
	return nil
}
