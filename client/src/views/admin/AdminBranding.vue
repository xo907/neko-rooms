<template>
  <div v-if="draft">
    <v-alert type="info" text dense class="mb-4">
      Changes preview live across the whole site while you edit. Visitors only see them after you save.
    </v-alert>

    <v-card>
      <v-tabs v-model="tab" show-arrows>
        <v-tab><v-icon left small>mdi-card-account-details-outline</v-icon>Identity</v-tab>
        <v-tab><v-icon left small>mdi-palette-outline</v-icon>Colors</v-tab>
        <v-tab><v-icon left small>mdi-format-font</v-icon>Typography & shape</v-tab>
        <v-tab><v-icon left small>mdi-page-layout-header</v-icon>Header & footer</v-tab>
        <v-tab><v-icon left small>mdi-home-outline</v-icon>Homepage</v-tab>
        <v-tab><v-icon left small>mdi-login</v-icon>Sign-in page</v-tab>
        <v-tab><v-icon left small>mdi-door</v-icon>Room status pages</v-tab>
        <v-tab><v-icon left small>mdi-monitor-share</v-icon>Inside rooms</v-tab>
        <v-tab><v-icon left small>mdi-code-tags</v-icon>Advanced</v-tab>
      </v-tabs>
      <v-divider />

      <v-tabs-items v-model="tab">
        <!-- identity -->
        <v-tab-item>
          <v-card-text>
            <v-row>
              <v-col cols="12" md="6">
                <v-text-field v-model="draft.app_name" label="Site name" outlined dense class="mb-2" />
                <v-text-field v-model="draft.page_title" label="Browser tab title" hint="Placeholders: {app_name}, {year}" persistent-hint outlined dense class="mb-4" />
                <v-text-field v-model="draft.tagline" label="Tagline" outlined dense class="mb-2" />
                <v-textarea v-model="draft.description" label="Description (search engines & link previews)" rows="2" auto-grow outlined dense />
                <ColorField v-model="draft.theme_color" label="Browser theme color (mobile address bar)" />
              </v-col>
              <v-col cols="12" md="6">
                <AssetField v-model="draft.logo" asset="logo" label="Logo" hint="PNG, SVG, WebP… up to 5 MB" class="mb-4" />
                <AssetField v-model="draft.logo_light" asset="logo-light" label="Logo for light mode (optional)" class="mb-4" />
                <AssetField v-model="draft.favicon" asset="favicon" accept="image/*,.ico" label="Favicon" hint="ICO, PNG or SVG" />
              </v-col>
            </v-row>
          </v-card-text>
        </v-tab-item>

        <!-- colors -->
        <v-tab-item>
          <v-card-text>
            <div class="subtitle-2 mb-2">Presets</div>
            <div class="d-flex flex-wrap mb-6" style="gap: 10px">
              <v-card v-for="p in presets" :key="p.name" outlined class="nr-preset pa-2" @click="applyPreset(p)">
                <div class="d-flex mb-2">
                  <div v-for="c in presetColors(p)" :key="c" class="nr-preset-dot" :style="{ background: c }" />
                </div>
                <div class="caption font-weight-bold">{{ p.name }}</div>
              </v-card>
            </div>

            <v-row>
              <v-col cols="12" md="4">
                <v-select v-model="draft.theme.mode" :items="modeItems" label="Default mode" outlined dense />
              </v-col>
              <v-col cols="12" md="8">
                <v-switch v-model="draft.theme.allow_user_toggle" inset dense label="Let visitors switch between light and dark" class="mt-1" />
              </v-col>
            </v-row>

            <v-tabs v-model="paletteTab" class="mb-4">
              <v-tab>Dark palette</v-tab>
              <v-tab>Light palette</v-tab>
            </v-tabs>
            <v-row dense>
              <v-col v-for="f in paletteFields" :key="f.key" cols="12" sm="6" md="4" lg="3">
                <ColorField v-model="palette[f.key]" :label="f.label" :hint="f.hint" class="mb-2" />
              </v-col>
            </v-row>
            <div class="mt-2">
              <v-btn small text @click="copyPalette">
                <v-icon left small>mdi-content-copy</v-icon>Copy {{ paletteTab === 0 ? 'light → dark' : 'dark → light' }}
              </v-btn>
            </div>
          </v-card-text>
        </v-tab-item>

        <!-- typography -->
        <v-tab-item>
          <v-card-text>
            <v-row>
              <v-col cols="12" md="6">
                <v-combobox v-model="draft.theme.font_family" :items="fontFamilies" label="Font family" hint="CSS font-family list" persistent-hint outlined dense class="mb-4" />
                <v-text-field v-model="draft.theme.font_url" label="Font stylesheet URL" hint="e.g. a Google Fonts CSS link" persistent-hint outlined dense class="mb-4" />
                <AssetField v-model="draft.theme.font_file" asset="font" accept=".woff,.woff2,.ttf,.otf,font/*" label="Or upload a font file" hint="Registered under the first name of the font family" class="mb-4" />
                <v-text-field v-model="draft.theme.heading_font_family" label="Heading font family (optional)" outlined dense />
              </v-col>
              <v-col cols="12" md="6">
                <div class="subtitle-2">Base font size: {{ draft.theme.font_size }}px</div>
                <v-slider v-model="draft.theme.font_size" min="11" max="20" step="1" thumb-label />
                <div class="subtitle-2">Corner radius: {{ draft.theme.border_radius }}px</div>
                <v-slider v-model="draft.theme.border_radius" min="0" max="28" step="1" thumb-label />
                <v-switch v-model="draft.theme.button_uppercase" inset dense label="Uppercase buttons" />
                <v-switch v-model="draft.theme.dense" inset dense label="Compact header" />
              </v-col>
            </v-row>
          </v-card-text>
        </v-tab-item>

        <!-- header & footer -->
        <v-tab-item>
          <v-card-text>
            <v-row>
              <v-col cols="12" md="6">
                <div class="subtitle-1 font-weight-bold mb-2">Header</div>
                <v-switch v-model="draft.header.show" inset dense label="Show header" />
                <v-switch v-model="draft.header.show_logo" inset dense label="Show logo" />
                <v-switch v-model="draft.header.show_app_name" inset dense label="Show site name" />
                <div class="subtitle-2 mt-2">Logo height: {{ draft.header.logo_height }}px</div>
                <v-slider v-model="draft.header.logo_height" min="16" max="64" thumb-label />
                <div class="subtitle-2">Shadow</div>
                <v-slider v-model="draft.header.elevation" min="0" max="12" thumb-label />
              </v-col>
              <v-col cols="12" md="6">
                <div class="subtitle-1 font-weight-bold mb-2">Announcement banner</div>
                <v-textarea v-model="draft.header.announcement" label="Message (empty = hidden)" rows="2" auto-grow outlined dense />
                <v-select v-model="draft.header.announcement_type" :items="['info', 'success', 'warning', 'error']" label="Style" outlined dense />
              </v-col>
              <v-col cols="12">
                <div class="subtitle-2 mb-2">Header links</div>
                <LinksEditor v-model="draft.header.links" />
              </v-col>
            </v-row>
            <v-divider class="my-6" />
            <v-row>
              <v-col cols="12" md="6">
                <div class="subtitle-1 font-weight-bold mb-2">Footer</div>
                <v-switch v-model="draft.footer.show" inset dense label="Show footer" />
                <v-text-field v-model="draft.footer.text" label="Footer text" hint="Placeholders: {app_name}, {year}" persistent-hint outlined dense class="mb-4" />
                <v-switch v-model="draft.footer.show_powered_by" inset dense label="Show “powered by” credit" />
                <v-text-field v-model="draft.footer.powered_by_label" :disabled="!draft.footer.show_powered_by" label="Credit label" outlined dense />
              </v-col>
              <v-col cols="12" md="6">
                <div class="subtitle-2 mb-2">Footer links</div>
                <LinksEditor v-model="draft.footer.links" />
              </v-col>
            </v-row>
          </v-card-text>
        </v-tab-item>

        <!-- homepage -->
        <v-tab-item>
          <v-card-text>
            <v-row>
              <v-col cols="12" md="6">
                <div class="subtitle-1 font-weight-bold mb-2">Hero banner</div>
                <v-select v-model="draft.home.hero_show" :items="heroItems" label="Show hero" outlined dense />
                <v-text-field v-model="draft.home.hero_title" label="Headline" outlined dense />
                <v-textarea v-model="draft.home.hero_subtitle" label="Subtitle" rows="2" auto-grow outlined dense />
                <v-text-field v-model="draft.home.cta_label" label="Button label" outlined dense />
                <AssetField v-model="draft.home.hero_image" asset="hero" label="Background image" class="mb-2" />
                <v-switch v-model="draft.home.hero_gradient" inset dense label="Color gradient overlay (primary → accent)" />
              </v-col>
              <v-col cols="12" md="6">
                <div class="subtitle-1 font-weight-bold mb-2">Room directory</div>
                <v-row dense>
                  <v-col cols="6"><v-text-field v-model="draft.home.featured_label" label="Featured section" outlined dense /></v-col>
                  <v-col cols="6"><v-text-field v-model="draft.home.friends_label" label="Friends section" outlined dense /></v-col>
                  <v-col cols="6"><v-text-field v-model="draft.home.public_label" label="Public section" outlined dense /></v-col>
                  <v-col cols="6"><v-text-field v-model="draft.home.mine_label" label="Own rooms section" outlined dense /></v-col>
                </v-row>
                <v-text-field v-model="draft.home.empty_text" label="Text when no rooms are open" outlined dense />
                <v-select v-model="draft.home.card_style" :items="[{ text: 'Large tiles', value: 'tile' }, { text: 'Compact tiles', value: 'compact' }]" label="Card style" outlined dense />
                <v-switch v-model="draft.home.show_search" inset dense label="Search box" />
                <v-switch v-model="draft.home.show_categories" inset dense label="Category filter" />
                <v-switch v-model="draft.home.show_friends" inset dense label="Friends sidebar" />
                <v-switch v-model="draft.home.show_offline" inset dense label="List rooms that are not running" />
                <v-combobox v-model="draft.home.categories" label="Categories" multiple chips small-chips deletable-chips outlined dense hint="Suggested when creating rooms, press enter to add" persistent-hint />
              </v-col>
            </v-row>
          </v-card-text>
        </v-tab-item>

        <!-- login -->
        <v-tab-item>
          <v-card-text>
            <v-row>
              <v-col cols="12" md="6">
                <v-text-field v-model="draft.login.title" label="Title" outlined dense />
                <v-textarea v-model="draft.login.message" label="Message" rows="2" auto-grow outlined dense />
                <v-textarea v-model="draft.login.footer_text" label="Small print below the form" rows="2" auto-grow outlined dense />
                <v-switch v-model="draft.login.show_logo" inset dense label="Show logo" />
              </v-col>
              <v-col cols="12" md="6">
                <ColorField v-model="draft.login.background_color" label="Background color (empty = theme)" class="mb-4" />
                <AssetField v-model="draft.login.background_image" asset="login-background" label="Background image" class="mb-4" />
                <v-select v-model="draft.login.position" :items="['center', 'left', 'right']" label="Form position" outlined dense />
                <div class="subtitle-2">Card opacity: {{ draft.login.card_opacity }}%</div>
                <v-slider v-model="draft.login.card_opacity" min="0" max="100" thumb-label />
                <v-btn small outlined :to="{ name: 'login' }" target="_blank" disabled>Preview after saving: sign out or open a private window</v-btn>
              </v-col>
            </v-row>
          </v-card-text>
        </v-tab-item>

        <!-- lobby -->
        <v-tab-item>
          <v-card-text>
            <p class="text--secondary">Shown when someone opens a room link while the room is missing, stopped, paused or starting.</p>
            <v-row>
              <v-col cols="12" md="5">
                <v-text-field v-model="draft.lobby.page_title" label="Tab title" outlined dense />
                <v-switch v-model="draft.lobby.show_logo" inset dense label="Show logo" />
                <v-row dense>
                  <v-col cols="6"><ColorField v-model="draft.lobby.background_color" label="Background" /></v-col>
                  <v-col cols="6"><ColorField v-model="draft.lobby.popup_color" label="Card" /></v-col>
                  <v-col cols="6"><ColorField v-model="draft.lobby.text_color" label="Text" /></v-col>
                  <v-col cols="6"><ColorField v-model="draft.lobby.button_color" label="Buttons & spinner" /></v-col>
                </v-row>
                <AssetField v-model="draft.lobby.background_image" asset="lobby-background" label="Background image" class="my-4" />
                <v-text-field v-model="draft.lobby.font_family" label="Font family" outlined dense />
                <v-textarea v-model="draft.lobby.custom_css" label="Custom CSS" rows="3" auto-grow outlined dense class="nr-code" />

                <div class="nr-lobby-preview mt-2" :style="lobbyPreviewStyle">
                  <div class="nr-lobby-card" :style="{ background: draft.lobby.popup_color, color: draft.lobby.text_color }">
                    <img v-if="draft.lobby.show_logo && draft.logo" :src="draft.logo" alt="" style="max-height: 32px">
                    <div class="font-weight-bold mt-2">{{ draft.lobby.not_running.title }}</div>
                    <div class="caption">{{ draft.lobby.not_running.message }}</div>
                    <div class="nr-lobby-btn mt-3" :style="{ background: draft.lobby.button_color }">Reload</div>
                  </div>
                </div>
              </v-col>
              <v-col cols="12" md="7">
                <div v-for="m in lobbyMessages" :key="m.key" class="mb-3">
                  <div class="subtitle-2 mb-1">{{ m.label }}</div>
                  <v-text-field v-model="draft.lobby[m.key].title" label="Title" outlined dense hide-details class="mb-2" />
                  <v-textarea v-model="draft.lobby[m.key].message" label="Message" rows="1" auto-grow outlined dense hide-details />
                </div>
              </v-col>
            </v-row>
          </v-card-text>
        </v-tab-item>

        <!-- rooms -->
        <v-tab-item>
          <v-card-text>
            <v-alert type="info" text dense>
              Applies to the neko player page of every room. Works with rooms proxied through neko-rooms (the default), not with rooms routed by Traefik directly. Reload an open room to see changes.
            </v-alert>
            <v-switch v-model="draft.rooms.inject" inset label="Brand the pages inside rooms" />
            <v-row :class="{ 'nr-disabled': !draft.rooms.inject }">
              <v-col cols="12" md="6">
                <v-switch v-model="draft.rooms.replace_logo" inset dense label="Replace the n.eko logo and name with your logo and site name" hint="Uses the logo from Identity, or the favicon if there is no logo" persistent-hint class="mt-0 mb-4" />
                <v-switch v-model="draft.rooms.site_colors" inset dense label="Use the site's dark palette inside rooms" hint="Backgrounds, side panel, join dialog, buttons, sliders" persistent-hint class="mb-6" />
                <v-text-field v-model="draft.rooms.page_title" label="Room tab title (empty = neko default)" hint="Placeholders: {room}, {app_name}, {year}" persistent-hint outlined dense class="mb-4" />
                <AssetField v-model="draft.rooms.favicon" asset="room-favicon" accept="image/*,.ico" label="Room favicon (empty = site favicon)" />
              </v-col>
              <v-col cols="12" md="6">
                <v-textarea v-model="draft.rooms.custom_css" label="Custom CSS for the neko client" rows="6" auto-grow outlined dense class="nr-code" />
              </v-col>
              <v-col cols="12" md="6">
                <v-textarea v-model="draft.rooms.custom_head" label="Extra HTML in <head>" rows="3" auto-grow outlined dense class="nr-code" />
              </v-col>
              <v-col cols="12" md="6">
                <v-textarea v-model="draft.rooms.custom_js" label="Custom JavaScript" rows="3" auto-grow outlined dense class="nr-code" />
              </v-col>
            </v-row>
          </v-card-text>
        </v-tab-item>

        <!-- advanced -->
        <v-tab-item>
          <v-card-text>
            <v-alert type="warning" text dense>Custom code runs for every visitor. Only add code you trust.</v-alert>
            <v-textarea v-model="draft.custom_css" label="Custom CSS" rows="8" auto-grow outlined class="nr-code" hint="Theme variables: --nr-primary, --nr-accent, --nr-bg, --nr-surface, --nr-text, --nr-radius, --nr-font …" persistent-hint />
            <v-textarea v-model="draft.custom_head" label="Extra HTML in <head> (analytics, meta tags…)" rows="4" auto-grow outlined class="nr-code mt-4" />
            <v-textarea v-model="draft.custom_js" label="Custom JavaScript" rows="4" auto-grow outlined class="nr-code" />

            <v-divider class="my-6" />
            <div class="subtitle-1 font-weight-bold mb-2">Backup & restore</div>
            <div class="d-flex flex-wrap" style="gap: 8px">
              <v-btn outlined @click="exportJson"><v-icon left>mdi-download</v-icon>Export theme</v-btn>
              <v-btn outlined @click="($refs.import).click()"><v-icon left>mdi-upload</v-icon>Import theme</v-btn>
              <input ref="import" type="file" accept="application/json,.json" class="d-none" @change="importJson">
              <v-spacer />
              <v-btn outlined color="error" @click="reset"><v-icon left>mdi-restore</v-icon>Reset everything to defaults</v-btn>
            </div>

            <div class="subtitle-1 font-weight-bold mt-8 mb-2">Uploaded files</div>
            <v-simple-table dense>
              <tbody>
                <tr v-for="a in assets" :key="a.name">
                  <td><code>{{ a.name }}</code></td>
                  <td>{{ a.mime }}</td>
                  <td>{{ (a.size / 1024).toFixed(1) }} KB</td>
                  <td class="text-right">
                    <v-btn icon small @click="copy(a.url)" title="Copy URL"><v-icon small>mdi-link-variant</v-icon></v-btn>
                    <v-btn icon small @click="deleteAsset(a.name)" title="Delete"><v-icon small>mdi-delete-outline</v-icon></v-btn>
                  </td>
                </tr>
                <tr v-if="!assets.length"><td class="text--secondary">No uploads yet.</td></tr>
              </tbody>
            </v-simple-table>
          </v-card-text>
        </v-tab-item>
      </v-tabs-items>
    </v-card>

    <SaveBar v-if="dirty">
      <span class="mr-2">Unsaved changes</span>
      <v-btn text @click="discard">Discard</v-btn>
      <v-btn color="primary" depressed :loading="saving" @click="save">Save & publish</v-btn>
    </SaveBar>
  </div>
