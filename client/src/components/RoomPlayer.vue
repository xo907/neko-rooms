<template>
  <div
    v-if="active"
    class="nr-player"
    :class="['nr-player--' + mode, { 'is-ready': ready, 'is-settled': settled }]"
  >
    <iframe
      :key="state.session"
      ref="frame"
      :src="src"
      class="nr-frame"
      :class="{ 'is-ready': ready }"
      :title="title"
      allow="autoplay; fullscreen; clipboard-read; clipboard-write; microphone; camera; display-capture"
      allowfullscreen
      @load="onLoad"
    />

    <!-- covers the connecting neko client, grows from the clicked thumbnail -->
    <transition name="nr-curtain">
      <div v-if="!ready" class="nr-curtain">
        <div class="nr-curtain-image" :style="curtainStyle" />
        <div class="nr-curtain-body">
          <template v-if="mode === 'full'">
            <span class="nr-curtain-eyebrow">Joining</span>
            <div class="nr-curtain-title">{{ title }}</div>
            <div v-if="room && room.now_playing" class="nr-curtain-note">{{ room.now_playing }}</div>
            <ol class="nr-steps" aria-live="polite">
              <li v-for="(step, i) in steps" :key="step" :class="{ 'is-done': i < phase, 'is-active': i === phase }">
                <span class="nr-step-mark">
                  <v-icon v-if="i < phase" x-small>mdi-check</v-icon>
                  <span v-else-if="i === phase" class="nr-spinner" />
                </span>
                <span>{{ step }}</span>
              </li>
            </ol>
            <button type="button" class="nr-curtain-cancel" @click="leave">Cancel</button>
          </template>
          <span v-else class="nr-spinner" />
        </div>
      </div>
    </transition>

    <!-- sits on the empty side of the neko header, reads as one bar; its own
         background also covers the neko logo should the frame not be styled -->
    <div v-if="mode === 'full'" class="nr-bar" :style="barStyle">
      <button type="button" class="nr-bar-btn" title="Back to the site, keep watching" aria-label="Back to the site, keep watching" @click="back">
        <v-icon small>mdi-chevron-left</v-icon>
      </button>
      <div class="nr-bar-title">
        <span class="nr-bar-name">{{ title }}</span>
        <span v-if="room && room.ready" class="nr-bar-live">
          <span class="nr-dot" />{{ room.viewers }} watching<template v-if="elapsed"> · {{ elapsed }}</template>
        </span>
      </div>
      <span class="flex-grow-1" />
      <div v-if="members.length" class="nr-bar-members d-none d-md-flex">
        <UserAvatar v-for="m in members" :key="m" :name="m" :src="room && room.avatars ? room.avatars[m] : undefined" :size="24" ring tooltip />
      </div>
      <span class="nr-bar-sep d-none d-md-block" />
      <button type="button" class="nr-bar-btn" title="Continue on another device" aria-label="Continue on another device" @click="handoff">
        <v-icon small>mdi-cellphone</v-icon>
      </button>
      <button type="button" class="nr-bar-btn" :class="{ 'is-done': copied }" :title="copied ? 'Link copied' : 'Copy link'" :aria-label="copied ? 'Link copied' : 'Copy link'" @click="share">
        <v-icon small>{{ copied ? 'mdi-check' : 'mdi-link-variant' }}</v-icon>
      </button>
      <button type="button" class="nr-bar-leave" title="Leave the room" @click="leave">Leave</button>
    </div>

    <!-- mini player -->
    <!-- the whole mini player expands on click, not the player root: a click
         on "back" would bubble up to it after switching to mini -->
    <div v-if="mode === 'mini'" class="nr-mini-ui" @click="expand">
      <div class="nr-mini-info">
        <span v-if="room && room.ready" class="nr-dot" />
        <span class="text-truncate">{{ title }}</span>
      </div>
      <div class="nr-mini-actions">
        <button type="button" title="Back to room" aria-label="Back to room" @click.stop="expand"><v-icon small>mdi-arrow-expand</v-icon></button>
        <button type="button" title="Leave room" aria-label="Leave room" @click.stop="leave"><v-icon small>mdi-close</v-icon></button>
      </div>
    </div>
  </div>
</template>

<style lang="scss" scoped>
$ease: cubic-bezier(.22, 1, .36, 1);

