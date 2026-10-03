package branding

import (
	"fmt"
	"regexp"
	"strings"
)

type Link struct {
	Label  string `json:"label"`
	URL    string `json:"url"`
	Icon   string `json:"icon"` // mdi icon name, e.g. mdi-github
	NewTab bool   `json:"new_tab"`
}

type Palette struct {
	Primary    string `json:"primary"`
	Secondary  string `json:"secondary"`
	Accent     string `json:"accent"`
	Error      string `json:"error"`
	Info       string `json:"info"`
	Success    string `json:"success"`
	Warning    string `json:"warning"`
	Background string `json:"background"`
	Surface    string `json:"surface"`
	Text       string `json:"text"`
	AppBar     string `json:"app_bar"`
	AppBarText string `json:"app_bar_text"`
	Footer     string `json:"footer"`
	FooterText string `json:"footer_text"`
	Drawer     string `json:"drawer"`
}

type Theme struct {
	Mode            string  `json:"mode"` // dark | light | system
	AllowUserToggle bool    `json:"allow_user_toggle"`
	Dark            Palette `json:"dark"`
	Light           Palette `json:"light"`

	FontFamily        string `json:"font_family"`
	FontURL           string `json:"font_url"`  // stylesheet URL, e.g. Google Fonts
	FontFile          string `json:"font_file"` // uploaded font file, registered under the first font family name
	HeadingFontFamily string `json:"heading_font_family"`
	FontSize          int    `json:"font_size"`     // px
	BorderRadius      int    `json:"border_radius"` // px
	Dense             bool   `json:"dense"`
	ButtonUppercase   bool   `json:"button_uppercase"`
}

type Header struct {
	Show         bool   `json:"show"`
	ShowLogo     bool   `json:"show_logo"`
	ShowAppName  bool   `json:"show_app_name"`
	LogoHeight   int    `json:"logo_height"`
	Elevation    int    `json:"elevation"`
	Links        []Link `json:"links"`
	Announcement string `json:"announcement"` // banner shown under app bar
	AnnounceType string `json:"announcement_type"`
}

type Footer struct {
	Show           bool   `json:"show"`
	Text           string `json:"text"` // {year} and {app_name} placeholders
	Links          []Link `json:"links"`
	ShowPoweredBy  bool   `json:"show_powered_by"`
	PoweredByLabel string `json:"powered_by_label"`
}

type Home struct {
	HeroShow       string   `json:"hero_show"` // guests | always | never
	HeroTitle      string   `json:"hero_title"`
	HeroSubtitle   string   `json:"hero_subtitle"`
	HeroImage      string   `json:"hero_image"`
	HeroGradient   bool     `json:"hero_gradient"`
	CTALabel       string   `json:"cta_label"`
	FeaturedLabel  string   `json:"featured_label"`
	FriendsLabel   string   `json:"friends_label"`
	PublicLabel    string   `json:"public_label"`
	MineLabel      string   `json:"mine_label"`
	EmptyText      string   `json:"empty_text"`
	ShowSearch     bool     `json:"show_search"`
	ShowCategories bool     `json:"show_categories"`
	ShowFriends    bool     `json:"show_friends"` // friends sidebar
	ShowOffline    bool     `json:"show_offline"` // list rooms that are not running
	CardStyle      string   `json:"card_style"`   // tile | compact
	Categories     []string `json:"categories"`
}

type Login struct {
	Title           string `json:"title"`
	Message         string `json:"message"`
	ShowLogo        bool   `json:"show_logo"`
	BackgroundColor string `json:"background_color"`
	BackgroundImage string `json:"background_image"` // url or asset url
	CardOpacity     int    `json:"card_opacity"`     // 0-100
	Position        string `json:"position"`         // center | left | right
	FooterText      string `json:"footer_text"`
}

type LobbyMessage struct {
	Title   string `json:"title"`
	Message string `json:"message"`
}

