package proxy

import (
	"bytes"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/m1k1o/neko-rooms/internal/auth"
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
func (p *ProxyManagerCtx) injectBranding(res *http.Response) error {
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
		body = p.injectHTML(body)
	}

	res.Body = io.NopCloser(bytes.NewReader(body))
	res.ContentLength = int64(len(body))
	res.Header.Set("Content-Length", strconv.Itoa(len(body)))
	res.Header.Del("ETag")
	return nil
}

func (p *ProxyManagerCtx) injectHTML(body []byte) []byte {
	b := p.branding.Get()
	rb := b.Rooms

	if rb.PageTitle != "" {
		title := "<title>" + html.EscapeString(b.Expand(rb.PageTitle, time.Now().Year())) + "</title>"
		if titleRegex.Match(body) {
			body = titleRegex.ReplaceAllLiteral(body, []byte(title))
		} else {
			rb.CustomHead = title + rb.CustomHead
		}
	}

	var head strings.Builder
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
	if rb.CustomCSS != "" {
		head.WriteString("<style>")
		head.WriteString(strings.ReplaceAll(rb.CustomCSS, "</", `<\/`))
		head.WriteString("</style>")
	}
	head.WriteString(rb.CustomHead)

	if loc := headEndRegex.FindIndex(body); loc != nil {
		body = append(body[:loc[0]:loc[0]], append([]byte(head.String()), body[loc[0]:]...)...)
	}

	if rb.CustomJS != "" {
		script := "<script>" + strings.ReplaceAll(rb.CustomJS, "</script", `<\/script`) + "</script>"
		if loc := bodyEndRegex.FindIndex(body); loc != nil {
			body = append(body[:loc[0]:loc[0]], append([]byte(script), body[loc[0]:]...)...)
		} else {
			body = append(body, script...)
		}
	}

	return body
}
