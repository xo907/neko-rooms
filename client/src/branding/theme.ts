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

// readable text color on top of a solid color
function onColor(hex: string): string {
  const m = /^#?([0-9a-f]{3}|[0-9a-f]{6})$/i.exec(hex.trim())
  if (!m) return "#ffffff"
  let h = m[1]
  if (h.length === 3) h = h.split("").map(c => c + c).join("")
  const [r, g, b] = [0, 2, 4].map(i => {
    const c = parseInt(h.slice(i, i + 2), 16) / 255
    return c <= 0.03928 ? c / 12.92 : Math.pow((c + 0.055) / 1.055, 2.4)
  })
  const l = 0.2126 * r + 0.7152 * g + 0.0722 * b
  return l > 0.4 ? `color-mix(in srgb, ${hex} 12%, #000)` : "#ffffff"
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
  --bg-canvas: ${p.background};
  --bg-surface-1: ${p.surface};
  --bg-surface-2: color-mix(in srgb, ${p.surface} 96.5%, ${p.text});
  --bg-surface-3: color-mix(in srgb, ${p.surface} 92.5%, ${p.text});
  --text-primary: ${p.text};
  --text-secondary: color-mix(in srgb, ${p.text} 66%, ${p.background});
  --text-tertiary: color-mix(in srgb, ${p.text} 55%, ${p.background});
  --text-disabled: color-mix(in srgb, ${p.text} 36%, ${p.background});
  --border-subtle: color-mix(in srgb, ${p.text} 7%, transparent);
  --border-hover: color-mix(in srgb, ${p.text} 15%, transparent);
  --border-focus: color-mix(in srgb, ${p.text} 35%, transparent);
  --focus-ring: 0 0 0 3px color-mix(in srgb, ${p.text} 6%, transparent);
  --accent-primary: ${p.primary};
  --accent-ink: ${onColor(p.primary)};
  --font-sans: ${t.font_family || "'Geist', sans-serif"};
  --radius-lg: ${radius};
  --radius-md: ${Math.round(t.border_radius * 0.8)}px;
  --radius-sm: ${Math.round(t.border_radius * 0.6)}px;

  /* older names, still used by some components */
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
  --nr-font: var(--font-sans);
  --nr-heading-font: ${heading ? heading : 'var(--font-sans)'};
  --nr-font-size: ${t.font_size || 14}px;
  --nr-on-primary: var(--accent-ink);
  --nr-on-accent: ${onColor(p.accent)};
  --nr-line: var(--border-subtle);
  --nr-line-strong: var(--border-hover);
  --nr-hover: var(--bg-surface-2);
  --nr-press: var(--bg-surface-3);
  --nr-muted: var(--text-secondary);
  --nr-subtle: var(--text-tertiary);
  --nr-glass: var(--bg-surface-1);
  --nr-glow: transparent;
}
html, body { background: var(--bg-canvas); }
.v-application { font-size: var(--nr-font-size); }
.v-application, .v-application .text-h1, .v-application .text-h2, .v-application .text-h3,
.v-application .text-h4, .v-application .text-h5, .v-application .text-h6,
.v-application .headline, .v-application .title, .v-application .body-1, .v-application .body-2,
.v-application .text-body-1, .v-application .text-body-2, .v-application .subtitle-1, .v-application .subtitle-2,
.v-application .caption, .v-application .text-caption, .v-application .text-subtitle-1, .v-application .text-subtitle-2 {
  font-family: var(--font-sans) !important;
}
.v-application h1, .v-application h2, .v-application h3, .v-application .nr-heading {
  font-family: var(--nr-heading-font) !important;
}
`
  // buttons and tabs are never uppercase in this look, see _deck.scss
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
