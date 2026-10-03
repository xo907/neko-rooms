<template>
  <v-container class="py-8">
    <div class="d-flex align-center flex-wrap mb-6" style="gap: 12px">
      <div>
        <h1 class="nr-page-title nr-heading text-h4">{{ isAdmin ? 'All rooms' : 'My rooms' }}</h1>
        <div class="text--secondary" v-if="!isAdmin && roomLimit > 0">{{ roomCount }} of {{ roomLimit }} rooms used</div>
      </div>
    </div>

    <!-- show off-site -->
    <div v-if="isAdmin && configConnections" class="text-center mb-6">
      <p>Ports used:</p>
      <v-progress-circular
        :rotate="270"
        :size="100"
        :width="15"
        :value="(usedConnections / configConnections) * 100"
        :color="usedConnections == configConnections ? 'red' : 'blue'"
      >
        {{ usedConnections }} / {{ configConnections }}
      </v-progress-circular>
    </div>

    <v-row>
      <v-col class="mb-3">
        <v-select
          v-model="autoRefresh"
          :items="autoRefreshItems"
          dense
          outlined
          hide-details
          label="Auto refresh"
          class="mr-2"
          style="width:100px;display:inline-block"
        ></v-select>
        <v-tooltip right open-delay="300">
          <template v-slot:activator="{ on, attrs }">
            <v-btn v-bind="attrs" v-on="on" @click="LoadRooms" color="green" icon><v-icon>mdi-reload</v-icon></v-btn>
          </template>
          <span>Manual refresh</span>
        </v-tooltip>
      </v-col>
      <v-col class="text-right">
        <RoomsQuick v-if="canCreate" class="mr-3" />
        <v-dialog v-model="dialog" persistent max-width="640px">
          <template v-slot:activator="{ on, attrs }">
            <v-btn
              v-bind="attrs"
              v-on="on"
              color="primary"
              depressed
              :disabled="!canCreate"
            >
              <v-icon left>mdi-plus</v-icon> New room
            </v-btn>
          </template>

          <RoomsCreate v-if="dialog" @finished="onCreated" />
        </v-dialog>
      </v-col>
    </v-row>

    <RoomsList :loading="loading" />

    <div class="mt-5 text-center" v-if="canPull">
      <Pull />
    </div>
  </v-container>
</template>

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

  beforeDestroy() {
    this.autoRefresh = -1
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
