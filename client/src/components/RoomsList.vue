<template>
  <div>
    <div v-if="loading && !rooms.length" class="nr-rows" aria-busy="true">
      <div v-for="n in 3" :key="'sk' + n" class="nr-row nr-row--skeleton">
        <span class="nr-thumb" /><span class="nr-sk-line" />
      </div>
    </div>

    <div v-else-if="!rooms.length" class="nr-empty-state">
      <div class="nr-empty-icon"><v-icon>mdi-television-play</v-icon></div>
      <h2>No watch-party rooms running</h2>
      <p>A room is a shared browser in its own container. Spin one up and send the link around.</p>
      <button type="button" class="nr-btn is-primary" @click="$emit('create')"><v-icon small>mdi-plus</v-icon>New room</button>
    </div>

    <div v-else-if="!shown.length" class="nr-empty-state is-small">
      <p>No {{ filter }} rooms.</p>
    </div>

    <div v-else class="nr-rows" role="table" aria-label="Rooms">
      <div class="nr-row nr-row--head" role="row">
        <span class="nr-label" role="columnheader">Room</span>
        <span class="nr-label" role="columnheader">Status</span>
        <span class="nr-label nr-col-hide" role="columnheader">Image</span>
        <span class="nr-label nr-col-hide" role="columnheader">Viewers</span>
        <span class="nr-label nr-col-hide" role="columnheader">Created</span>
        <span class="nr-label nr-col-actions" role="columnheader">Actions</span>
      </div>

      <div v-for="item in shown" :key="item.id" class="nr-row" role="row">
        <div class="nr-room" role="cell">
          <img
            v-if="item.running && !thumbFailed[item.name]"
            :src="thumbUrl(item)"
            class="nr-thumb"
            alt=""
            @error="$set(thumbFailed, item.name, true)"
          >
          <span v-else class="nr-thumb">
            <span v-if="statusOf(item).key === 'starting'" class="nr-dot is-amber" />
          </span>
          <div class="nr-room-text">
            <router-link :to="{ name: 'room', params: { name: item.name } }" class="nr-room-name" :class="{ 'is-off': !item.running }">
              {{ titleOf(item) }}
            </router-link>
            <span class="nr-mono nr-room-sub">{{ item.name }}<template v-if="playingOf(item)"> · {{ playingOf(item) }}</template></span>
          </div>
        </div>

        <div role="cell">
          <span class="nr-status" :class="'is-' + statusOf(item).tone" :title="item.status">{{ statusOf(item).label }}</span>
        </div>

        <div class="nr-image nr-col-hide" role="cell" :title="item.neko_image">
          <span class="nr-tag">{{ engineOf(item.neko_image).name }}</span>
          <span v-if="item.is_outdated" class="nr-tag is-update" title="A newer version of this image is here. Update it from the menu.">update</span>
          <span v-else-if="engineOf(item.neko_image).tag" class="nr-mono nr-image-tag">{{ engineOf(item.neko_image).tag }}</span>
        </div>

        <div class="nr-col-hide" role="cell">
          <span class="nr-tag">
            <v-icon>mdi-account-outline</v-icon>{{ viewersOf(item) }}<template v-if="item.max_connections > 0"> / {{ item.max_connections }}</template><template v-else> · mux</template>
          </span>
        </div>

        <div class="nr-col-hide nr-mono nr-created" role="cell">{{ item.created | timeago }}</div>

        <div class="nr-actions" role="cell">
          <button v-if="item.running" type="button" class="nr-btn is-sm" :class="isFirstRunning(item) ? 'is-primary' : 'is-secondary'" @click="join(item)">
            <v-icon small>mdi-play</v-icon>Join
          </button>
          <RoomActionBtn v-else action="start" variant="button" :roomId="item.id" :disabled="item.running" />

          <v-tooltip bottom open-delay="300">
            <template v-slot:activator="{ on, attrs }">
              <button v-bind="attrs" v-on="on" type="button" class="nr-btn is-icon is-sm" :class="{ 'is-done': copied === item.id }" aria-label="Copy invite link" @click="copyInvite(item)">
                <v-icon small>{{ copied === item.id ? 'mdi-check' : 'mdi-content-copy' }}</v-icon>
              </button>
            </template>
            <span>{{ copied === item.id ? 'Copied' : 'Copy invite link' }}</span>
          </v-tooltip>

          <v-tooltip bottom open-delay="300">
            <template v-slot:activator="{ on, attrs }">
              <router-link v-bind="attrs" v-on="on" :to="{ name: 'room', params: { name: item.name } }" class="nr-btn is-icon is-sm" aria-label="Configure">
                <v-icon small>mdi-tune-variant</v-icon>
              </router-link>
            </template>
            <span>Room page: visibility, invites &amp; details</span>
          </v-tooltip>

          <v-menu offset-y left nudge-bottom="4" min-width="224" transition="slide-y-transition">
            <template v-slot:activator="{ on, attrs }">
              <button v-bind="attrs" v-on="on" type="button" class="nr-btn is-icon is-sm" aria-label="More actions">
                <v-icon small>mdi-dots-horizontal</v-icon>
              </button>
            </template>
            <v-list dense>
              <v-list-item :to="{ name: 'room', params: { name: item.name } }">
                <v-list-item-title>Room page</v-list-item-title>
                <span class="nr-mono nr-menu-hint">/r/{{ item.name }}</span>
              </v-list-item>
              <v-list-item :href="item.url" target="_blank" rel="noopener" :disabled="!item.running">
                <v-list-item-title>Open neko directly</v-list-item-title>
              </v-list-item>
              <v-list-item @click="roomId = item.id; dialog = true">
                <v-list-item-title>Details &amp; logs</v-list-item-title>
              </v-list-item>
              <v-divider class="my-1" />
              <RoomActionBtn action="restart" variant="item" :roomId="item.id" :disabled="!item.running" />
              <RoomActionBtn action="pause" variant="item" :roomId="item.id" :disabled="!item.running" />
              <RoomActionBtn action="stop" variant="item" :roomId="item.id" :disabled="!item.running && !item.paused" />
              <RoomActionBtn v-if="!item.running" action="start" variant="item" :roomId="item.id" />
              <v-divider class="my-1" />
              <RoomActionBtn action="upgrade" variant="item" :roomId="item.id" :roomName="item.name" :highlight="item.is_outdated" @done="$emit('refresh')" />
              <RoomActionBtn action="recreate" variant="item" :roomId="item.id" />
            </v-list>
          </v-menu>

          <RoomActionBtn action="remove" :roomId="item.id" />
        </div>
      </div>
    </div>

    <v-dialog v-model="dialog" max-width="920px">
      <v-card>
        <v-card-title>Room details</v-card-title>
        <v-card-text>
          <RoomInfo v-if="dialog" :roomId="roomId" />
        </v-card-text>
        <v-card-actions>
          <v-spacer></v-spacer>
          <v-btn text @click="dialog = false">Close</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<style lang="scss" scoped>