</template>

<style lang="scss" scoped>
.nr-preset {
  width: 132px;
  cursor: pointer;
}
.nr-preset-dot {
  width: 20px;
  height: 20px;
  border-radius: 50%;
  margin-right: 4px;
  border: 1px solid rgba(128, 128, 128, .3);
}
.nr-code ::v-deep textarea {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace !important;
  font-size: 12px;
}
.nr-disabled {
  opacity: .5;
  pointer-events: none;
}
.nr-lobby-preview {
  border-radius: 8px;
  padding: 24px;
  display: flex;
  justify-content: center;
  background-size: cover !important;
  background-position: center !important;
}
.nr-lobby-card {
  border-radius: 6px;
  padding: 16px 20px;
  text-align: center;
  max-width: 280px;
}
.nr-lobby-btn {
  display: inline-block;
  color: white;
  padding: 4px 14px;
  border-radius: 4px;
  font-size: 13px;
}
</style>

<script lang="ts">
import { Vue, Component, Watch } from 'vue-property-decorator'
import { api, Asset, Branding, errorMessage, Palette } from '@/api-ext'
import { presets, ThemePreset } from '@/branding/presets'
import { copyText } from '@/utils/join'
import { resolveDark } from '@/branding/theme'
import ColorField from '@/components/admin/ColorField.vue'
import AssetField from '@/components/admin/AssetField.vue'
import LinksEditor from '@/components/admin/LinksEditor.vue'
import SaveBar from '@/components/admin/SaveBar.vue'

