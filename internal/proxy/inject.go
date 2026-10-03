package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/m1k1o/neko-rooms/internal/auth"
	"github.com/m1k1o/neko-rooms/internal/branding"
)

const maxInjectSize = 4 << 20

var (
	titleRegex   = regexp.MustCompile(`(?is)<title>.*?</title>`)
	headEndRegex = regexp.MustCompile(`(?i)</head>`)
	bodyEndRegex = regexp.MustCompile(`(?i)</body>`)
)

func (p *ProxyManagerCtx) signedIn(r *http.Request) bool {
	u, _ := p.auth.UserFromRequest(r)
	return u != nil
}

// stripSessionCookie makes sure our session never reaches room containers.
func stripSessionCookie(r *http.Request) {
	cookies := r.Cookies()
	if len(cookies) == 0 {
		return
	}

	r.Header.Del("Cookie")
	for _, c := range cookies {
		if c.Name != auth.CookieName {
			r.AddCookie(c)
		}
	}
}

func isPageRequest(r *http.Request) bool {
	return r.Method == http.MethodGet && strings.Contains(r.Header.Get("Accept"), "text/html")
}

func (p *ProxyManagerCtx) injectEnabled() bool {
	return p.branding != nil && p.branding.Get().Rooms.Inject
}

// injectBranding rewrites room HTML pages to carry instance branding.
func (p *ProxyManagerCtx) injectBranding(res *http.Response, roomName string) error {
	if !p.injectEnabled() || res.Request == nil || !isPageRequest(res.Request) {
		return nil
	}
	if res.StatusCode != http.StatusOK || !strings.HasPrefix(res.Header.Get("Content-Type"), "text/html") {
		return nil
	}
	if enc := res.Header.Get("Content-Encoding"); enc != "" && enc != "identity" {
		return nil
	}

	body, err := io.ReadAll(io.LimitReader(res.Body, maxInjectSize+1))
	res.Body.Close()
	if err != nil {
		return err
	}

	if len(body) <= maxInjectSize {
		body = p.injectHTML(body, p.roomTitle(roomName))
	}

	res.Body = io.NopCloser(bytes.NewReader(body))
	res.ContentLength = int64(len(body))
	res.Header.Set("Content-Length", strconv.Itoa(len(body)))
	res.Header.Del("ETag")
	return nil
}

// roomTitle returns the community title of a room, or its name.
func (p *ProxyManagerCtx) roomTitle(name string) string {
	if p.auth != nil {
		if meta, err := p.auth.Store().GetRoomMeta(name); err == nil && meta.Title != "" {
			return meta.Title
		}
	}
	return name
}

// roomThemeCSS maps the site's dark palette onto the neko client, which
// only has a dark theme. Selectors follow the neko client components.
func roomThemeCSS(pal branding.Palette, radius int) string {
	return fmt.Sprintf(`
html, body, #neko .neko-main .header-container, #neko .neko-main .room-container { background: %[1]s !important; }
#neko { accent-color: %[3]s; }
#neko .neko-menu { background-color: %[2]s !important; }
#neko .neko-menu .tabs-container { background: %[1]s !important; }
#neko .neko-menu .tabs-container ul li { background: %[4]s !important; }
#neko .neko-menu .tabs-container ul li.active { background: %[2]s !important; }
.connect .window { background: %[2]s !important; border-radius: %[5]dpx !important; }
.connect .window .message input { background: %[1]s !important; }
.connect .window .message button { background: %[3]s !important; }
.connect .window .message button.oauth-login { background: %[1]s !important; border-color: %[3]s !important; }
.connect .window .loader .bounce1, .connect .window .loader .bounce2 { background-color: %[3]s !important; }
.volume input[type='range']::-webkit-slider-runnable-track { background: %[3]s !important; }
.volume input[type='range']::-moz-range-track { background: %[3]s !important; }
.switch input[type='checkbox']:checked + span { background-color: %[3]s !important; }
`, pal.Background, pal.Surface, pal.Primary, pal.Secondary, radius)
}

// logoScript swaps the n.eko logo and name for the site's, also when the
// neko client re-renders those parts.
const logoScript = `(function () {
  var B = %s;
  function apply() {
    document.querySelectorAll('.header .neko, .connect .window .logo, .unsupported .logo').forEach(function (el) {
      if (el.getAttribute('data-nr-branded')) return;
      el.setAttribute('data-nr-branded', '1');
      var img = el.querySelector('img'), span = el.querySelector('span');
      if (img) {
        if (B.logo) { img.src = B.logo; img.alt = B.name; img.style.objectFit = 'contain'; }
        else { img.style.display = 'none'; }
      }
      if (span) span.textContent = B.name;
      if (el.tagName === 'A') { el.href = B.home; el.removeAttribute('target'); }
      el.title = B.name;
    });
  }
  new MutationObserver(apply).observe(document.documentElement, { childList: true, subtree: true });
  apply();
})();`

func (p *ProxyManagerCtx) injectHTML(body []byte, roomTitle string) []byte {
	b := p.branding.Get()
	rb := b.Rooms
	year := time.Now().Year()

	var head strings.Builder

	if rb.PageTitle != "" {
		titleText := b.Expand(strings.ReplaceAll(rb.PageTitle, "{room}", roomTitle), year)
		title := "<title>" + html.EscapeString(titleText) + "</title>"
		if titleRegex.Match(body) {
			body = titleRegex.ReplaceAllLiteral(body, []byte(title))
		} else {
			head.WriteString(title)
		}
	}

	favicon := rb.Favicon
	if favicon == "" {
		favicon = b.Favicon
	}
	if favicon != "" {
		fmt.Fprintf(&head, `<link rel="icon" href="%s">`, html.EscapeString(favicon))
	}
	if b.ThemeColor != "" {
		fmt.Fprintf(&head, `<meta name="theme-color" content="%s">`, html.EscapeString(b.ThemeColor))
	}

	css := ""
	if rb.SiteColors {
		css += roomThemeCSS(b.Theme.Dark, b.Theme.BorderRadius)
	}
	css += rb.CustomCSS
	if css != "" {
		head.WriteString("<style>")
		head.WriteString(strings.ReplaceAll(css, "</", `<\/`))
		head.WriteString("</style>")
	}
	head.WriteString(rb.CustomHead)

	if loc := headEndRegex.FindIndex(body); loc != nil {
		body = append(body[:loc[0]:loc[0]], append([]byte(head.String()), body[loc[0]:]...)...)
	}

	var scripts strings.Builder
	if rb.ReplaceLogo {
		logo := b.Logo
		if logo == "" {
			logo = b.Favicon
		}
		// json.Marshal escapes <, > and &, safe inside a script tag
		data, _ := json.Marshal(map[string]string{
			"name": b.AppName,
			"logo": logo,
			"home": p.loginURL,
		})
		scripts.WriteString("<script>")
		fmt.Fprintf(&scripts, logoScript, data)
		scripts.WriteString("</script>")
	}
	if rb.CustomJS != "" {
		scripts.WriteString("<script>" + strings.ReplaceAll(rb.CustomJS, "</script", `<\/script`) + "</script>")
	}

	if scripts.Len() > 0 {
		if loc := bodyEndRegex.FindIndex(body); loc != nil {
			body = append(body[:loc[0]:loc[0]], append([]byte(scripts.String()), body[loc[0]:]...)...)
		} else {
			body = append(body, scripts.String()...)
		}
	}

	return body
}