.nr-player {
  position: fixed;
  z-index: 100;
  overflow: hidden;
  background: #000;
}
.nr-player--full {
  inset: 0;
}
.nr-player--mini {
  right: 16px;
  bottom: 16px;
  width: clamp(240px, 26vw, 380px);
  aspect-ratio: 16 / 9;
  border-radius: var(--radius-lg);
  box-shadow: 0 0 0 1px var(--border-hover), 0 24px 60px -20px rgba(0, 0, 0, .8);
  cursor: pointer;
  .nr-frame { pointer-events: none; }
  &:hover .nr-mini-ui { opacity: 1; }
}
@media (max-width: 600px) {
  .nr-player--mini { right: 12px; bottom: 12px; width: min(62vw, 300px); }
}

.nr-frame {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  border: 0;
  opacity: 0;
  transition: opacity .4s $ease;
  &.is-ready { opacity: 1; }
}

.nr-curtain {
  position: absolute;
  inset: 0;
  z-index: 1;
  overflow: hidden;
  background: #000;
}
.nr-curtain-image {
  position: absolute;
  inset: -60px;
  background-size: cover;
  background-position: center;
  transition: filter .6s $ease, transform .6s $ease;
}
.is-settled .nr-curtain-image {
  filter: blur(28px) brightness(.35) saturate(1.1);
  transform: scale(1.04);
}
.nr-curtain-body {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 24px;
  text-align: center;
  color: var(--text-primary);
  background: rgba(9, 9, 11, .55);
  opacity: 0;
  transition: opacity .3s $ease .1s;
}
.is-settled .nr-curtain-body { opacity: 1; }
.nr-curtain-eyebrow {
  font: 500 11.5px var(--font-mono);
  letter-spacing: .08em;
  text-transform: uppercase;
  color: var(--text-secondary);
}
.nr-curtain-title {
  max-width: min(80vw, 640px);
  font-family: var(--font-sans);
  font-size: clamp(28px, 4vw, 40px);
  font-weight: 600;
  line-height: 1.05;
  letter-spacing: -0.04em;
}
.nr-curtain-note {
  max-width: min(80vw, 520px);
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  font-size: 14px;
  color: var(--text-secondary);
}
.nr-steps {
  width: min(380px, 100%);
  margin: 16px 0 8px;
  padding: 0;
  list-style: none;
  border: 1px solid var(--border-hover);
  border-radius: var(--radius-lg);
  background: rgba(17, 17, 20, .88);
  overflow: hidden;
  text-align: left;
  li {
    display: flex;
    align-items: center;
    gap: 12px;
    height: 36px;
    padding: 0 16px;
    border-top: 1px solid var(--border-subtle);
    font: 400 12.5px var(--font-mono);
    color: var(--text-tertiary);
    &:first-child { border-top: 0; }
    &.is-active { color: var(--text-primary); }
    &.is-done { color: var(--text-secondary); }
  }
}
.nr-step-mark {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 14px;
  height: 14px;
  flex: none;
  .v-icon { color: var(--status-online) !important; }
}
.nr-spinner {
  display: inline-block;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  border: 2px solid var(--border-hover);
  border-top-color: var(--text-primary);
  animation: nr-spin .9s linear infinite;
}
.nr-curtain-cancel {
  height: 32px;
  padding: 0 12px;
  border-radius: var(--radius-md);
  border: 1px solid var(--border-hover);
  background: transparent;
  color: var(--text-secondary);
  font: 500 13px var(--font-sans);
  cursor: pointer;
  transition: color var(--dur) $ease, border-color var(--dur) $ease;
  &:hover { color: var(--text-primary); border-color: var(--border-focus); }
}

.nr-curtain-leave-active { transition: opacity .4s $ease; }
.nr-curtain-leave-to { opacity: 0; }

.nr-bar {
  position: absolute;
  z-index: 2;
  top: 0;
  left: 0;
  height: 55px; // the neko header border stays visible below
  min-width: 200px; // wider than the neko logo
  max-width: calc(100% - 160px);
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 8px 0 12px;
  background: var(--bg-canvas);
  color: var(--text-primary);
  animation: nr-bar-in .4s $ease .15s both;
}
@keyframes nr-bar-in { from { opacity: 0; } }