function clone<T>(v: T): T {
  return JSON.parse(JSON.stringify(v))
}

@Component({
  components: {
    ColorField,
    AssetField,
    LinksEditor,
    SaveBar,
  }
})
export default class AdminBranding extends Vue {
  tab = 0
  paletteTab = 0
  draft: Branding | null = null
  saved: Branding | null = null
  saving = false
  assets: Asset[] = []
  presets = presets

  modeItems = [
    { text: 'Dark', value: 'dark' },
    { text: 'Light', value: 'light' },
    { text: 'Follow visitor system setting', value: 'system' },
  ]

  heroItems = [
    { text: 'Only for signed-out visitors', value: 'guests' },
    { text: 'Always', value: 'always' },
    { text: 'Never', value: 'never' },
  ]

  fontFamilies = [
    'Nunito, Roboto, Helvetica Neue, Arial, sans-serif',
    'Inter, Roboto, Helvetica Neue, Arial, sans-serif',
    'Poppins, Roboto, Helvetica Neue, Arial, sans-serif',
    'Roboto, Helvetica Neue, Helvetica, Arial, sans-serif',
    'Rubik, Roboto, Helvetica Neue, Arial, sans-serif',
    'Manrope, Roboto, Helvetica Neue, Arial, sans-serif',
    'system-ui, -apple-system, Segoe UI, Roboto, sans-serif',
  ]

