package branding

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strings"
	"time"
)

var (
	indexTitleRegex   = regexp.MustCompile(`(?is)<title>.*?</title>`)
	indexFaviconRegex = regexp.MustCompile(`(?i)<link[^>]+rel="icon"[^>]*>\s*`)
	indexFontRegex    = regexp.MustCompile(`(?i)<link[^>]+fonts\.googleapis\.com[^>]*>\s*`)
)

const (
	headMarker = "<!--branding:head-->"
	bodyMarker = "<!--branding:body-->"
)

func escapeStyle(s string) string {
	return strings.ReplaceAll(s, "</", `<\/`)
}

// bootCSS paints the page before the app loads to avoid a flash of
// unbranded content.
func bootCSS(b Branding) string {
	var sb strings.Builder
	p := b.Theme.Dark
	if b.Theme.Mode == "light" {
		p = b.Theme.Light
	}

	fmt.Fprintf(&sb, "html,body{background:%s;color:%s;", p.Background, p.Text)
	if b.Theme.FontFamily != "" {
		fmt.Fprintf(&sb, "font-family:%s;", b.Theme.FontFamily)
	}
	sb.WriteString("}")

	if b.Theme.Mode == "system" {
		l := b.Theme.Light
		fmt.Fprintf(&sb, "@media (prefers-color-scheme: light){html,body{background:%s;color:%s}}", l.Background, l.Text)
	}

	return sb.String()
}

// RenderIndex injects branding into the client index.html.
func (m *Manager) RenderIndex(raw []byte) []byte {
	b := m.Get()
	year := time.Now().Year()

	out := raw
	title := "<title>" + html.EscapeString(b.Expand(b.PageTitle, year)) + "</title>"
	out = indexTitleRegex.ReplaceAllLiteral(out, []byte(title))

	if b.Favicon != "" {
		out = indexFaviconRegex.ReplaceAllLiteral(out, nil)
	}
	if b.Theme.FontURL != "" || b.Theme.FontFamily != "" {
		out = indexFontRegex.ReplaceAllLiteral(out, nil)
	}

	var head strings.Builder
	if b.Favicon != "" {
		fmt.Fprintf(&head, `<link rel="icon" href="%s">`, html.EscapeString(b.Favicon))
	}
	if b.ThemeColor != "" {
		fmt.Fprintf(&head, `<meta name="theme-color" content="%s">`, html.EscapeString(b.ThemeColor))
	}
	if b.Description != "" {
		fmt.Fprintf(&head, `<meta name="description" content="%s">`, html.EscapeString(b.Description))
		fmt.Fprintf(&head, `<meta property="og:description" content="%s">`, html.EscapeString(b.Description))
	}
	fmt.Fprintf(&head, `<meta property="og:title" content="%s">`, html.EscapeString(b.Expand(b.PageTitle, year)))
	if b.Logo != "" {
		fmt.Fprintf(&head, `<meta property="og:image" content="%s">`, html.EscapeString(b.Logo))
	}
	if b.Theme.FontURL != "" {
		fmt.Fprintf(&head, `<link rel="stylesheet" href="%s">`, html.EscapeString(b.Theme.FontURL))
	}

	fmt.Fprintf(&head, `<style id="branding-boot">%s</style>`, escapeStyle(bootCSS(b)))

	// initial branding for the app, json.Marshal escapes <, > and &
	data, _ := json.Marshal(b)
	fmt.Fprintf(&head, `<script>window.__BRANDING__=%s;</script>`, data)

	if b.CustomCSS != "" {
		fmt.Fprintf(&head, `<style id="branding-custom">%s</style>`, escapeStyle(b.CustomCSS))
	}
	head.WriteString(b.CustomHead)

	var body strings.Builder
	if b.CustomJS != "" {
		fmt.Fprintf(&body, `<script>%s</script>`, strings.ReplaceAll(b.CustomJS, "</script", `<\/script`))
	}

	if bytes.Contains(out, []byte(headMarker)) {
		out = bytes.Replace(out, []byte(headMarker), []byte(head.String()), 1)
	} else {
		out = bytes.Replace(out, []byte("</head>"), []byte(head.String()+"</head>"), 1)
	}

	if bytes.Contains(out, []byte(bodyMarker)) {
		out = bytes.Replace(out, []byte(bodyMarker), []byte(body.String()), 1)
	} else {
		out = bytes.Replace(out, []byte("</body>"), []byte(body.String()+"</body>"), 1)
	}

	return out
}
