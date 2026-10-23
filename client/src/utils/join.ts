import Vue from 'vue'
import { RawLocation } from 'vue-router'
// the same build the $swal plugin uses, so popups it opened can be closed here
import Swal from 'sweetalert2/dist/sweetalert2.js'
import { api, errorMessage, Room } from '@/api-ext'
import { isPlaying, openPlayer, player } from '@/player/state'

const NAME_KEY = 'neko-rooms-guest-name'

// resolveJoinUrl returns the room link with credentials, asking guests for
// a nickname first. It returns null when cancelled or not allowed.
export async function resolveJoinUrl(vm: Vue, name: string, invite?: string): Promise<string | null> {
  let displayName: string | undefined

  if (!vm.$store.getters.user) {
    let saved = ''
    try { saved = localStorage.getItem(NAME_KEY) || '' } catch { /* ignore */ }

    const res = await vm.$swal({
      title: 'Pick a nickname',
      input: 'text',
      inputValue: saved,
      inputPlaceholder: 'Your name',
      inputAttributes: { maxlength: '32', autocapitalize: 'off' },
      showCancelButton: true,
      confirmButtonText: 'Join room',
      inputValidator: (v: string) => (!v || !v.trim()) ? 'Please enter a name' : null,
    })
    if (!res.isConfirmed) return null

    displayName = String(res.value).trim()
    try { localStorage.setItem(NAME_KEY, displayName) } catch { /* ignore */ }
  }

  try {
    let res = await api.join(name, invite, displayName)
    if (res.waking) {
      if (!await waitAwake(vm, name, invite)) return null
      res = await api.join(name, invite, displayName)
    }
    return res.url || null
  } catch (e) {
    vm.$swal({
      title: 'Unable to join',
      text: errorMessage(e),
      icon: 'error',
    })
    return null
  }
}

// waitAwake shows a sleeping room starting up, false when cancelled
async function waitAwake(vm: Vue, name: string, invite?: string): Promise<boolean> {
  let cancelled = false
  vm.$swal({
    title: 'Waking up the room',
    text: 'It was asleep because nobody was in it. This takes a few seconds.',
    showConfirmButton: false,
    showCancelButton: true,
    cancelButtonText: 'Cancel',
    allowOutsideClick: false,
    didOpen: () => { Swal.showLoading() },
  }).then(() => { cancelled = true })

  // rooms whose health check never passes still open after a while
  const until = Date.now() + 60000
  while (!cancelled && Date.now() < until) {
    await new Promise(resolve => setTimeout(resolve, 2500))
    try {
      const room = await api.room(name, invite)
      if (room.ready) break
    } catch { /* try again */ }
  }
  if (cancelled) return false
  Swal.close()
  return true
}

export function watchRoute(name: string, invite?: string): RawLocation {
  return { name: 'watch', params: { name }, query: invite ? { invite } : {} }
}

export interface JoinOptions {
  invite?: string
  room?: Room | null
  from?: Element | null // the player grows from this element
  image?: string // preview shown while the room connects
}

// joinRoom opens a room in the player, growing it from the clicked element.
export async function joinRoom(vm: Vue, name: string, opts: JoinOptions = {}): Promise<void> {
  if (isPlaying(name)) {
    vm.$router.push(watchRoute(name, player.invite || opts.invite)).catch(() => { /* ignore */ })
    return
  }

  const url = await resolveJoinUrl(vm, name, opts.invite)
  if (!url) return

  openPlayer({
    name,
    invite: opts.invite,
    url,
    room: opts.room,
    image: opts.image,
    from: opts.from ? opts.from.getBoundingClientRect() : null,
    returnTo: vm.$route.fullPath,
  })

  // when growing from an element, the player opens the watch route itself
  if (!player.expanding) {
    vm.$router.push(watchRoute(name, opts.invite)).catch(() => { /* ignore */ })
  }
}

// share links use /r/{room}, the server gives chat apps a preview of the room
// there (they never see what follows a #) and sends visitors on to the app
function siteBase(): string {
  return location.href.split(/[?#]/)[0].replace(/[^/]*$/, '')
}

export function inviteLink(name: string, code: string): string {
  return `${siteBase()}r/${encodeURIComponent(name)}?invite=${encodeURIComponent(code)}`
}

export function roomLink(name: string): string {
  return `${siteBase()}r/${encodeURIComponent(name)}`
}

// appLink opens a room in the WatchParty Android app, on this server
export function appLink(name: string, invite?: string): string {
  const q = new URLSearchParams({ server: siteBase(), name })
  if (invite) q.set('invite', invite)
  return `watchparty://room?${q.toString()}`
}

export const isAndroid = /Android/i.test(navigator.userAgent)

// relativeTime says when something happens, e.g. "in 2 hours"
export function relativeTime(at: Date, now = Date.now()): string {
  const mins = Math.round((at.getTime() - now) / 60000)
  if (mins <= 0) return 'now'
  const rtf = new Intl.RelativeTimeFormat(undefined, { numeric: 'auto' })
  if (mins < 60) return rtf.format(mins, 'minute')
  if (mins < 36 * 60) return rtf.format(Math.round(mins / 60), 'hour')
  return rtf.format(Math.round(mins / 1440), 'day')
}

export async function copyText(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch {
    const el = document.createElement('textarea')
    el.value = text
    document.body.appendChild(el)
    el.select()
    const ok = document.execCommand('copy')
    el.remove()
    return ok
  }
}