.nr-rows {
  background: var(--bg-surface-1);
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-lg);
}
.nr-row {
  display: grid;
  grid-template-columns: minmax(0, 2.3fr) 128px minmax(0, 1.4fr) 112px 112px 236px;
  align-items: center;
  gap: 16px;
  min-height: 61px;
  padding: 12px 16px;
  border-top: 1px solid var(--border-subtle);
  transition: background-color var(--dur) var(--ease);
  &:first-child { border-top: 0; }
  &:not(.nr-row--head):hover { background: rgba(255, 255, 255, .015); }
}
.nr-row--head { min-height: 0; }
.nr-col-actions { text-align: right; }
.nr-row--skeleton {
  grid-template-columns: 64px minmax(0, 1fr);
  .nr-sk-line { height: 10px; width: 40%; border-radius: 4px; background: var(--bg-surface-2); animation: nr-pulse 1.6s ease-in-out infinite; }
  .nr-thumb { animation: nr-pulse 1.6s ease-in-out infinite; }
}

.nr-room { display: flex; align-items: center; gap: 12px; min-width: 0; }
.nr-thumb {
  flex: none;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 64px;
  height: 36px;
  border-radius: 5px;
  border: 1px solid var(--border-subtle);
  background: var(--bg-surface-2);
  object-fit: cover;
}
.nr-dot.is-amber { background: var(--status-provisioning); box-shadow: 0 0 8px var(--status-provisioning); }
.nr-room-text { display: flex; flex-direction: column; gap: 4px; min-width: 0; }
.nr-room-name {
  font-size: 14px;
  font-weight: 500;
  letter-spacing: -0.02em;
  color: var(--text-primary) !important;
  text-decoration: none;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  &:hover { text-decoration: underline; text-underline-offset: 3px; text-decoration-color: var(--border-focus); }
  &.is-off { color: var(--text-secondary) !important; }
}
.nr-room-sub {
  font-size: 11.5px;
  color: var(--text-tertiary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.nr-image { display: flex; align-items: center; gap: 8px; min-width: 0; }
.nr-image-tag { font-size: 11.5px; color: var(--text-tertiary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.nr-created { font-size: 12.5px; color: var(--text-secondary); white-space: nowrap; }
.nr-actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 4px;
}
.nr-menu-hint { margin-left: 16px; font-size: 11px; color: var(--text-tertiary); white-space: nowrap; }
.nr-empty-state.is-small { padding: 32px 24px; }

@media (max-width: 1000px) {
  .nr-row { grid-template-columns: minmax(0, 1fr) auto; row-gap: 12px; }
  .nr-row > [role='cell']:nth-child(2) { justify-self: end; }
  .nr-actions { grid-column: 1 / -1; justify-content: flex-start; }
  .nr-col-hide { display: none; }
  .nr-row--head { display: none; }
  .nr-row:nth-child(2) { border-top: 0; }
}
</style>

<script lang="ts">
import { Vue, Component, Prop } from 'vue-property-decorator'
import RoomInfo from '@/components/RoomInfo.vue'
import RoomActionBtn from '@/components/RoomActionBtn.vue'
import { api, Room } from '@/api-ext'
import { RoomEntry } from '@/api/index'
import { copyText, inviteLink, joinRoom, roomLink } from '@/utils/join'

interface Status {
  key: string
  label: string
  tone: string
}

@Component({
  components: {
    RoomInfo,
    RoomActionBtn,
  }
})
export default class RoomsList extends Vue {
  @Prop(Boolean) readonly loading!: boolean
  @Prop({ type: String, default: 'all' }) readonly filter!: string
  @Prop({ type: Number, default: 0 }) readonly tick!: number

  public dialog = false
  public roomId = ''
  public copied = ''
  public thumbFailed: Record<string, boolean> = {}

  private copiedTimer = 0

  get rooms(): RoomEntry[] {
    return this.$store.state.rooms
  }

  get shown(): RoomEntry[] {
    if (this.filter === 'running') return this.rooms.filter(r => r.running)
    if (this.filter === 'stopped') return this.rooms.filter(r => !r.running)
    return this.rooms
  }

  // community details of each room (title, viewers, what is playing)
  get directory(): Record<string, Room> {
    const map: Record<string, Room> = {}
    ;(this.$store.state.app.directory || []).forEach((r: Room) => { map[r.name] = r })
    return map
  }

  titleOf(item: RoomEntry): string {
    const r = this.directory[item.name || '']
    return (r && r.title) || item.name || ''
  }

  playingOf(item: RoomEntry): string {
    const r = this.directory[item.name || '']
    return r && item.running ? (r.now_playing || '') : ''
  }

  viewersOf(item: RoomEntry): number | string {
    const r = this.directory[item.name || '']
    if (!item.running) return '—'
    return r ? r.viewers : 0
  }

  thumbUrl(item: RoomEntry): string {
    return api.thumbnailUrl(item.name || '', this.tick)
  }

  statusOf(item: RoomEntry): Status {
    const st = (item.status || '').toLowerCase()
    if (item.paused) return { key: 'paused', label: 'Paused', tone: 'plain' }
    if (!item.running) return { key: 'stopped', label: 'Stopped', tone: 'stopped' }
    if (st.includes('unhealthy')) return { key: 'unhealthy', label: 'Unhealthy', tone: 'warn' }
    if (st.includes('starting') || item.is_ready === false) return { key: 'starting', label: 'Starting', tone: 'starting' }
    return { key: 'running', label: 'Running', tone: 'online' }
  }

  // engineOf turns ghcr.io/m1k1o/neko/firefox:latest or m1k1o/neko:firefox
  // into the browser it runs and its tag
  engineOf(image?: string): { name: string; tag: string } {
    const last = (image || '').split('/').pop() || ''
    const [base, tag] = last.split(':')
    if (base === 'neko' && tag) return { name: tag, tag: '' }
    return { name: base || image || '', tag: tag ? ':' + tag : '' }
  }

  isFirstRunning(item: RoomEntry): boolean {
    const first = this.shown.find(r => r.running)
    return !!first && first.id === item.id
  }

  join(item: RoomEntry) {
    const room = this.directory[item.name || '']
    joinRoom(this, item.name || '', { room, invite: room && room.invite_code ? room.invite_code : undefined })
  }

  async copyInvite(item: RoomEntry) {
    const room = this.directory[item.name || '']
    const link = room && room.invite_code ? inviteLink(item.name || '', room.invite_code) : roomLink(item.name || '')
    if (await copyText(link)) {
      this.copied = item.id || ''
      window.clearTimeout(this.copiedTimer)
      this.copiedTimer = window.setTimeout(() => { this.copied = '' }, 1500)
    }
  }

  beforeDestroy() {
    window.clearTimeout(this.copiedTimer)
  }
}
</script>
