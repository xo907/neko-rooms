// Package captions turns what is said in a room into live subtitles, in the
// language each viewer picks.
//
// The sound comes from the hidden viewer of the rewind package, Opus as the
// room sends it. It is cut into short pieces at pauses, packed into Ogg and
// sent to a speech-to-text server with the OpenAI transcription API, by
// default a local whisper.cpp. Lines in other languages come from a
// LibreTranslate server, or from whisper itself for English.
//
// Nothing runs unless someone in a room has subtitles on, and pieces
// without speech are never sent.
package captions

import (
	"bytes"
	"context"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Original is the language code for subtitles in the language spoken.
const Original = "orig"

const (
	sampleRate = 48000
	minPiece   = 3 * sampleRate      // cut at a pause after this much
	maxPiece   = 10 * sampleRate     // cut anyway after this much
	pauseLen   = sampleRate * 3 / 10 // a pause is this much quiet
	minVoiced  = sampleRate / 2      // pieces with less sound are dropped
	queueLen   = 6
	jobTimeout = 30 * time.Second
)

// Config is where speech is turned into text and translated.
type Config struct {
	URL          string // speech-to-text server, OpenAI transcription API
	Model        string // model name sent to it, whisper.cpp ignores it
	TranslateURL string // LibreTranslate server, optional
	TranslateKey string
}

func (c Config) Enabled() bool { return c.URL != "" }

// Emit delivers a subtitle line to the viewers of a room in lang.
type Emit func(room, lang, text string)

type piece struct {
	packets [][]byte
	samples int
	voiced  int
	quiet   int
	avg     float64
}

type job struct {
	room   string
	ogg    []byte
	prompt string
	langs  []string
}

type Manager struct {
	logger zerolog.Logger
	emit   Emit

	mu     sync.Mutex
	cfg    Config
	wanted map[string][]string // room -> languages viewers picked
	pieces map[string]*piece
	prompt map[string]string // room -> last line, context for the next

	jobs   chan job
	client *serverClient
	ctx    context.Context
	cancel context.CancelFunc
}

func New(emit Emit) *Manager {
	ctx, cancel := context.WithCancel(context.Background())
	m := &Manager{
		logger: log.With().Str("module", "captions").Logger(),
		emit:   emit,
		wanted: map[string][]string{},
		pieces: map[string]*piece{},
		prompt: map[string]string{},
		jobs:   make(chan job, queueLen),
		client: newServerClient(),
		ctx:    ctx,
		cancel: cancel,
	}
	go m.work()
	return m
}

func (m *Manager) Configure(c Config) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cfg = c
}

func (m *Manager) Config() Config {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cfg
}

// Want sets the languages viewers of a room picked, none stops listening.
func (m *Manager) Want(room string, langs []string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(langs) == 0 {
		delete(m.wanted, room)
		delete(m.pieces, room)
		delete(m.prompt, room)
		return
	}
	m.wanted[room] = langs
}

// Wanted is whether anyone in the room has subtitles on.
func (m *Manager) Wanted(room string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cfg.Enabled() && len(m.wanted[room]) > 0
}

// Audio takes one Opus packet of a room as it arrives.
func (m *Manager) Audio(room string, packet []byte) {
	n := opusSamples(packet)
	if n == 0 {
		return
	}

	m.mu.Lock()
	langs := m.wanted[room]
	if !m.cfg.Enabled() || len(langs) == 0 {
		m.mu.Unlock()
		return
	}
	p := m.pieces[room]
	if p == nil {
		p = &piece{avg: float64(len(packet))}
		m.pieces[room] = p
	}

	// pauses are quiet stretches: Opus spends far fewer bytes on them
	size := float64(len(packet))
	quiet := len(packet) <= 12 || size < p.avg*0.45
	p.avg = p.avg*0.97 + size*0.03
	p.packets = append(p.packets, append([]byte(nil), packet...))
	p.samples += n
	if quiet {
		p.quiet += n
	} else {
		p.quiet = 0
		p.voiced += n
	}

	cut := (p.samples >= minPiece && p.quiet >= pauseLen) || p.samples >= maxPiece
	if !cut {
		m.mu.Unlock()
		return
	}
	delete(m.pieces, room)
	if p.voiced < minVoiced {
		m.mu.Unlock()
		return
	}
	j := job{room: room, prompt: m.prompt[room], langs: append([]string(nil), langs...)}
	m.mu.Unlock()

	var b bytes.Buffer
	if err := writeOgg(&b, p.packets); err != nil {
		return
	}
	j.ogg = b.Bytes()

	select {
	case m.jobs <- j:
	default:
		// behind: newer speech matters more than old
		m.logger.Debug().Str("room", room).Msg("speech server is behind, skipping a piece")
	}
}