  paletteFields: { key: keyof Palette; label: string; hint?: string }[] = [
    { key: 'primary', label: 'Primary', hint: 'Buttons, links, highlights' },
    { key: 'accent', label: 'Accent', hint: 'Badges, gradients' },
    { key: 'secondary', label: 'Secondary' },
    { key: 'background', label: 'Page background' },
    { key: 'surface', label: 'Cards & dialogs' },
    { key: 'text', label: 'Text' },
    { key: 'app_bar', label: 'Header background' },
    { key: 'app_bar_text', label: 'Header text' },
    { key: 'footer', label: 'Footer background' },
    { key: 'footer_text', label: 'Footer text' },
    { key: 'drawer', label: 'Side menu' },
    { key: 'success', label: 'Success' },
    { key: 'info', label: 'Info' },
    { key: 'warning', label: 'Warning' },
    { key: 'error', label: 'Error' },
  ]

  lobbyMessages = [
    { key: 'not_found', label: 'Room does not exist' },
    { key: 'not_running', label: 'Room is stopped' },
    { key: 'paused', label: 'Room is paused' },
    { key: 'not_ready', label: 'Room is starting' },
    { key: 'ready', label: 'Room is ready (shown on misconfiguration)' },
    { key: 'login_required', label: 'Sign-in required' },
  ]

