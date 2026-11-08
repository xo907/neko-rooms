<template>
  <div class="nr-deck nr-container">
    <div class="nr-deck-head">
      <div class="nr-deck-title">
        <h1 class="nr-page-title">{{ isAdmin ? 'All rooms' : 'My rooms' }}</h1>
        <span v-if="!isAdmin && roomLimit > 0" class="nr-mono nr-deck-sub">{{ roomCount }} of {{ roomLimit }} rooms used</span>
        <span v-else class="nr-mono nr-deck-sub">{{ rooms.length }} {{ rooms.length === 1 ? 'room' : 'rooms' }}</span>
      </div>
      <div class="nr-deck-actions">
        <RoomsQuick v-if="canCreate" />
        <button type="button" class="nr-btn is-primary" :disabled="!canCreate" @click="dialog = true">
          <v-icon small>mdi-plus</v-icon>New room
        </button>
      </div>
    </div>

    <section class="nr-telemetry nr-surface" aria-label="Telemetry">
      <div class="nr-stat">
        <span class="nr-label">Running</span>
        <span class="nr-mono nr-stat-value">{{ runningCount }}<span class="nr-stat-of"> / {{ rooms.length }}</span></span>
      </div>
      <div class="nr-stat">
        <span class="nr-label">Watching now</span>
        <span class="nr-mono nr-stat-value">{{ watchingCount }}</span>
      </div>
      <div class="nr-stat">
        <span class="nr-label">{{ configConnections ? 'Ports in use' : 'Connections' }}</span>
        <span v-if="configConnections" class="nr-mono nr-stat-value">{{ usedConnections }}<span class="nr-stat-of"> / {{ configConnections }}</span></span>
        <span v-else class="nr-mono nr-stat-value">mux</span>
        <div v-if="configConnections" class="nr-meter" :class="{ 'is-full': usedConnections >= configConnections }">
          <i :style="{ width: Math.min(100, (usedConnections / configConnections) * 100) + '%' }" />
        </div>
      </div>
      <div class="nr-stat">
        <span class="nr-label">Images</span>
        <span class="nr-mono nr-stat-value">{{ imageCount }}<span v-if="outdatedCount" class="nr-stat-warn"> · {{ outdatedCount }} {{ outdatedCount === 1 ? 'update' : 'updates' }}</span></span>
      </div>
    </section>

    <section class="nr-deck-rooms" aria-label="Rooms">
      <div class="nr-deck-toolbar">
        <div class="nr-seg" role="group" aria-label="Filter">
          <button type="button" :class="{ 'is-on': filter === 'all' }" @click="filter = 'all'">All <span class="nr-mono">{{ rooms.length }}</span></button>
          <button type="button" :class="{ 'is-on': filter === 'running' }" @click="filter = 'running'">Running <span class="nr-mono">{{ runningCount }}</span></button>
          <button type="button" :class="{ 'is-on': filter === 'stopped' }" @click="filter = 'stopped'">Stopped <span class="nr-mono">{{ rooms.length - runningCount }}</span></button>
        </div>
        <div class="nr-deck-refresh">
          <span class="nr-label d-none d-sm-inline">Refresh</span>
          <div class="nr-seg" role="group" aria-label="Auto refresh">
            <button
              v-for="item in refreshItems"
              :key="item.value"
              type="button"
              :class="{ 'is-on': autoRefresh === item.value }"
              @click="autoRefresh = item.value"
            >
              <span v-if="item.value === 0" class="nr-dot" :class="{ 'is-idle': autoRefresh !== 0 }" />{{ item.text }}
            </button>
          </div>
          <v-tooltip bottom open-delay="300">
            <template v-slot:activator="{ on, attrs }">
              <button v-bind="attrs" v-on="on" type="button" class="nr-btn is-icon" aria-label="Refresh now" @click="LoadRooms">
                <v-icon small :class="{ 'nr-spin': loading }">mdi-refresh</v-icon>
              </button>
            </template>
            <span>Refresh now</span>
          </v-tooltip>
        </div>
      </div>

      <RoomsList :loading="loading" :filter="filter" :tick="tick" @create="dialog = true" @refresh="LoadRooms" />

      <div class="nr-deck-foot">
        <span class="nr-mono">{{ updatedLabel }}</span>
        <Pull v-if="canPull" />
      </div>
    </section>

    <v-dialog
      v-model="dialog"
      persistent
      content-class="nr-drawer-dialog"
      transition="scroll-x-reverse-transition"
      :fullscreen="$vuetify.breakpoint.xsOnly"
    >
      <RoomsCreate v-if="dialog" @finished="onCreated" />
    </v-dialog>
  </div>