func (m *Manager) work() {
	for {
		select {
		case <-m.ctx.Done():
			return
		case j := <-m.jobs:
			m.run(j)
		}
	}
}

func (m *Manager) run(j job) {
	ctx, cancel := context.WithTimeout(m.ctx, jobTimeout)
	defer cancel()
	cfg := m.Config()
	if !cfg.Enabled() {
		return
	}

	res, err := m.client.transcribe(ctx, cfg, j.ogg, j.prompt, false)
	if err != nil {
		m.logger.Warn().Err(err).Str("room", j.room).Msg("speech to text failed")
		return
	}
	text := res.clean()
	if text == "" {
		return
	}
	src := languageCode(res.Language)

	m.mu.Lock()
	if _, ok := m.wanted[j.room]; ok {
		m.prompt[j.room] = lastWords(text, 200)
	}
	m.mu.Unlock()

	sent := map[string]bool{}
	for _, lang := range j.langs {
		if sent[lang] {
			continue
		}
		sent[lang] = true
		switch {
		case lang == Original || lang == src:
			m.emit(j.room, lang, text)
		case cfg.TranslateURL != "":
			out, err := m.client.translate(ctx, cfg, text, src, lang)
			if err != nil {
				m.logger.Warn().Err(err).Str("lang", lang).Msg("translation failed")
				continue
			}
			m.emit(j.room, lang, out)
		case lang == "en":
			// whisper translates to English by itself
			tr, err := m.client.transcribe(ctx, cfg, j.ogg, "", true)
			if err != nil {
				m.logger.Warn().Err(err).Msg("translation to English failed")
				continue
			}
			if out := tr.clean(); out != "" {
				m.emit(j.room, lang, out)
			}
		}
	}
}

// Languages are what viewers can pick: the original, and what the
// translation server offers, or English without one.
func (m *Manager) Languages(ctx context.Context) []Language {
	cfg := m.Config()
	out := []Language{{Code: Original, Name: "Original language"}}
	if !cfg.Enabled() {
		return nil
	}
	if cfg.TranslateURL == "" {
		return append(out, Language{Code: "en", Name: "English"})
	}
	langs, err := m.client.languages(ctx, cfg)
	if err != nil {
		m.logger.Warn().Err(err).Msg("unable to list translation languages")
		return append(out, Language{Code: "en", Name: "English"})
	}
	return append(out, langs...)
}

// Test transcribes a short sample, and translates it to lang when set.
func (m *Manager) Test(ctx context.Context, cfg Config, lang string) (text, translated string, err error) {
	res, err := m.client.transcribe(ctx, cfg, sampleOgg, "", false)
	if err != nil {
		return "", "", err
	}
	text = res.clean()
	if lang != "" && cfg.TranslateURL != "" {
		translated, err = m.client.translate(ctx, cfg, text, languageCode(res.Language), lang)
	}
	return text, translated, err
}

func (m *Manager) Shutdown() {
	m.cancel()
}

// lastWords is the end of s, at most n bytes, from a word start
func lastWords(s string, n int) string {
	if len(s) <= n {
		return s
	}
	s = s[len(s)-n:]
	if i := bytes.IndexByte([]byte(s), ' '); i >= 0 {
		s = s[i+1:]
	}
	return s
}