  get palette(): Palette {
    const d = this.draft as Branding
    return this.paletteTab === 0 ? d.theme.dark : d.theme.light
  }

  get dirty(): boolean {
    return !!this.draft && JSON.stringify(this.draft) !== JSON.stringify(this.saved)
  }

  get lobbyPreviewStyle() {
    const l = (this.draft as Branding).lobby
    return {
      background: l.background_image ? `url("${l.background_image}")` : l.background_color,
      fontFamily: l.font_family,
    }
  }

  presetColors(p: ThemePreset) {
    const pal = p.mode === 'light' ? p.light : p.dark
    return [pal.background, pal.surface, pal.primary, pal.accent]
  }

  // live preview: push the draft into the app store
  @Watch('draft', { deep: true })
  onDraft() {
    if (this.draft) this.$store.commit('BRANDING_SET', clone(this.draft))
  }

  @Watch('paletteTab')
  @Watch('draft.theme.mode')
  onPaletteTab() {
    // preview the palette that is being edited
    if (!this.draft) return
    this.$store.commit('DARK_SET', this.paletteTab === 0)
  }

  applyPreset(p: ThemePreset) {
    if (!this.draft) return
    this.draft.theme.mode = p.mode
    this.draft.theme.dark = clone(p.dark)
    this.draft.theme.light = clone(p.light)
    this.draft.theme.font_family = p.font_family
    this.draft.theme.font_url = p.font_url
    this.draft.theme.border_radius = p.border_radius
    this.draft.theme.button_uppercase = p.button_uppercase
    const pal = p.mode === 'light' ? p.light : p.dark
    this.draft.theme_color = pal.primary
    // keep room status pages in line with the theme
    this.draft.lobby.background_color = pal.background
    this.draft.lobby.popup_color = pal.surface
    this.draft.lobby.text_color = pal.text
    this.draft.lobby.button_color = pal.primary
    this.draft.lobby.font_family = p.font_family
    this.paletteTab = p.mode === 'light' ? 1 : 0
  }

