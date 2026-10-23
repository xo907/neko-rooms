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
          <span class="nr-spinner" />
          <div v-if="mode === 'full'" class="nr-curtain-title">{{ title }}</div>
          <div v-if="mode === 'full'" class="nr-curtain-note">Joining room…</div>
        </div>
      </div>
    </transition>

    <!-- sits on the empty side of the neko header, reads as one bar; its own
         background also covers the neko logo should the frame not be styled -->
    <div v-if="mode === 'full'" class="nr-bar" :style="barStyle">
      <button type="button" class="nr-bar-btn" title="Back to the site, keep watching" @click="back">
        <v-icon>mdi-chevron-left</v-icon>
      </button>
      <span class="nr-bar-sep" />
      <div class="nr-bar-title">
        <span class="nr-bar-name">{{ title }}</span>
        <span v-if="room && room.ready" class="nr-bar-live">
          <span class="nr-dot" />{{ room.viewers }} watching
        </span>
      </div>
      <div v-if="members.length" class="nr-bar-members d-none d-md-flex">
        <UserAvatar v-for="m in members" :key="m" :name="m" :src="room && room.avatars ? room.avatars[m] : undefined" :size="24" ring tooltip />
      </div>
      <button type="button" class="nr-bar-btn" title="Continue on another device" @click="handoff">
        <v-icon small>mdi-cellphone-arrow-down</v-icon>
      </button>
      <button type="button" class="nr-bar-btn" :title="copied ? 'Link copied' : 'Copy link'" @click="share">
        <v-icon small>{{ copied ? 'mdi-check' : 'mdi-link-variant' }}</v-icon>
      </button>
      <button type="button" class="nr-bar-btn nr-bar-btn--leave" title="Leave the room" @click="leave">
        <v-icon small>mdi-logout-variant</v-icon><span class="d-none d-sm-inline">Leave</span>
      </button>
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
        <button type="button" title="Back to room" @click.stop="expand"><v-icon small>mdi-arrow-expand</v-icon></button>
        <button type="button" title="Leave room" @click.stop="leave"><v-icon small>mdi-close</v-icon></button>
      </div>
    </div>
  </div>
</template>

<style lang="scss" scoped>
$ease: cubic-bezier(.2, .8, .2, 1);

.nr-player {
  position: fixed;
  z-index: 100;
  overflow: hidden;
  background: var(--nr-bg);
}
.nr-player--full {
  inset: 0;
}
.nr-player--mini {
  right: 20px;
  bottom: 20px;
  width: clamp(240px, 26vw, 380px);
  aspect-ratio: 16 / 9;
  border-radius: 16px;
  background: #000;
  box-shadow: 0 0 0 1px var(--nr-line-strong), 0 30px 70px -20px rgba(0, 0, 0, .8);
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
  transition: opacity .5s $ease;
  &.is-ready { opacity: 1; }
}

.nr-curtain {
  position: absolute;
  inset: 0;
  z-index: 1;
  overflow: hidden;
}
.nr-curtain-image {
  position: absolute;
  inset: -40px;
  background-size: cover;
  background-position: center;
  transition: filter .7s $ease, transform .7s $ease;
}
.is-settled .nr-curtain-image {
  filter: blur(22px) brightness(.45) saturate(1.2);
  transform: scale(1.06);
}
.nr-curtain-body {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 14px;
  text-align: center;
  color: #fff;
  opacity: 0;
  transition: opacity .4s $ease .15s;
}
.is-settled .nr-curtain-body { opacity: 1; }
.nr-curtain-title {
  max-width: min(80vw, 640px);
  font-family: 'Instrument Serif', var(--nr-heading-font), Georgia, serif;
  font-size: clamp(2rem, 5vw, 3.4rem);
  line-height: 1.05;
  letter-spacing: -0.01em;
}
.nr-curtain-note {
  font-size: 13px;
  color: rgba(255, 255, 255, .7);
}
.nr-spinner {
  width: 22px;
  height: 22px;
  border-radius: 50%;
  border: 2px solid rgba(255, 255, 255, .2);
  border-top-color: var(--nr-primary);
  animation: nr-spin .9s linear infinite;
}
@keyframes nr-spin { to { transform: rotate(360deg); } }

.nr-curtain-leave-active { transition: opacity .5s $ease; }
.nr-curtain-leave-to { opacity: 0; }

.nr-bar {
  position: absolute;
  z-index: 2;
  top: 0;
  left: 0;
  height: 51px; // the neko header border stays visible below
  min-width: 200px; // wider than the neko logo
  max-width: calc(100% - 160px);
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 0 10px;
  background: var(--nr-bg);
  color: var(--nr-text);
  animation: nr-bar-in .5s $ease .2s both;
}
@keyframes nr-bar-in { from { opacity: 0; transform: translateY(-6px); } }

// same size and shape as the neko header buttons on the right
.nr-bar-btn {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-width: 34px;
  height: 34px;
  padding: 0 8px;
  border-radius: 10px;
  color: var(--nr-muted);
  font-size: 13px;
  font-weight: 500;
  transition: background-color .18s, color .18s;
  .v-icon { color: inherit !important; }
  &:hover, &:focus-visible { background: var(--nr-hover); color: var(--nr-text); }
}
.nr-bar-btn--leave:hover, .nr-bar-btn--leave:focus-visible {
  background: color-mix(in srgb, #f87171 12%, transparent);
  color: #fca5a5;
}
.nr-bar-sep {
  flex-shrink: 0;
  width: 1px;
  height: 18px;
  margin: 0 4px;
  background: var(--nr-line-strong);
}
.nr-bar-title {
  display: flex;
  flex-direction: column;
  justify-content: center;
  min-width: 0;
  margin-right: 6px;
  line-height: 1.2;
}
.nr-bar-name {
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
  font-size: 14px;
  font-weight: 600;
  letter-spacing: -0.01em;
}
.nr-bar-live {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-top: 2px;
  font-size: 11.5px;
  color: var(--nr-muted);
  white-space: nowrap;
  .nr-dot { width: 5px; height: 5px; }
}
.nr-bar-members {
  flex-shrink: 0;
  align-items: center;
  margin-right: 4px;
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
  padding: 10px;
  background: linear-gradient(180deg, transparent 45%, rgba(0, 0, 0, .7));
  color: #fff;
  opacity: 0;
  transition: opacity .2s;
}
.nr-player--mini:not(.is-ready) .nr-mini-ui { opacity: 1; }
.nr-mini-info {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  font-size: 13px;
  font-weight: 600;
}
.nr-mini-actions {
  display: flex;
  gap: 4px;
  button {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 30px;
    height: 30px;
    border-radius: 50%;
    background: rgba(10, 10, 12, .6);
    backdrop-filter: blur(8px);
    box-shadow: inset 0 0 0 1px rgba(255, 255, 255, .14);
    transition: background-color .18s;
    .v-icon { color: #fff !important; }
    &:hover { background: rgba(255, 255, 255, .18); }
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

  get members(): string[] {
    return this.room ? this.room.members.slice(0, 4) : []
  }

  get curtainStyle() {
    const tint = 'radial-gradient(120% 90% at 0% 0%, color-mix(in srgb, var(--nr-primary) 22%, transparent), transparent 60%), linear-gradient(135deg, var(--nr-surface), var(--nr-bg))'
    const image = this.state.image.replace(/"/g, '')
    return { backgroundImage: image ? `url("${image}"), ${tint}` : tint }
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