type Lobby struct {
	PageTitle       string       `json:"page_title"`
	ShowLogo        bool         `json:"show_logo"`
	BackgroundColor string       `json:"background_color"`
	BackgroundImage string       `json:"background_image"`
	PopupColor      string       `json:"popup_color"`
	TextColor       string       `json:"text_color"`
	ButtonColor     string       `json:"button_color"`
	FontFamily      string       `json:"font_family"`
	NotFound        LobbyMessage `json:"not_found"`
	NotRunning      LobbyMessage `json:"not_running"`
	Paused          LobbyMessage `json:"paused"`
	NotReady        LobbyMessage `json:"not_ready"`
	Ready           LobbyMessage `json:"ready"`
	LoginRequired   LobbyMessage `json:"login_required"`
	CustomCSS       string       `json:"custom_css"`
}

type Rooms struct {
	Inject     bool   `json:"inject"` // inject branding into neko room pages
	PageTitle  string `json:"page_title"`
	Favicon    string `json:"favicon"` // url or asset url, empty = global favicon
	CustomCSS  string `json:"custom_css"`
	CustomJS   string `json:"custom_js"`
	CustomHead string `json:"custom_head"`
}

type Branding struct {
	AppName     string `json:"app_name"`
	PageTitle   string `json:"page_title"`
	Tagline     string `json:"tagline"`
	Description string `json:"description"` // meta description
	Logo        string `json:"logo"`        // url or asset url
	LogoLight   string `json:"logo_light"`  // used in light mode, falls back to logo
	Favicon     string `json:"favicon"`
	ThemeColor  string `json:"theme_color"` // <meta name="theme-color">

	Theme  Theme  `json:"theme"`
	Header Header `json:"header"`
	Footer Footer `json:"footer"`
	Home   Home   `json:"home"`
	Login  Login  `json:"login"`
	Lobby  Lobby  `json:"lobby"`
	Rooms  Rooms  `json:"rooms"`

	CustomCSS  string `json:"custom_css"`
	CustomHead string `json:"custom_head"` // raw HTML injected into <head>
	CustomJS   string `json:"custom_js"`
}

