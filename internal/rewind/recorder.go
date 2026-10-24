package rewind

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"github.com/pion/interceptor"
	"github.com/pion/rtcp"
	"github.com/pion/rtp/codecs"
	"github.com/pion/webrtc/v4"
	"github.com/pion/webrtc/v4/pkg/media/samplebuilder"
	"github.com/rs/zerolog"

	"github.com/m1k1o/neko-rooms/internal/types"
)

// The recorder joins a room like a viewer, under types.RewindRecorderName,
// and writes the video and sound neko already encodes for everyone to the
// buffer. Nothing is decoded or encoded again. Without a buffer it only
// listens: the sound goes to audio, for subtitles, and no video is sent.

// errUnsupported means the room can not be recorded, e.g. an older neko
// or a video codec other than VP8
type errUnsupported struct{ reason string }

func (e errUnsupported) Error() string { return "unsupported: " + e.reason }

type recorder struct {
	logger zerolog.Logger
	host   string            // room web address, host:port
	pass   string            // user password of the room
	buf    *buffer           // nil: sound only
	audio  func(data []byte) // each Opus packet, may be nil

	alive atomic.Int64 // unix ms of the last frame
	wsMu  sync.Mutex
	ws    *websocket.Conn
}

type wsMessage struct {
	Event   string          `json:"event"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

func (r *recorder) send(event string, payload any) error {
	msg := wsMessage{Event: event}
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		msg.Payload = data
	}
	r.wsMu.Lock()
	defer r.wsMu.Unlock()
	r.ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return r.ws.WriteJSON(msg)
}

// login signs in to neko and returns the session cookie or token
func (r *recorder) login(ctx context.Context) (*http.Cookie, string, error) {
	body, _ := json.Marshal(map[string]string{
		"username": types.RewindRecorderName,
		"password": r.pass,
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://"+r.host+"/api/login", bytes.NewReader(body))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return nil, "", errUnsupported{"no neko v3 api"}
	}
	if res.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("login: %s", res.Status)
	}
	var data struct {
		Token string `json:"token"`
	}
	json.NewDecoder(res.Body).Decode(&data)
	for _, c := range res.Cookies() {
		if c.Name == "NEKO_SESSION" {
			return c, "", nil
		}
	}
	if data.Token == "" {
		return nil, "", errors.New("login: no session")
	}
	return nil, data.Token, nil
}

func (r *recorder) logout(cookie *http.Cookie, token string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	u := "http://" + r.host + "/api/logout"
	if token != "" {
		u += "?token=" + url.QueryEscape(token)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, nil)
	if err != nil {
		return
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	if res, err := http.DefaultClient.Do(req); err == nil {
		res.Body.Close()
	}
}

// run records until ctx ends or the connection fails
func (r *recorder) run(ctx context.Context) error {
	ctx, cancel := context.WithCancelCause(ctx)
	defer cancel(nil)

	cookie, token, err := r.login(ctx)
	if err != nil {
		return err
	}
	defer r.logout(cookie, token)

	wsURL := "ws://" + r.host + "/api/ws"
	if token != "" {
		wsURL += "?token=" + url.QueryEscape(token)
	}
	header := http.Header{}
	if cookie != nil {
		header.Set("Cookie", cookie.String())
	}
	dialer := websocket.Dialer{HandshakeTimeout: 10 * time.Second}
	ws, _, err := dialer.DialContext(ctx, wsURL, header)
	if err != nil {
		return fmt.Errorf("websocket: %w", err)
	}
	r.ws = ws
	defer ws.Close()

	// the address the room is reached on from here, its candidates are
	// tried there too: they carry the address viewers use, which is not
	// always reachable from inside the host
	roomIPs := lookupHost(ctx, r.host)

	pc, err := newPeerConnection()
	if err != nil {
		return err
	}
	defer pc.Close()

	pc.OnICECandidate(func(c *webrtc.ICECandidate) {
		if c == nil {
			return
		}
		r.send("signal/candidate", c.ToJSON())
	})
	pc.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		r.logger.Debug().Str("state", s.String()).Msg("recorder connection")
		if s == webrtc.PeerConnectionStateFailed || s == webrtc.PeerConnectionStateClosed {
			cancel(errors.New("connection " + s.String()))
		}
	})
	pc.OnTrack(func(t *webrtc.TrackRemote, rec *webrtc.RTPReceiver) {
		go func() {
			if err := r.track(ctx, pc, t); err != nil {
				cancel(err)
			}
		}()
	})

	// neko sends candidates as it finds them, also before its offer
	var early []webrtc.ICECandidateInit
	addCandidate := func(c webrtc.ICECandidateInit) {
		if pc.RemoteDescription() == nil {
			early = append(early, c)
			return
		}
		pc.AddICECandidate(c)
		for _, alt := range rewriteCandidate(c.Candidate, roomIPs) {
			pc.AddICECandidate(webrtc.ICECandidateInit{Candidate: alt, SDPMid: c.SDPMid, SDPMLineIndex: c.SDPMLineIndex})
		}
	}

	answer := func(sdp string) error {
		if err := pc.SetRemoteDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: sdp}); err != nil {
			return err
		}
		for _, c := range early {
			addCandidate(c)
		}
		early = nil
		ans, err := pc.CreateAnswer(nil)
		if err != nil {
			return err
		}
		if err := pc.SetLocalDescription(ans); err != nil {
			return err
		}
		// candidates in the offer itself
		for _, line := range strings.Split(sdp, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "a=candidate:") {
				zero := uint16(0)
				for _, alt := range rewriteCandidate(strings.TrimPrefix(line, "a="), roomIPs) {
					pc.AddICECandidate(webrtc.ICECandidateInit{Candidate: alt, SDPMLineIndex: &zero})
				}
			}
		}
		return r.send("signal/answer", map[string]string{"sdp": ans.SDP})
	}

	// neko expects a sign of life now and then
	go func() {
		t := time.NewTicker(10 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				// a clean close: neko waits for a reconnect otherwise
				r.wsMu.Lock()
				ws.WriteControl(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""), time.Now().Add(time.Second))
				r.wsMu.Unlock()
				ws.Close()
				return
			case <-t.C:
				r.send("client/heartbeat", nil)
			}
		}
	}()

	// nothing arrives: the room can not be reached over webrtc from here
	r.alive.Store(time.Now().UnixMilli())
	go func() {
		t := time.NewTicker(5 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				if time.Now().UnixMilli()-r.alive.Load() > 30000 {
					cancel(errors.New("nothing arrives from the room"))
					return
				}
			}
		}
	}()

	for {
		var msg wsMessage
		if err := ws.ReadJSON(&msg); err != nil {
			if cause := context.Cause(ctx); cause != nil {
				return cause
			}
			return fmt.Errorf("websocket: %w", err)
		}

		switch msg.Event {
		case "system/init":
			video := map[string]any{}
			if r.buf == nil {
				video["disabled"] = true
			}
			r.send("signal/request", map[string]any{"video": video, "audio": map[string]any{}})
		case "signal/provide", "signal/offer", "signal/restart":
			var p struct {
				SDP string `json:"sdp"`
			}
			if err := json.Unmarshal(msg.Payload, &p); err != nil {
				return err
			}
			if err := answer(p.SDP); err != nil {
				return fmt.Errorf("signal: %w", err)
			}
		case "signal/candidate":
			var c webrtc.ICECandidateInit
			if err := json.Unmarshal(msg.Payload, &c); err == nil {
				addCandidate(c)
			}
		case "system/disconnect":
			return errors.New("disconnected by the room")
		}
	}
}

func newPeerConnection() (*webrtc.PeerConnection, error) {
	m := &webrtc.MediaEngine{}
	if err := m.RegisterDefaultCodecs(); err != nil {
		return nil, err
	}
	i := &interceptor.Registry{}
	if err := webrtc.RegisterDefaultInterceptors(m, i); err != nil {
		return nil, err
	}
	api := webrtc.NewAPI(webrtc.WithMediaEngine(m), webrtc.WithInterceptorRegistry(i))
	return api.NewPeerConnection(webrtc.Configuration{})
}

// track writes a track of the room to the buffer
func (r *recorder) track(ctx context.Context, pc *webrtc.PeerConnection, t *webrtc.TrackRemote) error {
	mime := strings.ToLower(t.Codec().MimeType)
	var sb *samplebuilder.SampleBuilder
	video := false
	switch mime {
	case strings.ToLower(webrtc.MimeTypeVP8):
		if r.buf == nil {
			return nil // sound only
		}
		video = true
		sb = samplebuilder.New(128, &codecs.VP8Packet{}, t.Codec().ClockRate)
		// a keyframe right away instead of within the next second
		pc.WriteRTCP([]rtcp.Packet{&rtcp.PictureLossIndication{MediaSSRC: uint32(t.SSRC())}})
	case strings.ToLower(webrtc.MimeTypeOpus):
		sb = samplebuilder.New(32, &codecs.OpusPacket{}, t.Codec().ClockRate)
	default:
		if t.Kind() == webrtc.RTPCodecTypeVideo {
			return errUnsupported{"video codec " + mime}
		}
		return nil // other sound codecs: video only
	}

	c := &clock{rate: int64(t.Codec().ClockRate)}
	for {
		pkt, _, err := t.ReadRTP()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			return err
		}
		sb.Push(pkt)
		for s := sb.Pop(); s != nil; s = sb.Pop() {
			r.alive.Store(time.Now().UnixMilli())
			if !video && r.audio != nil {
				r.audio(s.Data)
			}
			if r.buf == nil {
				continue
			}
			ts := c.at(s.PacketTimestamp, r.buf.now())
			if err := r.buf.write(frame{video: video, ts: ts, data: s.Data}); err != nil {
				return err
			}
		}
	}
}

// lookupHost resolves the host of a host:port address
func lookupHost(ctx context.Context, hostport string) []string {
	host, _, err := net.SplitHostPort(hostport)
	if err != nil {
		host = hostport
	}
	if ip := net.ParseIP(host); ip != nil {
		return []string{ip.String()}
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupHost(ctx, host)
	if err != nil {
		return nil
	}
	return addrs
}

// rewriteCandidate is the candidate with its address swapped for each of
// ips, for UDP and TCP candidates on other addresses
func rewriteCandidate(c string, ips []string) []string {
	// candidate:<foundation> <component> <proto> <priority> <ip> <port> typ ...
	f := strings.Fields(c)
	if len(f) < 8 || !strings.HasPrefix(f[0], "candidate:") {
		return nil
	}
	var out []string
	for _, ip := range ips {
		if ip == f[4] || strings.Contains(ip, ":") != strings.Contains(f[4], ":") {
			continue
		}
		g := append([]string(nil), f...)
		g[0] += "r"
		g[4] = ip
		out = append(out, strings.Join(g, " "))
	}
	return out
}