  copyPalette() {
    if (!this.draft) return
    if (this.paletteTab === 0) this.draft.theme.dark = clone(this.draft.theme.light)
    else this.draft.theme.light = clone(this.draft.theme.dark)
  }

  async loadAssets() {
    this.assets = await api.assets()
  }

  async load() {
    const b = await api.branding()
    this.saved = clone(b)
    this.draft = clone(b)
    this.paletteTab = b.theme.mode === 'light' ? 1 : 0
    this.loadAssets()
  }

  async save() {
    if (!this.draft) return
    this.saving = true
    try {
      const b = await api.updateBranding({
        ...this.draft,
        theme: { ...this.draft.theme, font_size: Number(this.draft.theme.font_size), border_radius: Number(this.draft.theme.border_radius) },
      })
      this.saved = clone(b)
      this.draft = clone(b)
      this.loadAssets()
      this.$swal({ toast: true, position: 'bottom-end', timer: 2500, showConfirmButton: false, icon: 'success', title: 'Branding published' })
    } catch (e) {
      this.$swal({ icon: 'error', title: 'Unable to save', text: errorMessage(e) })
    } finally {
      this.saving = false
    }
  }

  discard() {
    if (this.saved) this.draft = clone(this.saved)
  }

  async reset() {
    const { value } = await this.$swal({
      title: 'Reset branding?',
      text: 'All branding, colors, texts and custom code go back to defaults. Uploaded files are kept.',
      icon: 'warning',
      showCancelButton: true,
      confirmButtonText: 'Reset',
    })
    if (!value) return
    const b = await api.resetBranding()
    this.saved = clone(b)
    this.draft = clone(b)
  }

