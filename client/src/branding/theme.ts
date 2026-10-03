// Applies instance branding to the running app: vuetify colors, CSS
// variables, fonts, custom CSS, title and favicon.
import { Framework } from 'vuetify'
import { Branding, Palette } from '@/api-ext'

const THEME_KEY = 'neko-rooms-theme'

export function getUserThemePref(): 'dark' | 'light' | null {
  try {
    const v = localStorage.getItem(THEME_KEY)
    return v === 'dark' || v === 'light' ? v : null
  } catch {
    return null
  }
}

export function setUserThemePref(v: 'dark' | 'light' | null) {
  try {
    if (v) localStorage.setItem(THEME_KEY, v)
    else localStorage.removeItem(THEME_KEY)
  } catch {
    // storage unavailable
  }
}

export function resolveDark(b: Branding): boolean {
  if (b.theme.allow_user_toggle) {
    const pref = getUserThemePref()
    if (pref) return pref === 'dark'
  }
  if (b.theme.mode === 'system') {
    return !window.matchMedia || !window.matchMedia('(prefers-color-scheme: light)').matches
  }
  return b.theme.mode !== 'light'
}

function styleTag(id: string): HTMLStyleElement {
  let el = document.getElementById(id) as HTMLStyleElement | null
  if (!el) {
    el = document.createElement('style')
    el.id = id
    document.head.appendChild(el)
  }
  return el
}

function setLink(id: string, rel: string, href: string) {
  let el = document.getElementById(id) as HTMLLinkElement | null
  if (!href) {
    if (el) el.remove()
    return
  }
  if (!el) {
    el = document.createElement('link')
    el.id = id
    el.rel = rel
    document.head.appendChild(el)
  }
  if (el.href !== href) el.href = href
}

function setMeta(name: string, content: string) {
  let el = document.querySelector(`meta[name="${name}"]`) as HTMLMetaElement | null
  if (!el) {
    el = document.createElement('meta')
    el.name = name
    document.head.appendChild(el)
  }
  el.content = content
}

