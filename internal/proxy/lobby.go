package proxy

import (
	"fmt"
	"html"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/m1k1o/neko-rooms/internal/branding"
	"github.com/m1k1o/neko-rooms/internal/utils"
)

func roomWait(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte(`<script>
	(async function() {
		document.querySelector(".swal2-loader").style.display = 'block'
		document.querySelector(".swal2-loader").style.visibility = 'hidden'

		let lastAttempt = (new Date()).getTime()
		while(true) {
			try {
				lastAttempt = (new Date()).getTime()
				document.querySelector(".swal2-loader").style.visibility = 'visible'

				await fetch("?wait")
				location.href = location.href
			} catch {
				let now = (new Date()).getTime()
				let diff = now - lastAttempt

				// if the gap between last attempt and now
				// is gt 20s, do reconnect immediatly
				if ((now - lastAttempt) > 20*1000) {
					continue
				}

				// wait for 10 sec
				await new Promise(res => window.setTimeout(res, 2500))
				document.querySelector(".swal2-loader").style.visibility = 'hidden'
				await new Promise(res => window.setTimeout(res, 7500))
			}
		}
	}())
	</script>`))
}

// lobbyPage builds the page shell from current branding.
func (p *ProxyManagerCtx) lobbyPage() (utils.Swal2Page, branding.Lobby) {
	page := utils.DefaultSwal2Page()
	if p.branding == nil {
		return page, branding.Default().Lobby
	}

	b := p.branding.Get()
	l := b.Lobby

	page.Title = b.Expand(l.PageTitle, time.Now().Year())
	page.AppName = b.AppName
	page.Favicon = b.Favicon
	page.FontURL = b.Theme.FontURL
	page.PoweredBy = b.Footer.ShowPoweredBy
	if l.ShowLogo {
		page.Logo = b.Logo
	}
	if l.FontFamily != "" {
		page.FontFamily = template.CSS(l.FontFamily)
	}
	if l.BackgroundColor != "" {
		page.BackgroundColor = template.CSS(l.BackgroundColor)
	}
	if l.BackgroundImage != "" {
		page.BackgroundImage = template.CSS(l.BackgroundImage)
	}
	if l.PopupColor != "" {
		page.PopupColor = template.CSS(l.PopupColor)
	}
	if l.TextColor != "" {
		page.TextColor = template.CSS(l.TextColor)
	}
	if l.ButtonColor != "" {
		page.ButtonColor = template.CSS(l.ButtonColor)
	}
	page.CustomCSS = template.CSS(l.CustomCSS)

	return page, l
}

func lobbyMessage(msg string) string {
	var sb strings.Builder
	for _, line := range strings.Split(msg, "\n") {
		sb.WriteString("<div>")
		sb.WriteString(html.EscapeString(line))
		sb.WriteString("</div>")
	}
	return sb.String()
}

func lobbyBody(icon, title, message, actions string) template.HTML {
	return template.HTML(fmt.Sprintf(`
		<div class="swal2-header">
			%s
			<h2 class="swal2-title">%s</h2>
		</div>
		<div class="swal2-content">%s</div>
		<div class="swal2-actions">%s</div>
	`, icon, html.EscapeString(title), lobbyMessage(message), actions))
}

const (
	iconError   = `<div class="swal2-icon swal2-error"><div class="swal2-icon-content">X</div></div>`
	iconWarning = `<div class="swal2-icon swal2-warning"><div class="swal2-icon-content">!</div></div>`
	iconInfo    = `<div class="swal2-icon swal2-info"><div class="swal2-icon-content">i</div></div>`

	loaderHidden  = `<div class="swal2-loader" style="display:none;"></div>`
	loaderVisible = `<div class="swal2-loader"></div>`
	reloadButton  = `<button type="button" onclick="location = location" class="swal2-confirm swal2-styled" style="margin-top: 1.25em">Reload</button>`
)

func (p *ProxyManagerCtx) RoomNotFound(w http.ResponseWriter, r *http.Request, waitEnabled bool) {
	page, l := p.lobbyPage()
	page.Body = lobbyBody(iconError, l.NotFound.Title, l.NotFound.Message, loaderHidden)
	utils.Swal2Render(w, page)

	if waitEnabled {
		roomWait(w, r)
	} else {
		w.Write([]byte(`<meta http-equiv="refresh" content="60">`))
	}
}

func (p *ProxyManagerCtx) RoomNotRunning(w http.ResponseWriter, r *http.Request, waitEnabled bool) {
	page, l := p.lobbyPage()
	page.Body = lobbyBody(iconWarning, l.NotRunning.Title, l.NotRunning.Message, loaderHidden)
	utils.Swal2Render(w, page)

	if waitEnabled {
		roomWait(w, r)
	} else {
		w.Write([]byte(`<meta http-equiv="refresh" content="10">`))
	}
}

func (p *ProxyManagerCtx) RoomPaused(w http.ResponseWriter, r *http.Request, waitEnabled bool) {
	page, l := p.lobbyPage()
	page.Body = lobbyBody(iconWarning, l.Paused.Title, l.Paused.Message, loaderHidden+reloadButton)
	utils.Swal2Render(w, page)

	if waitEnabled {
		roomWait(w, r)
	} else {
		w.Write([]byte(`<meta http-equiv="refresh" content="2">`))
	}
}

func (p *ProxyManagerCtx) RoomNotReady(w http.ResponseWriter, r *http.Request, waitEnabled bool) {
	page, l := p.lobbyPage()
	page.Body = `<meta http-equiv="refresh" content="2">` + lobbyBody(iconInfo, l.NotReady.Title, l.NotReady.Message, loaderVisible+reloadButton)
	utils.Swal2Render(w, page)

	if waitEnabled {
		roomWait(w, r)
	} else {
		w.Write([]byte(`<meta http-equiv="refresh" content="2">`))
	}
}

func (p *ProxyManagerCtx) RoomReady(w http.ResponseWriter, r *http.Request) {
	page, l := p.lobbyPage()
	popup := html.EscapeString(string(page.PopupColor))
	icon := fmt.Sprintf(`
		<div class="swal2-icon swal2-success swal2-icon-show" style="display: flex;">
			<div class="swal2-success-circular-line-left" style="background-color: %[1]s;"></div>
			<span class="swal2-success-line-tip"></span> <span class="swal2-success-line-long"></span>
			<div class="swal2-success-ring"></div> <div class="swal2-success-fix" style="background-color: %[1]s;"></div>
			<div class="swal2-success-circular-line-right" style="background-color: %[1]s;"></div>
		</div>`, popup)
	page.Body = lobbyBody(icon, l.Ready.Title, l.Ready.Message,
		`<button type="button" onclick="location = location" class="swal2-confirm swal2-styled">Go to room</button>`) +
		`<div class="swal2-content swal2-actions">
			<small>If you see this page after refresh, <br /> it can mean misconfiguration on your side.</small>
		</div>`
	utils.Swal2Render(w, page)
}

func (p *ProxyManagerCtx) RoomLoginRequired(w http.ResponseWriter, r *http.Request, loginURL string) {
	page, l := p.lobbyPage()
	target := loginURL + "#/login?next=" + url.QueryEscape(r.URL.RequestURI())
	page.Body = lobbyBody(iconInfo, l.LoginRequired.Title, l.LoginRequired.Message,
		fmt.Sprintf(`<a href="%s" class="swal2-confirm swal2-styled" style="text-decoration:none">Sign in</a>`, html.EscapeString(target)))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	utils.Swal2Render(w, page)
}