  exportJson() {
    const blob = new Blob([JSON.stringify(this.draft, null, 2)], { type: 'application/json' })
    const a = document.createElement('a')
    a.href = URL.createObjectURL(blob)
    a.download = 'neko-rooms-branding.json'
    a.click()
    URL.revokeObjectURL(a.href)
  }

  async importJson(e: Event) {
    const input = e.target as HTMLInputElement
    const file = input.files && input.files[0]
    input.value = ''
    if (!file || !this.draft) return
    try {
      const data = JSON.parse(await file.text())
      const defaults = await api.brandingDefaults()
      // merge onto defaults so missing keys are filled in
      const merge = (base: any, over: any): any => { // eslint-disable-line
        if (Array.isArray(base) || typeof base !== 'object' || base === null) return over === undefined ? base : over
        const out = { ...base }
        Object.keys(over || {}).forEach(k => { out[k] = k in base ? merge(base[k], over[k]) : over[k] })
        return out
      }
      this.draft = merge(defaults, data)
      this.$swal({ toast: true, position: 'bottom-end', timer: 2500, showConfirmButton: false, icon: 'info', title: 'Imported, review and save to publish' })
    } catch (err) {
      this.$swal({ icon: 'error', title: 'Invalid file', text: String(err) })
    }
  }

  async deleteAsset(name: string) {
    const { value } = await this.$swal({ title: `Delete ${name}?`, text: 'Anything still using this file will show a broken image.', icon: 'warning', showCancelButton: true, confirmButtonText: 'Delete' })
    if (!value) return
    await api.deleteAsset(name)
    this.loadAssets()
  }

  async copy(url: string) {
    await copyText(location.origin + url)
  }

  mounted() {
    this.load()
  }

  beforeDestroy() {
    // leave without saving: restore published branding
    if (this.saved) {
      this.$store.commit('BRANDING_SET', clone(this.saved))
      this.$store.commit('DARK_SET', resolveDark(this.saved))
      this.$store.dispatch('APP_BRANDING').catch(() => { /* ignore */ })
    }
  }

  beforeRouteLeave(to: unknown, from: unknown, next: (ok?: boolean) => void) {
    if (this.dirty && !window.confirm('Discard unsaved branding changes?')) {
      next(false)
      return
    }
    next()
  }
}
</script>