</template>

<style lang="scss" scoped>
.nr-deck {
  padding-top: 48px;
  padding-bottom: 64px;
  display: flex;
  flex-direction: column;
  gap: 32px;
}
.nr-deck-head {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  justify-content: space-between;
  gap: 24px;
}
.nr-deck-title { display: flex; flex-direction: column; gap: 8px; }
.nr-deck-sub { font-size: 12.5px; color: var(--text-tertiary); }
.nr-deck-actions { display: flex; flex-wrap: wrap; gap: 8px; }

.nr-telemetry {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
}
.nr-stat {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 16px 20px;
  min-width: 0;
  & + & { border-left: 1px solid var(--border-subtle); }
}
.nr-stat-value { font-size: 24px; font-weight: 500; letter-spacing: -0.02em; line-height: 1.1; white-space: nowrap; }
.nr-stat-of { color: var(--text-tertiary); }
.nr-stat-warn { font-size: 12px; letter-spacing: 0; color: var(--status-provisioning-ink); }
.nr-meter {
  height: 2px;
  border-radius: 1px;
  background: var(--bg-surface-3);
  overflow: hidden;
  i { display: block; height: 100%; background: var(--text-secondary); transition: width .4s var(--ease); }
  &.is-full i { background: var(--status-danger); }
}

.nr-deck-rooms { display: flex; flex-direction: column; gap: 12px; }
.nr-deck-toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
}
.nr-deck-refresh { display: flex; align-items: center; gap: 8px; }
.nr-dot.is-idle { background: var(--text-tertiary); box-shadow: none; }
.nr-spin { animation: nr-spin .9s linear infinite; }
.nr-deck-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 0 4px;
  min-height: 36px;
  > .nr-mono { font-size: 11.5px; color: var(--text-tertiary); }
}

