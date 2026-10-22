// The room player keeps a joined room alive while browsing the site: it
// fills the screen on the watch route and shrinks to a mini player elsewhere.
import Vue from 'vue'
import { RawLocation } from 'vue-router'
import { Room } from '@/api-ext'

export interface PlayerState {
  name: string
  invite: string
  url: string // room link with credentials, loaded in the player frame
  room: Room | null
  image: string // shown while the room connects
  from: DOMRect | null // where the join started, the player grows from there
  expanding: boolean // growing to full screen before the watch route opens
  returnTo: RawLocation | null // where "back" goes from the watch route
  session: number // changes for every joined room, recreates the frame
  openedAt: number
}

export const player: PlayerState = Vue.observable({
  name: '',
  invite: '',
  url: '',
  room: null,
  image: '',
  from: null,
  expanding: false,
  returnTo: null,
  session: 0,
  openedAt: 0,
})

export function openPlayer(opts: {
  name: string
  invite?: string
  url: string
  room?: Room | null
  image?: string
  from?: DOMRect | null
  returnTo?: RawLocation | null
}) {
  player.name = opts.name
  player.invite = opts.invite || ''
  player.url = opts.url
  player.room = opts.room || null
  player.image = opts.image || ''
  player.from = opts.from || null
  player.returnTo = opts.returnTo || null
  player.expanding = !!opts.from
  player.openedAt = Date.now()
  player.session++
}

export function closePlayer() {
  player.name = ''
  player.invite = ''
  player.url = ''
  player.room = null
  player.image = ''
  player.from = null
  player.expanding = false
  player.returnTo = null
}

export function isPlaying(name: string): boolean {
  return !!player.url && player.name === name
}
