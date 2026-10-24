package captions

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// sampleOgg is a short spoken sentence, to test a speech server with
//
//go:embed sample.ogg
var sampleOgg []byte

const transcriptionPath = "/v1/audio/transcriptions"

type serverClient struct {
	http *http.Client

	mu        sync.Mutex
	langs     []Language
	langsURL  string
	langsTime time.Time
}

func newServerClient() *serverClient {
	return &serverClient{http: &http.Client{Timeout: jobTimeout}}
}

type transcription struct {
	Language string `json:"language"`
	Text     string `json:"text"`
	Segments []struct {
		Text         string   `json:"text"`
		NoSpeechProb *float64 `json:"no_speech_prob"`
	} `json:"segments"`
}

// what whisper writes for sounds that are not speech, and lines it makes up
// from silence or music
var (
	nonSpeech     = regexp.MustCompile(`\[[^\]]*\]|\([^)]*\)|\*[^*]*\*|♪+|♫+`)
	hallucination = map[string]bool{
		"you": true, "thank you.": true, "thank you": true, "thanks for watching!": true,
		"thanks for watching.": true, "thank you for watching.": true, "bye.": true,
		"subtitles by the amara.org community": true, ".": true,
	}
)

// clean is the spoken text, without sounds, made up lines and pieces whisper
// itself thinks are not speech
func (t *transcription) clean() string {
	var parts []string
	if len(t.Segments) > 0 {
		for _, s := range t.Segments {
			if s.NoSpeechProb != nil && *s.NoSpeechProb > 0.6 {
				continue
			}
			parts = append(parts, s.Text)
		}
	} else {
		parts = []string{t.Text}
	}
	text := nonSpeech.ReplaceAllString(strings.Join(parts, " "), "")
	text = strings.Join(strings.Fields(text), " ")
	if hallucination[strings.ToLower(text)] || strings.Trim(text, " .,!?-…") == "" {
		return ""
	}
	return text
}

func endpoint(base string) string {
	base = strings.TrimRight(base, "/")
	if strings.HasSuffix(base, transcriptionPath) {
		return base
	}
	return base + transcriptionPath
}

// transcribe sends audio to the speech server. translate asks whisper for
// English instead of the language spoken.
func (c *serverClient) transcribe(ctx context.Context, cfg Config, ogg []byte, prompt string, translate bool) (*transcription, error) {
	var body bytes.Buffer
	mw := multipart.NewWriter(&body)
	fw, err := mw.CreateFormFile("file", "speech.ogg")
	if err != nil {
		return nil, err
	}
	fw.Write(ogg)
	model := cfg.Model
	if model == "" {
		model = "whisper-1"
	}
	mw.WriteField("model", model)
	mw.WriteField("response_format", "verbose_json")
	mw.WriteField("temperature", "0")
	if prompt != "" {
		mw.WriteField("prompt", prompt)
	}
	if translate {
		mw.WriteField("translate", "true")
	}
	mw.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint(cfg.URL), &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("speech server: %s: %s", res.Status, strings.TrimSpace(string(data)))
	}
	var t transcription
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("speech server: %w", err)
	}
	return &t, nil
}

// translate uses the LibreTranslate API
func (c *serverClient) translate(ctx context.Context, cfg Config, text, source, target string) (string, error) {
	if source == "" {
		source = "auto"
	}
	payload, _ := json.Marshal(map[string]string{
		"q":       text,
		"source":  source,
		"target":  target,
		"format":  "text",
		"api_key": cfg.TranslateKey,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(cfg.TranslateURL, "/")+"/translate", bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	var out struct {
		Text  string `json:"translatedText"`
		Error string `json:"error"`
	}
	json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&out)
	if res.StatusCode != http.StatusOK {
		if out.Error == "" {
			out.Error = res.Status
		}
		return "", fmt.Errorf("translation server: %s", out.Error)
	}
	return strings.TrimSpace(out.Text), nil
}

// Language is a language viewers can pick.
type Language struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// languages are the targets of the translation server, cached a while
func (c *serverClient) languages(ctx context.Context, cfg Config) ([]Language, error) {
	c.mu.Lock()
	if c.langsURL == cfg.TranslateURL && time.Since(c.langsTime) < 10*time.Minute {
		defer c.mu.Unlock()
		return c.langs, nil
	}
	c.mu.Unlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(cfg.TranslateURL, "/")+"/languages", nil)
	if err != nil {
		return nil, err
	}
	res, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("translation server: %s", res.Status)
	}
	var langs []Language
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&langs); err != nil {
		return nil, err
	}
	sort.Slice(langs, func(i, j int) bool { return langs[i].Name < langs[j].Name })

	c.mu.Lock()
	c.langs, c.langsURL, c.langsTime = langs, cfg.TranslateURL, time.Now()
	c.mu.Unlock()
	return langs, nil
}

// whisper names languages in English, translation servers use ISO codes
var whisperLanguages = map[string]string{
	"english": "en", "chinese": "zh", "german": "de", "spanish": "es", "russian": "ru",
	"korean": "ko", "french": "fr", "japanese": "ja", "portuguese": "pt", "turkish": "tr",
	"polish": "pl", "catalan": "ca", "dutch": "nl", "arabic": "ar", "swedish": "sv",
	"italian": "it", "indonesian": "id", "hindi": "hi", "finnish": "fi", "vietnamese": "vi",
	"hebrew": "he", "ukrainian": "uk", "greek": "el", "malay": "ms", "czech": "cs",
	"romanian": "ro", "danish": "da", "hungarian": "hu", "tamil": "ta", "norwegian": "nb",
	"thai": "th", "urdu": "ur", "croatian": "hr", "bulgarian": "bg", "lithuanian": "lt",
	"latin": "la", "welsh": "cy", "slovak": "sk", "persian": "fa", "latvian": "lv",
	"bengali": "bn", "serbian": "sr", "azerbaijani": "az", "slovenian": "sl", "estonian": "et",
	"tagalog": "tl", "filipino": "tl", "irish": "ga", "albanian": "sq", "basque": "eu",
	"galician": "gl", "esperanto": "eo", "kazakh": "kk", "icelandic": "is",
}

// languageCode is the ISO code of a language whisper reports, by name or
// already as a code
func languageCode(name string) string {
	n := strings.ToLower(strings.TrimSpace(name))
	if code, ok := whisperLanguages[n]; ok {
		return code
	}
	if len(n) == 2 {
		return n
	}
	return ""
}