@media (max-width: 760px) {
  .nr-telemetry { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .nr-stat:nth-child(3) { border-left: 0; }
  .nr-stat:nth-child(n + 3) { border-top: 1px solid var(--border-subtle); }
}
</style>

<style lang="scss">
// the create form slides in from the right as a full height drawer
.v-dialog__content:has(> .nr-drawer-dialog) { justify-content: flex-end; }
.v-dialog.nr-drawer-dialog {
  margin: 0;
  width: min(560px, 100%);
  max-width: 100%;
  height: 100%;
  max-height: 100%;
  border-radius: 0 !important;
  > .v-card {
    height: 100%;
    border-radius: 0 !important;
    border-top: 0 !important;
    border-right: 0 !important;
    border-bottom: 0 !important;
  }
}
</style>

<script lang="ts">
import { Component, Vue, Watch } from 'vue-property-decorator'
import { AxiosError } from 'axios'
import RoomsList from '@/components/RoomsList.vue'
import RoomsQuick from '@/components/RoomsQuick.vue'
import RoomsCreate from '@/components/RoomsCreate.vue'
import Pull from '@/components/Pull.vue'
import { RoomEntry } from '@/api/index'

@Component({
  components: {
    RoomsList,
    RoomsQuick,
    RoomsCreate,
    Pull,
  }
})
export default class Home extends Vue {
  public loading = false
  public dialog = false
  public filter: 'all' | 'running' | 'stopped' = 'all'
  public updatedAt = 0
  public now = Date.now()
  public tick = Math.floor(Date.now() / 30000)
  private clock = 0

  public interval!: number
  public autoRefresh = 0
  public autoRefreshItems = [
    { text: 'Off', value: -1 },
    { text: 'Live', value: 0 },
    { text: '5s', value: 5 },
    { text: '10s', value: 10 },
    { text: '30s', value: 30 },
    { text: '60s', value: 60 },
  ]

  get refreshItems() {
    return [
      { text: 'Live', value: 0 },
      { text: '5s', value: 5 },
      { text: '30s', value: 30 },
      { text: 'Off', value: -1 },
    ]
  }

  get rooms(): RoomEntry[] {
    return this.$store.state.rooms
  }

  get runningCount(): number {
    return this.rooms.filter(r => r.running).length
  }

  // viewers come from the community directory, which knows who is inside
  get watchingCount(): number {
    const names = new Set(this.rooms.map(r => r.name))
    // eslint-disable-next-line
    return (this.$store.state.app.directory || []).reduce((sum: number, r: any) => sum + (names.has(r.name) ? (r.viewers || 0) : 0), 0)
  }

  get imageCount(): number {
    return new Set(this.rooms.map(r => r.neko_image)).size
  }

  get outdatedCount(): number {
    return this.rooms.filter(r => r.is_outdated).length
  }

  get updatedLabel(): string {
    if (!this.updatedAt) return 'Loading…'
    const s = Math.max(0, Math.round((this.now - this.updatedAt) / 1000))
    const ago = s < 60 ? s + 's ago' : Math.round(s / 60) + 'm ago'
    const mode = this.autoRefresh === 0 ? 'live' : this.autoRefresh > 0 ? 'every ' + this.autoRefresh + 's' : 'manual'
    return 'Updated ' + ago + ' · ' + mode
  }

  @Watch('$route.query.create')
  onCreateQuery(v: string) {
    if (v && this.canCreate) this.dialog = true
  }

  get isAdmin(): boolean {
    return this.$store.getters.isAdmin
  }

  get canCreate(): boolean {
    return this.$store.getters.canCreateRooms
  }

  get canPull(): boolean {
    const p = this.$store.getters.policy
    return this.isAdmin || (p && p.users_can_pull_images)
  }

  get roomLimit(): number {
    const st = this.$store.state.app.status
    return st ? st.room_limit : 0
  }

  get roomCount(): number {
    const st = this.$store.state.app.status
    return st ? st.room_count : 0
  }

  onCreated() {
    this.dialog = false
    this.$store.dispatch('APP_STATUS')
    if (this.$route.query.create) {
      this.$router.replace({ query: {} }).catch(() => { /* ignore */ })
    }
  }

  get configConnections() {
    return this.$store.state.roomsConfig.connections
  }

  get usedConnections() {
    // eslint-disable-next-line
    return this.$store.state.rooms.reduce((sum: number, { max_connections }: RoomEntry) => sum + (max_connections || 1/* 0 when using mux */), 0)
  }

  async LoadRooms() {
    // do not load config if document is hidden
    if (document.hidden) return

    this.loading = true

    try {
      await this.$store.dispatch('ROOMS_LOAD')
      this.updatedAt = Date.now()
      this.tick = Math.floor(Date.now() / 30000)
      this.$store.dispatch('APP_DIRECTORY').catch(() => { /* viewers are optional */ })
    } catch(e) {
      const response = (e as AxiosError).response
      if (response) {
        console.error('Server error', response.data)
      } else {
        console.error('Network error', String(e))
      }
    } finally {
      this.loading = false
    }
  }

  @Watch('autoRefresh', { immediate: true })
  onAutoRefresh() {
    if (this.interval) {
      window.clearInterval(this.interval)
    }
  
    if (this.autoRefresh > 0) {
      this.sseClose()
      this.interval = window.setInterval(this.LoadRooms, this.autoRefresh * 1000)
    } else if (this.autoRefresh == 0) {
      this.sseStart()
    }
  }

  async mounted() {
    try {
      await this.$store.dispatch('ROOMS_CONFIG')
    } catch(e) {
      const response = (e as AxiosError).response
      if (response) {
        console.error('Server error', response.data)
      } else {
        console.error('Network error', String(e))
      }
    }

    await this.LoadRooms()

    if (this.$route.query.create && this.canCreate) {
      this.dialog = true
    }
  }

  created() {
    this.clock = window.setInterval(() => { this.now = Date.now() }, 1000)
  }

  beforeDestroy() {
    this.autoRefresh = -1
    window.clearInterval(this.clock)
  }

  sse: EventSource | null = null
  sseIsOpen = false
  sseReconnBackoff = 1000
  sseReconnRetries = 0
  
  async sseStart(reconnect = false) {
    if (!reconnect) {
      this.sseIsOpen = true
    }

    if (this.sse) {
      this.sse.close()
    }

    this.sse = await this.$store.dispatch('EVENTS_SSE') as EventSource
    this.sseReconnRetries++

    this.sse.addEventListener('rooms', (e) => {
      console.log('SSE rooms', e)
      this.LoadRooms()
    })
  
    this.sse.addEventListener('error', (e: Event) => {
      console.log('SSE error', e)

      // if is closed, don't try to reconnect
      if (!this.sseIsOpen) return

      const wait = this.sseReconnBackoff + 100 * this.sseReconnRetries

      // try to reconnect
      setTimeout(() => {
        this.sseStart(true)
      }, wait)

      console.log('SSE error, trying to reconnect in ' + wait + 'ms')
    })

    this.sse.addEventListener('message', (e: Event) => {
      console.log('SSE message', e)
    })

    this.sse.addEventListener('open', (e: Event) => {
      console.log('SSE opened', e)
    })
  }

  sseClose() {
    if (this.sse) {
      this.sse.close()
    }
    this.sse = null
    this.sseIsOpen = false
  }
}
</script>