function firstFamily(family: string): string {
  return (family.split(',')[0] || '').trim().replace(/^["']|["']$/g, '')
}

const cssEscape = (s: string) => s.replace(/<\/?style/gi, '')

export function expand(b: Branding, s: string): string {
  return (s || '').replace(/\{year\}/g, String(new Date().getFullYear())).replace(/\{app_name\}/g, b.app_name)
}

export function logoFor(b: Branding, dark: boolean): string {
  return (!dark && b.logo_light) ? b.logo_light : b.logo
}

function vuetifyColors(p: Palette) {
  return {
    primary: p.primary,
    secondary: p.secondary,
    accent: p.accent,
    error: p.error,
    info: p.info,
    success: p.success,
    warning: p.warning,
  }
}

export function applyBranding(vuetify: Framework, b: Branding, dark: boolean) {
  const theme = vuetify.theme
  Object.assign(theme.themes.dark, vuetifyColors(b.theme.dark))
  Object.assign(theme.themes.light, vuetifyColors(b.theme.light))
  theme.dark = dark

  const p = dark ? b.theme.dark : b.theme.light
  const t = b.theme
  const radius = `${t.border_radius}px`
  const heading = t.heading_font_family || t.font_family

  let css = `
:root {
  --nr-primary: ${p.primary};
  --nr-accent: ${p.accent};
  --nr-bg: ${p.background};
  --nr-surface: ${p.surface};
  --nr-secondary: ${p.secondary};
  --nr-text: ${p.text};
  --nr-appbar: ${p.app_bar};
  --nr-appbar-text: ${p.app_bar_text};
  --nr-footer: ${p.footer};
  --nr-footer-text: ${p.footer_text};
  --nr-drawer: ${p.drawer};
  --nr-radius: ${radius};
  --nr-font: ${t.font_family || 'Roboto, sans-serif'};
  --nr-heading-font: ${heading || 'Roboto, sans-serif'};
  --nr-font-size: ${t.font_size || 14}px;
}
html, body { background: var(--nr-bg); }
.v-application, .v-application .text-h1, .v-application .text-h2, .v-application .text-h3,
.v-application .text-h4, .v-application .text-h5, .v-application .text-h6,
.v-application .headline, .v-application .title, .v-application .body-1, .v-application .body-2,
.v-application .text-body-1, .v-application .text-body-2, .v-application .subtitle-1, .v-application .subtitle-2,
.v-application .caption, .v-application .text-caption, .v-application .text-subtitle-1, .v-application .text-subtitle-2 {
  font-family: var(--nr-font) !important;
}
.v-application h1, .v-application h2, .v-application h3, .v-application .nr-heading,
.v-application .text-h3, .v-application .text-h4, .v-application .text-h5, .v-application .text-h6 {
  font-family: var(--nr-heading-font) !important;
}
.v-application { font-size: var(--nr-font-size); }
.v-application.theme--dark, .v-application.theme--light { background: var(--nr-bg) !important; color: var(--nr-text) !important; }
.v-application .theme--dark.v-card, .v-application .theme--light.v-card,
.v-application .theme--dark.v-sheet:not(.v-app-bar):not(.v-footer):not(.v-navigation-drawer):not(.transparent),
.v-application .theme--light.v-sheet:not(.v-app-bar):not(.v-footer):not(.v-navigation-drawer):not(.transparent),
.v-application .theme--dark.v-list, .v-application .theme--light.v-list,
.v-application .theme--dark.v-data-table, .v-application .theme--light.v-data-table,
.v-application .theme--dark.v-expansion-panels .v-expansion-panel, .v-application .theme--light.v-expansion-panels .v-expansion-panel,
.v-application .theme--dark.v-tabs-items, .v-application .theme--light.v-tabs-items,
.v-application .theme--dark.v-tabs > .v-tabs-bar, .v-application .theme--light.v-tabs > .v-tabs-bar,
.theme--dark.v-menu__content .v-list, .theme--light.v-menu__content .v-list,
.v-dialog .theme--dark.v-card, .v-dialog .theme--light.v-card {
  background-color: var(--nr-surface) !important;
}
.v-application .nr-appbar.v-app-bar { background-color: var(--nr-appbar) !important; color: var(--nr-appbar-text) !important; }
.v-application .nr-appbar .v-btn, .v-application .nr-appbar .v-icon, .v-application .nr-appbar .nr-appbar-text { color: var(--nr-appbar-text) !important; }
.v-application .nr-footer.v-footer { background-color: var(--nr-footer) !important; color: var(--nr-footer-text) !important; }
.v-application .nr-footer a { color: var(--nr-footer-text) !important; }
.v-application .nr-drawer.v-navigation-drawer { background-color: var(--nr-drawer) !important; }
.v-application .nr-drawer.v-navigation-drawer .v-list { background-color: transparent !important; }
.v-application .v-card, .v-application .v-sheet.nr-rounded, .v-dialog, .v-menu__content,
.v-application .v-btn:not(.v-btn--round):not(.v-btn--fab):not(.v-btn--icon),
.v-application .v-text-field--outlined fieldset, .v-application .v-alert,
.v-application .v-text-field--solo .v-input__slot, .v-application .v-data-table {
  border-radius: var(--nr-radius) !important;
}
`
  if (!t.button_uppercase) {
    css += `.v-application .v-btn, .v-application .v-tab { text-transform: none !important; letter-spacing: normal !important; }\n`
  }
  if (t.font_file && t.font_family) {
    css += `@font-face { font-family: "${firstFamily(t.font_family).replace(/"/g, '')}"; src: url("${t.font_file.replace(/"/g, '')}"); font-display: swap; }\n`
  }

  styleTag('branding-theme').textContent = cssEscape(css)
  styleTag('branding-custom').textContent = cssEscape(b.custom_css || '')

  // the boot style is replaced by the theme above
  const boot = document.getElementById('branding-boot')
  if (boot) boot.textContent = ''

  setLink('branding-font', 'stylesheet', t.font_url)

  // favicon
  document.querySelectorAll('link[rel="icon"]:not(#branding-favicon)').forEach(el => {
    if (b.favicon) el.remove()
  })
  setLink('branding-favicon', 'icon', b.favicon)

  if (b.theme_color) setMeta('theme-color', b.theme_color)
}

export function setTitle(b: Branding, page?: string) {
  const base = expand(b, b.page_title || b.app_name)
  document.title = page ? `${page} · ${base}` : base
}
