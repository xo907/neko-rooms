package utils

import (
	_ "embed"
	"html/template"
	"net/http"
)

//go:embed swal2.html
var swal2Template string

var swal2Tmpl = template.Must(template.New("main").Parse(swal2Template))

// Swal2Page holds everything needed to render a lobby page. CSS values must be
// validated by the caller, they are inserted verbatim into the stylesheet.
type Swal2Page struct {
	Title   string
	AppName string
	Favicon string
	FontURL string
	Logo    string

	FontFamily      template.CSS
	BackgroundColor template.CSS
	BackgroundImage template.CSS
	PopupColor      template.CSS
	TextColor       template.CSS
	ButtonColor     template.CSS
	CustomCSS       template.CSS

	PoweredBy bool
	Body      template.HTML
}

func DefaultSwal2Page() Swal2Page {
	return Swal2Page{
		Title:           "Neko rooms",
		AppName:         "neko-rooms",
		FontFamily:      "Arial, sans-serif",
		BackgroundColor: "black",
		PopupColor:      "#2f3136",
		TextColor:       "#dcddde",
		ButtonColor:     "#202225",
		PoweredBy:       true,
	}
}

func Swal2Render(w http.ResponseWriter, page Swal2Page) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	if err := swal2Tmpl.Execute(w, page); err != nil {
		http.Error(w, err.Error(), 500)
	}
}

func Swal2Response(w http.ResponseWriter, body string) {
	page := DefaultSwal2Page()
	page.Body = template.HTML(body)
	Swal2Render(w, page)
}