.nr-bar-btn {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  border-radius: var(--radius-md);
  color: var(--text-secondary);
  transition: background-color var(--dur) $ease, color var(--dur) $ease;
  .v-icon { color: inherit !important; }
  &:hover, &:focus-visible { background: var(--bg-surface-2); color: var(--text-primary); }
  &.is-done { color: var(--status-online); }
}
.nr-bar-leave {
  flex-shrink: 0;
  height: 32px;
  padding: 0 12px;
  margin-left: 4px;
  border-radius: var(--radius-md);
  border: 1px solid var(--border-subtle);
  background: var(--bg-surface-2);
  color: var(--text-primary);
  font: 500 13px var(--font-sans);
  transition: background-color var(--dur) $ease, border-color var(--dur) $ease, color var(--dur) $ease;
  &:hover, &:focus-visible { color: var(--status-danger-ink); border-color: rgba(239, 68, 68, .35); background: rgba(239, 68, 68, .08); }
}
.nr-bar-sep {
  flex-shrink: 0;
  width: 1px;
  height: 20px;
  margin: 0 4px;
  background: var(--border-subtle);
}
.nr-bar-title {
  display: flex;
  flex-direction: column;
  justify-content: center;
  gap: 4px;
  min-width: 0;
  margin-left: 4px;
}
.nr-bar-name {
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  font-size: 14px;
  font-weight: 600;
  letter-spacing: -0.025em;
  line-height: 1.1;
}
.nr-bar-live {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font: 400 11px var(--font-mono);
  color: var(--text-tertiary);
  white-space: nowrap;
  line-height: 1;
  .nr-dot { width: 6px; height: 6px; }
}
.nr-bar-members {
  flex-shrink: 0;
  align-items: center;
  > * + * { margin-left: -6px; }
}

.nr-mini-ui {
  position: absolute;
  z-index: 2;
  inset: 0;
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: 8px;
  padding: 8px;
  color: var(--text-primary);
  opacity: 0;
  transition: opacity var(--dur) $ease;
}
.nr-player--mini:not(.is-ready) .nr-mini-ui { opacity: 1; }
.nr-mini-info {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  height: 28px;
  padding: 0 10px;
  border-radius: var(--radius-sm);
  background: rgba(9, 9, 11, .82);
  border: 1px solid var(--border-hover);
  font-size: 12.5px;
  font-weight: 600;
  letter-spacing: -0.02em;
}
.nr-mini-actions {
  display: flex;
  gap: 4px;
  button {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    border-radius: var(--radius-sm);
    background: rgba(9, 9, 11, .82);
    border: 1px solid var(--border-hover);
    transition: background-color var(--dur) $ease;
    .v-icon { color: var(--text-primary) !important; }
    &:hover { background: var(--bg-surface-3); }
  }
}
</style>

<script lang="ts">
import { Vue, Component, Watch } from 'vue-property-decorator'
import { api, Room } from '@/api-ext'
import { closePlayer, player, PlayerState } from '@/player/state'
import { copyText, inviteLink, roomLink, watchRoute } from '@/utils/join'
import UserAvatar from '@/components/community/UserAvatar.vue'
import QRCode from 'qrcode'

type Mode = 'full' | 'mini' | null

const EASE = 'cubic-bezier(.2, .8, .2, 1)'
const READY_TIMEOUT = 20000

// styles for the neko client inside the frame, see decorate()
const FRAME_CSS = `
html.nr-embedded #neko .header .neko { visibility: hidden !important; }
html.nr-embedded #neko .room-menu .settings li:has(> .fa-question-circle),
html.nr-embedded #neko .room-menu .settings li:has(> select) { display: none !important; }
html.nr-mini, html.nr-mini body { overflow: hidden !important; }
html.nr-mini #neko .neko-main .header-container,
html.nr-mini #neko .neko-main .room-container,
html.nr-mini #neko .neko-menu,
html.nr-mini #neko .video .player .video-menu,
html.nr-mini .emote-container,
html.nr-mini .vue-notification-group { display: none !important; }
html.nr-mini #neko .neko-main { height: 100vh !important; min-width: 0 !important; }
html.nr-mini #neko .video .player .player-container { border-radius: 0 !important; box-shadow: none !important; }
`

