package proxy

import (
	_ "embed"
	"encoding/json"
	"strings"

	"github.com/m1k1o/neko-rooms/internal/community"
	"github.com/m1k1o/neko-rooms/internal/types"
)

//go:embed extras.js
var extrasJS string

// extrasScript is the room overlay with reactions, raised hands, rewind and
// what is playing, for a room.
func (p *ProxyManagerCtx) extrasScript(roomName string) string {
	// json.Marshal escapes <, > and &, safe inside a script tag
	cfg, _ := json.Marshal(map[string]any{
		"api":       p.loginURL + "api/",
		"room":      roomName,
		"reactions": community.Reactions,
		"recorder":  types.RewindRecorderName,
	})
	js := strings.Replace(extrasJS, "__NR_CONFIG__", string(cfg), 1)
	return "<script>" + strings.ReplaceAll(js, "</script", `<\/script`) + "</script>"
}