func Default() Branding {
	return Branding{
		AppName:     "neko-rooms",
		PageTitle:   "neko-rooms",
		Tagline:     "Self-hosted collaborative browser rooms",
		Description: "Room management for n.eko",
		ThemeColor:  "#8c5cff",

		Theme: Theme{
			Mode:            "dark",
			AllowUserToggle: true,
			Dark: Palette{
				Primary:    "#8c5cff",
				Secondary:  "#2a2540",
				Accent:     "#ff5ca8",
				Error:      "#ff5252",
				Info:       "#3ec5ff",
				Success:    "#2ee59d",
				Warning:    "#ffb547",
				Background: "#13111c",
				Surface:    "#1d1a2b",
				Text:       "#ece9f7",
				AppBar:     "#1d1a2b",
				AppBarText: "#ffffff",
				Footer:     "#13111c",
				FooterText: "#a29dbb",
				Drawer:     "#1a1726",
			},
			Light: Palette{
				Primary:    "#6c3cf0",
				Secondary:  "#e9e4fb",
				Accent:     "#e83e8c",
				Error:      "#e53935",
				Info:       "#0091ea",
				Success:    "#00b974",
				Warning:    "#f59e0b",
				Background: "#f5f3fc",
				Surface:    "#ffffff",
				Text:       "#1e1b2e",
				AppBar:     "#ffffff",
				AppBarText: "#1e1b2e",
				Footer:     "#efecf9",
				FooterText: "#5d5875",
				Drawer:     "#ffffff",
			},
			FontFamily:        "Nunito, Roboto, Helvetica Neue, Arial, sans-serif",
			FontURL:           "https://fonts.googleapis.com/css2?family=Nunito:wght@400;600;700;800&display=swap",
			HeadingFontFamily: "",
			FontSize:          14,
			BorderRadius:      12,
			ButtonUppercase:   false,
		},

		Header: Header{
			Show:         true,
			ShowLogo:     true,
			ShowAppName:  true,
			LogoHeight:   36,
			Elevation:    4,
			Links:        []Link{},
			AnnounceType: "info",
		},

		Footer: Footer{
			Show:           true,
			Text:           "© {year} {app_name}",
			Links:          []Link{},
			ShowPoweredBy:  true,
			PoweredByLabel: "based on n.eko",
		},

		Home: Home{
			HeroShow:       "guests",
			HeroTitle:      "Watch anything, together.",
			HeroSubtitle:   "Hop into a room, share a browser and hang out with friends in real time.",
			HeroGradient:   true,
			CTALabel:       "Create a room",
			FeaturedLabel:  "Featured",
			FriendsLabel:   "Friends' rooms",
			PublicLabel:    "Public rooms",
			MineLabel:      "Your rooms",
			EmptyText:      "No rooms are open right now. Why not start one?",
			ShowSearch:     true,
			ShowCategories: true,
			ShowFriends:    true,
			ShowOffline:    false,
			CardStyle:      "tile",
			Categories:     []string{"Movies", "Music", "Gaming", "Sports", "Study", "Chill"},
		},

		Login: Login{
			Title:       "Sign in",
			Message:     "",
			ShowLogo:    true,
			CardOpacity: 100,
			Position:    "center",
		},

		Lobby: Lobby{
			PageTitle:       "{app_name}",
			ShowLogo:        true,
			BackgroundColor: "#13111c",
			PopupColor:      "#1d1a2b",
			TextColor:       "#ece9f7",
			ButtonColor:     "#8c5cff",
			FontFamily:      "Arial, sans-serif",
			NotFound: LobbyMessage{
				Title:   "Room not found!",
				Message: "The room you are trying to join does not exist.\nYou can wait on this page until it will be created.",
			},
			NotRunning: LobbyMessage{
				Title:   "Room is not running!",
				Message: "The room you are trying to join is not running.\nYou can wait on this page until it will be started.",
			},
			Paused: LobbyMessage{
				Title:   "Room is paused!",
				Message: "The room you are trying to join is paused.\nYou can wait on this page until it will be unpaused.",
			},
			NotReady: LobbyMessage{
				Title:   "Room is not ready, yet!",
				Message: "Please wait, until this room is ready so you can join. This should happen any second now.",
			},
			Ready: LobbyMessage{
				Title:   "Room is ready!",
				Message: "Requested room is ready, you can join now.\nTry to reload page.",
			},
			LoginRequired: LobbyMessage{
				Title:   "Sign in required",
				Message: "You need to sign in before you can join this room.",
			},
		},

		Rooms: Rooms{
			Inject: false,
		},
	}
}

var (
	colorRegex = regexp.MustCompile(`^(#[0-9a-fA-F]{3,8}|(rgb|rgba|hsl|hsla)\([0-9.,%\s]+\)|[a-zA-Z]+)?$`)
	iconRegex  = regexp.MustCompile(`^(mdi-[a-z0-9-]+)?$`)
)

func checkColor(name, value string) error {
	if !colorRegex.MatchString(value) {
		return fmt.Errorf("invalid color for %s: %q", name, value)
	}
	return nil
}

func checkURL(name, value string) error {
	if value == "" {
		return nil
	}
	lower := strings.ToLower(strings.TrimSpace(value))
	if strings.HasPrefix(lower, "javascript:") || strings.HasPrefix(lower, "vbscript:") {
		return fmt.Errorf("invalid url for %s", name)
	}
	if strings.ContainsAny(value, "\"'<>\n\r") {
		return fmt.Errorf("invalid characters in url for %s", name)
	}
	return nil
}

func checkCSSValue(name, value string) error {
	if strings.ContainsAny(value, "{};<>\n\r") {
		return fmt.Errorf("invalid characters in %s", name)
	}
	return nil
}

func (p Palette) validate(prefix string) error {
	for name, v := range map[string]string{
		"primary": p.Primary, "secondary": p.Secondary, "accent": p.Accent,
		"error": p.Error, "info": p.Info, "success": p.Success, "warning": p.Warning,
		"background": p.Background, "surface": p.Surface, "text": p.Text,
		"app_bar": p.AppBar, "app_bar_text": p.AppBarText,
		"footer": p.Footer, "footer_text": p.FooterText, "drawer": p.Drawer,
	} {
		if err := checkColor(prefix+"."+name, v); err != nil {
			return err
		}
	}
	return nil
}

