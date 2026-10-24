package community

import (
	"context"
	"errors"
	"sort"

	"github.com/m1k1o/neko-rooms/internal/captions"
)

// Subtitles: viewers pick a language on the live channel, the room's sound
// is transcribed while anyone has them on, lines go to those who picked
// their language.

func (s *Service) captionsConfig(i Integrations) captions.Config {
	return captions.Config{
		URL:          i.CaptionsURL,
		Model:        i.CaptionsModel,
		TranslateURL: i.TranslateURL,
		TranslateKey: i.TranslateKey,
	}
}

// CaptionLanguages are the languages viewers of a room can pick, none when
// subtitles are off.
func (s *Service) CaptionLanguages(ctx context.Context, room, ticket string) ([]captions.Language, error) {
	if _, err := s.ParseTicket(room, ticket); err != nil {
		return nil, err
	}
	return s.captions.Languages(ctx), nil
}

// SetCaptions sets the language a viewer reads subtitles in, "" for none.
func (s *Service) SetCaptions(v *LiveViewer, lang string) error {
	if len(lang) > 16 {
		return errors.New("unknown language")
	}
	if lang != "" && !s.captions.Config().Enabled() {
		return errors.New("subtitles are off on this site")
	}
	room := v.Ticket.Room

	s.live.mu.Lock()
	r := s.live.room(room)
	if lang == "" {
		delete(r.captions, v.Ticket.ID)
	} else {
		r.captions[v.Ticket.ID] = lang
	}
	langs := r.captionLangs()
	s.live.mu.Unlock()

	s.captions.Want(room, langs)
	s.resyncRewind()
	return nil
}

// captionLangs are the languages picked in a room, call with the hub locked
func (r *liveRoom) captionLangs() []string {
	set := map[string]bool{}
	for _, lang := range r.captions {
		set[lang] = true
	}
	out := make([]string, 0, len(set))
	for lang := range set {
		out = append(out, lang)
	}
	sort.Strings(out)
	return out
}

// sendCaption delivers a line to the viewers who read lang
func (s *Service) sendCaption(room, lang, text string) {
	s.live.mu.Lock()
	defer s.live.mu.Unlock()
	r, ok := s.live.rooms[room]
	if !ok {
		return
	}
	data := mustJSON(liveEvent{Type: "caption", Lang: lang, Text: text})
	for sub := range r.subs {
		if r.captions[sub.id] != lang {
			continue
		}
		select {
		case sub.ch <- data:
		default:
		}
	}
}

// TestCaptions transcribes a spoken sample with the given settings.
func (s *Service) TestCaptions(ctx context.Context, i Integrations, lang string) (string, string, error) {
	if err := i.Validate(); err != nil {
		return "", "", err
	}
	if i.CaptionsURL == "" {
		return "", "", errors.New("set the speech to text server first")
	}
	return s.captions.Test(ctx, s.captionsConfig(i), lang)
}