function reducedMotion(): boolean {
  return !!window.matchMedia && window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

@Component({
  components: {
    UserAvatar,
  }
})
export default class RoomPlayer extends Vue {
  state: PlayerState = player
  ready = false
  settled = false
  copied = false
  barWidth = 0
  // joining: 0 the page is loading, 1 neko is connecting, 2 the video starts
  phase = 0
  now = Date.now()

  private poller = 0
  private roomTimer = 0
  private startedAt = 0

  get active(): boolean {
    return !!this.state.url
  }

  get onWatch(): boolean {
    return this.$route.name === 'watch' && this.$route.params.name === this.state.name
  }

  get mode(): Mode {
    if (!this.active) return null
    return this.state.expanding || this.onWatch ? 'full' : 'mini'
  }

  // a fresh page for every join, never a stale cached room page
  get src(): string {
    const url = this.state.url
    return url + (url.includes('?') ? '&' : '?') + 'nr=' + this.state.openedAt
  }

  get room(): Room | null {
    return this.state.room
  }

  get title(): string {
    return this.room ? this.room.title : this.state.name
  }

  get steps(): string[] {
    return ['Opening room', 'Connecting', 'Starting video']
  }

  // how long this visit has been going, a quiet clock in the bar
  get elapsed(): string {
    if (!this.startedAt || !this.ready) return ''
    const t = Math.max(0, Math.floor((this.now - this.startedAt) / 1000))
    const h = Math.floor(t / 3600)
    const m = Math.floor((t % 3600) / 60)
    const pad = (n: number) => (n < 10 ? '0' : '') + n
    return (h ? h + ':' + pad(m) : String(m)) + ':' + pad(t % 60)
  }

  get members(): string[] {
    return this.room ? this.room.members.slice(0, 4) : []
  }

  get curtainStyle() {
    const image = this.state.image.replace(/"/g, '')
    return image ? { backgroundImage: `url("${image}")` } : {}
  }


  get barStyle() {
    return this.barWidth > 0 ? { maxWidth: this.barWidth + 'px' } : {}
  }

  // a method, not a getter: getters are cached and $refs are not reactive
  frameEl(): HTMLIFrameElement | null {
    return (this.$refs.frame as HTMLIFrameElement) || null
  }

  // the frame document, null while loading or when it is cross-origin
  frameDoc(): Document | null {
    const frame = this.frameEl()
    try {
      return frame ? frame.contentDocument : null
    } catch {
      return null
    }
  }

  @Watch('state.session')
  async onSession() {
    this.stop()
    this.ready = false
    this.settled = false
    this.barWidth = 0
    this.phase = 0
    if (!this.active) return

    this.startedAt = Date.now()
    this.poller = window.setInterval(this.poll, 250)
    this.loadRoom()
    this.roomTimer = window.setInterval(this.loadRoom, 15000)

    await this.$nextTick()
    const el = this.$el as HTMLElement
    if (!el || !el.animate) {
      this.settle()
      return
    }

    const from = this.state.from
    if (from && !reducedMotion()) {
      await this.flip(from, 14, 560)
    } else {
      await el.animate([{ opacity: 0, transform: 'scale(.98)' }, { opacity: 1, transform: 'none' }], { duration: 320, easing: EASE }).finished.catch(() => { /* ignore */ })
    }
    this.settle()
  }

  // the player covers the screen now, open the watch route below it
  settle() {
    this.settled = true
    if (this.state.expanding) {
      const done = () => { this.state.expanding = false }
      this.$router.push(watchRoute(this.state.name, this.state.invite || undefined)).then(done, done)
    }
  }

  @Watch('mode')
  onMode(next: Mode, prev: Mode) {
    this.decorate()
    if (!next || !prev || next === prev || this.state.expanding) return
    const el = this.$el as HTMLElement
    if (!el || !el.getBoundingClientRect || reducedMotion()) return

    // watchers run before the DOM updates, so this is the old geometry
    const from = el.getBoundingClientRect()
    this.$nextTick(() => {
      this.flip(from, prev === 'mini' ? 16 : 0, 480)
      const frame = this.frameEl()
      if (next === 'full' && this.ready && frame) frame.focus()
    })
  }

  // flip animates the player from the given rect to where it is now
  flip(from: DOMRect, radius: number, duration: number): Promise<unknown> {
    const el = this.$el as HTMLElement
    const to = el.getBoundingClientRect()
    if (!to.width || !to.height) return Promise.resolve()

    const sx = from.width / to.width
    const sy = from.height / to.height
    const dx = from.left - to.left
    const dy = from.top - to.top
    const toRadius = this.mode === 'mini' ? 16 : 0

    return el.animate([
      { transformOrigin: '0 0', transform: `translate(${dx}px, ${dy}px) scale(${sx}, ${sy})`, borderRadius: `${radius / sx}px / ${radius / sy}px` },
      { transformOrigin: '0 0', transform: 'none', borderRadius: `${toRadius}px` },
    ], { duration, easing: EASE }).finished.catch(() => { /* ignore */ })
  }

  onLoad() {
    this.decorate()
    // cross-origin rooms can not be inspected, show them once loaded
    if (!this.frameDoc()) this.reveal()
  }

  poll() {
    const doc = this.frameDoc()
    if (doc && doc.documentElement) {
      this.decorate()

      if (!this.ready) {
        const video = doc.querySelector('video')
        const lobby = !doc.getElementById('neko') && doc.querySelector('.swal2-popup')
        if (doc.getElementById('neko')) this.phase = video ? 2 : 1
        if (lobby || (video && video.readyState >= 2)) this.reveal()
      }

      // keep the bar clear of the neko header buttons
      const menu = doc.querySelector('#neko .header .menu')
      if (menu && this.mode === 'full') {
        const left = Math.floor(menu.getBoundingClientRect().left) - 12
        if (left > 0 && left !== this.barWidth) this.barWidth = left
      }
    }

    if (!this.ready && Date.now() - this.startedAt > READY_TIMEOUT) this.reveal()
    // once a second is enough for the clock
    if (this.ready && Date.now() - this.now >= 1000) this.now = Date.now()
  }

  reveal() {
    if (this.ready) return
    this.ready = true
    const frame = this.frameEl()
    if (this.mode === 'full' && frame) frame.focus()
  }

  // decorate marks the neko client as embedded, and as mini player
  decorate() {
    const doc = this.frameDoc()
    if (!doc || !doc.documentElement || !doc.head) return

    if (!doc.getElementById('nr-player-style')) {
      const style = doc.createElement('style')
      style.id = 'nr-player-style'
      style.textContent = FRAME_CSS
      doc.head.appendChild(style)
    }
    doc.documentElement.classList.add('nr-embedded')
    doc.documentElement.classList.toggle('nr-mini', this.mode === 'mini')
  }

  async loadRoom() {
    if (!this.active || document.hidden) return
    const name = this.state.name
    try {
      const room = await api.room(name, this.state.invite || undefined)
      if (this.state.name === name) this.state.room = room
    } catch {
      // keep the last known state
    }
  }

  expand() {
    this.$router.push(watchRoute(this.state.name, this.state.invite || undefined)).catch(() => { /* ignore */ })
  }

  back() {
    const to = this.state.returnTo || { name: 'room', params: { name: this.state.name } }
    this.$router.push(to).catch(() => {
      this.$router.push({ name: 'home' }).catch(() => { /* ignore */ })
    })
  }

  async leave() {
    const wasFull = this.mode === 'full'
    const name = this.state.name
    const returnTo = this.state.returnTo

    const el = this.$el as HTMLElement
    if (el && el.animate && !reducedMotion()) {
      await el.animate([{ opacity: 1, transform: 'none' }, { opacity: 0, transform: 'scale(.97)' }], { duration: 220, easing: EASE }).finished.catch(() => { /* ignore */ })
    }

    closePlayer()
    if (wasFull) {
      this.$router.push(returnTo || { name: 'room', params: { name } }).catch(() => { /* ignore */ })
    }
  }

  // handoff shows the room as a QR code, a phone camera opens it there
  async handoff() {
    const link = this.state.invite ? inviteLink(this.state.name, this.state.invite) : roomLink(this.state.name)
    const image = await QRCode.toDataURL(link, { margin: 1, width: 240, errorCorrectionLevel: 'M' })
    this.$swal({
      title: 'Continue on another device',
      text: 'Scan it with your phone camera to open this room there.',
      imageUrl: image,
      imageWidth: 240,
      imageHeight: 240,
      imageAlt: link,
      confirmButtonText: 'Done',
    })
  }

  async share() {
    const link = this.state.invite ? inviteLink(this.state.name, this.state.invite) : roomLink(this.state.name)
    this.copied = await copyText(link)
    setTimeout(() => { this.copied = false }, 2000)
  }

  // the app sets the page title on route changes, this runs after it
  @Watch('room')
  @Watch('mode')
  @Watch('$route')
  setTitle() {
    if (this.mode === 'full') {
      document.title = this.title + ' · ' + document.title.split(' · ').pop()
    }
  }

  @Watch('active')
  onActive(active: boolean) {
    if (!active) this.stop()
  }

  stop() {
    window.clearInterval(this.poller)
    window.clearInterval(this.roomTimer)
  }

  beforeDestroy() {
    this.stop()
  }
}
</script>
