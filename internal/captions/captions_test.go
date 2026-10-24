package captions

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestOpusSamples(t *testing.T) {
	cases := []struct {
		p    []byte
		want int
	}{
		{[]byte{31 << 3}, 960},       // CELT 20 ms
		{[]byte{1 << 3}, 960},        // SILK 20 ms
		{[]byte{3 << 3}, 2880},       // SILK 60 ms
		{[]byte{31<<3 | 1, 0}, 1920}, // two frames
		{[]byte{31<<3 | 3, 3}, 2880}, // three frames, code 3
		{[]byte{16 << 3}, 120},       // CELT 2.5 ms
		{nil, 0},
	}
	for _, c := range cases {
		if got := opusSamples(c.p); got != c.want {
			t.Errorf("opusSamples(%x) = %d, want %d", c.p, got, c.want)
		}
	}
}

func TestOggPages(t *testing.T) {
	packets := make([][]byte, 120)
	for i := range packets {
		packets[i] = bytes.Repeat([]byte{31 << 3}, 80)
	}
	var b bytes.Buffer
	if err := writeOgg(&b, packets); err != nil {
		t.Fatal(err)
	}
	data := b.Bytes()

	// walk the pages: checksums, sequence, flags and the last granule
	var seq uint32
	var granule uint64
	var last byte
	for len(data) > 0 {
		if !bytes.HasPrefix(data, []byte("OggS")) {
			t.Fatalf("page %d does not start with OggS", seq)
		}
		n := int(data[26])
		size := 27 + n
		for _, l := range data[27 : 27+n] {
			size += int(l)
		}
		page := append([]byte(nil), data[:size]...)
		crc := binary.LittleEndian.Uint32(page[22:])
		binary.LittleEndian.PutUint32(page[22:], 0)
		if oggCRC(page) != crc {
			t.Fatalf("page %d checksum", seq)
		}
		if got := binary.LittleEndian.Uint32(page[18:]); got != seq {
			t.Fatalf("page sequence %d, want %d", got, seq)
		}
		if seq == 0 && page[5] != 0x02 {
			t.Fatal("first page is not the start of the stream")
		}
		granule = binary.LittleEndian.Uint64(page[6:])
		last = page[5]
		seq++
		data = data[size:]
	}
	if last != 0x04 || granule != 120*960 {
		t.Fatalf("last page flags %x granule %d", last, granule)
	}
}

func TestOggCRC(t *testing.T) {
	// the Ogg CRC of "123456789" (CRC-32/MPEG-2 without the final xor)
	if got := oggCRC([]byte("123456789")); got != 0x89a1897f {
		t.Fatalf("crc = %x", got)
	}
}

func TestClean(t *testing.T) {
	p := func(f float64) *float64 { return &f }
	tr := transcription{}
	tr.Segments = append(tr.Segments,
		struct {
			Text         string   `json:"text"`
			NoSpeechProb *float64 `json:"no_speech_prob"`
		}{" Hello [Music] there", p(0.1)},
		struct {
			Text         string   `json:"text"`
			NoSpeechProb *float64 `json:"no_speech_prob"`
		}{" made up", p(0.9)},
		struct {
			Text         string   `json:"text"`
			NoSpeechProb *float64 `json:"no_speech_prob"`
		}{" ♪ friend.", nil},
	)
	if got := tr.clean(); got != "Hello there friend." {
		t.Fatalf("clean = %q", got)
	}
	for _, s := range []string{"[BLANK_AUDIO]", " Thank you.", "(music)", "..."} {
		if got := (&transcription{Text: s}).clean(); got != "" {
			t.Errorf("clean(%q) = %q, want nothing", s, got)
		}
	}
}

func TestLanguageCode(t *testing.T) {
	for in, want := range map[string]string{"english": "en", "Spanish": "es", "de": "de", "klingon": ""} {
		if got := languageCode(in); got != want {
			t.Errorf("languageCode(%q) = %q, want %q", in, got, want)
		}
	}
}

// speech: 3 seconds of loud packets, then a pause
func feedSpeech(m *Manager, room string) {
	loud := bytes.Repeat([]byte{31 << 3}, 120)
	quiet := []byte{31 << 3, 0, 0}
	for i := 0; i < 160; i++ {
		m.Audio(room, loud)
	}
	for i := 0; i < 20; i++ {
		m.Audio(room, quiet)
	}
}

func TestPipeline(t *testing.T) {
	var mu sync.Mutex
	var requests []string
	speech := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != transcriptionPath {
			http.NotFound(w, r)
			return
		}
		r.ParseMultipartForm(1 << 20)
		f, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "no file", 400)
			return
		}
		head := make([]byte, 4)
		f.Read(head)
		mu.Lock()
		requests = append(requests, string(head)+" translate="+r.FormValue("translate")+" format="+r.FormValue("response_format"))
		mu.Unlock()
		if r.FormValue("translate") == "true" {
			json.NewEncoder(w).Encode(map[string]any{"language": "english", "text": " Good evening."})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{
			"language": "spanish",
			"text":     " Buenas noches.",
			"segments": []map[string]any{{"text": " Buenas noches.", "no_speech_prob": 0.05}},
		})
	}))
	defer speech.Close()

	type line struct{ lang, text string }
	lines := make(chan line, 10)
	m := New(func(room, lang, text string) { lines <- line{lang, text} })
	defer m.Shutdown()
	m.Configure(Config{URL: speech.URL})

	// nobody wants subtitles: nothing is sent
	feedSpeech(m, "r")
	time.Sleep(50 * time.Millisecond)
	if len(requests) != 0 {
		t.Fatal("transcribed without anyone asking")
	}

	m.Want("r", []string{Original, "en", "fr"})
	// silence alone is never sent
	for i := 0; i < 600; i++ {
		m.Audio("r", []byte{31 << 3, 0})
	}
	feedSpeech(m, "r")

	var got []string
	timeout := time.After(3 * time.Second)
	for len(got) < 2 {
		select {
		case l := <-lines:
			got = append(got, l.lang+": "+l.text)
		case <-timeout:
			t.Fatalf("lines = %v", got)
		}
	}
	sort.Strings(got)
	// French needs a translation server, there is none
	if strings.Join(got, "|") != "en: Good evening.|orig: Buenas noches." {
		t.Fatalf("lines = %v", got)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(requests) != 2 || requests[0] != "OggS translate= format=verbose_json" {
		t.Fatalf("requests = %v", requests)
	}
}

func TestTranslateServer(t *testing.T) {
	lt := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/translate":
			var in map[string]string
			json.NewDecoder(r.Body).Decode(&in)
			json.NewEncoder(w).Encode(map[string]string{"translatedText": in["source"] + ">" + in["target"] + ": " + in["q"]})
		case "/languages":
			json.NewEncoder(w).Encode([]Language{{"fr", "French"}, {"de", "German"}})
		}
	}))
	defer lt.Close()

	c := newServerClient()
	cfg := Config{URL: "http://unused", TranslateURL: lt.URL}
	out, err := c.translate(t.Context(), cfg, "hola", "es", "fr")
	if err != nil || out != "es>fr: hola" {
		t.Fatalf("translate = %q, %v", out, err)
	}
	m := New(func(string, string, string) {})
	defer m.Shutdown()
	m.Configure(cfg)
	langs := m.Languages(t.Context())
	if len(langs) != 3 || langs[0].Code != Original || langs[1].Name != "French" {
		t.Fatalf("languages = %v", langs)
	}
}