// Validate checks values that end up inside generated CSS or HTML attributes.
// Free-form custom CSS/JS/HTML is intentionally allowed, it is admin-only.
func (b *Branding) Validate() error {
	switch b.Theme.Mode {
	case "dark", "light", "system":
	default:
		return fmt.Errorf("invalid theme mode: %q", b.Theme.Mode)
	}

	if err := b.Theme.Dark.validate("theme.dark"); err != nil {
		return err
	}
	if err := b.Theme.Light.validate("theme.light"); err != nil {
		return err
	}

	for name, v := range map[string]string{
		"theme_color":            b.ThemeColor,
		"login.background_color": b.Login.BackgroundColor,
		"lobby.background_color": b.Lobby.BackgroundColor,
		"lobby.popup_color":      b.Lobby.PopupColor,
		"lobby.text_color":       b.Lobby.TextColor,
		"lobby.button_color":     b.Lobby.ButtonColor,
	} {
		if err := checkColor(name, v); err != nil {
			return err
		}
	}

	for name, v := range map[string]string{
		"logo": b.Logo, "logo_light": b.LogoLight, "favicon": b.Favicon,
		"theme.font_url":         b.Theme.FontURL,
		"theme.font_file":        b.Theme.FontFile,
		"home.hero_image":        b.Home.HeroImage,
		"login.background_image": b.Login.BackgroundImage,
		"lobby.background_image": b.Lobby.BackgroundImage,
		"rooms.favicon":          b.Rooms.Favicon,
	} {
		if err := checkURL(name, v); err != nil {
			return err
		}
	}

	for name, v := range map[string]string{
		"theme.font_family":         b.Theme.FontFamily,
		"theme.heading_font_family": b.Theme.HeadingFontFamily,
		"lobby.font_family":         b.Lobby.FontFamily,
	} {
		if err := checkCSSValue(name, v); err != nil {
			return err
		}
	}

	for _, links := range [][]Link{b.Header.Links, b.Footer.Links} {
		for _, l := range links {
			if err := checkURL("link", l.URL); err != nil {
				return err
			}
			if !iconRegex.MatchString(l.Icon) {
				return fmt.Errorf("invalid icon name: %q", l.Icon)
			}
		}
	}

	if b.Theme.FontSize != 0 && (b.Theme.FontSize < 8 || b.Theme.FontSize > 32) {
		return fmt.Errorf("font size must be between 8 and 32")
	}
	if b.Theme.BorderRadius < 0 || b.Theme.BorderRadius > 64 {
		return fmt.Errorf("border radius must be between 0 and 64")
	}
	if b.Header.LogoHeight < 0 || b.Header.LogoHeight > 128 {
		return fmt.Errorf("logo height must be between 0 and 128")
	}
	if b.Login.CardOpacity < 0 || b.Login.CardOpacity > 100 {
		return fmt.Errorf("card opacity must be between 0 and 100")
	}
	switch b.Home.HeroShow {
	case "", "guests", "always", "never":
	default:
		return fmt.Errorf("invalid hero visibility: %q", b.Home.HeroShow)
	}
	switch b.Home.CardStyle {
	case "", "tile", "compact":
	default:
		return fmt.Errorf("invalid card style: %q", b.Home.CardStyle)
	}
	if b.Home.Categories == nil {
		b.Home.Categories = []string{}
	}

	switch b.Login.Position {
	case "", "center", "left", "right":
	default:
		return fmt.Errorf("invalid login position: %q", b.Login.Position)
	}

	if b.Header.Links == nil {
		b.Header.Links = []Link{}
	}
	if b.Footer.Links == nil {
		b.Footer.Links = []Link{}
	}

	return nil
}

// Expand replaces {year} and {app_name} placeholders.
func (b *Branding) Expand(s string, year int) string {
	return strings.NewReplacer(
		"{year}", fmt.Sprint(year),
		"{app_name}", b.AppName,
	).Replace(s)
}
