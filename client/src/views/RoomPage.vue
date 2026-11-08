<template>
  <div class="nr-room-page">
    <router-link :to="{ name: 'home' }" class="nr-back">
      <v-icon small>mdi-arrow-left</v-icon> All rooms
    </router-link>

    <div v-if="loading && !room" class="text-center py-16">
      <v-progress-circular indeterminate color="primary" size="32" width="2" />
    </div>

    <div v-else-if="error" class="nr-state nr-rise">
      <div class="nr-state-icon" :class="errorStatus === 403 ? 'is-warn' : 'is-danger'">
        <v-icon>{{ errorStatus === 403 ? 'mdi-lock-outline' : 'mdi-door-closed' }}</v-icon>
      </div>
      <h1 class="nr-state-title">{{ errorStatus === 403 ? 'This room is private' : 'Room not found' }}</h1>
      <p class="nr-state-text">
        {{ errorStatus === 403 ? 'Ask the owner for an invite link, or sign in if you are one of their friends.' : 'It may have been closed or renamed.' }}
      </p>
      <v-btn v-if="errorStatus === 403 && !user" color="primary" depressed :to="{ name: 'login', query: { redirect: $route.fullPath } }">Sign in</v-btn>
    </div>

    <div v-else-if="room" class="nr-room-layout nr-rise">
      <main>
        <div ref="preview" class="nr-preview" :style="previewStyle">
          <img v-if="showThumb" :src="thumbUrl" @error="thumbFailed = true" alt="">
          <div v-else class="nr-preview-fallback">
            <v-icon size="48">mdi-play-circle-outline</v-icon>
          </div>
          <div class="nr-preview-overlay" v-if="room.can_join">
            <v-btn large color="primary" depressed :disabled="!canJoinNow" @click="join" class="px-8">
              <v-icon v-if="room.sleeping" left>mdi-weather-night</v-icon>{{ joinLabel }}
            </v-btn>
            <div v-if="room.ready" class="nr-preview-note">
              <span class="nr-dot" />{{ room.viewers }} {{ room.viewers === 1 ? 'person' : 'people' }} watching
            </div>
            <div v-else-if="room.sleeping" class="nr-preview-note">Asleep while empty, joining wakes it up</div>
            <div v-else-if="partyAt" class="nr-preview-note">{{ partyDate }}</div>
            <v-btn v-if="isAndroid" small text dark :href="appHref" class="nr-app-link">
              <v-icon small left>mdi-cellphone</v-icon>Open in the WatchParty app
            </v-btn>
          </div>
        </div>

        <div class="nr-room-head">
          <div class="flex-grow-1" style="min-width: 0">
            <h1 class="nr-room-name">{{ room.title }}</h1>
            <div class="nr-room-tags">
              <span v-if="room.ready" class="nr-pill-chip is-live"><span class="nr-dot" />Live</span>
              <span v-else-if="room.paused" class="nr-pill-chip is-warn">Paused</span>
              <span v-else-if="room.sleeping" class="nr-pill-chip"><v-icon>mdi-weather-night</v-icon>Sleeping</span>
              <span v-else class="nr-pill-chip">Offline</span>
              <span v-if="partyAt && !room.ready" class="nr-pill-chip is-accent"><v-icon>mdi-calendar-star</v-icon>Watch party {{ partyRelative }}</span>
              <span class="nr-pill-chip"><v-icon>{{ visibilityIcon }}</v-icon>{{ room.visibility }}</span>
              <span v-if="room.category" class="nr-pill-chip">{{ room.category }}</span>
              <span v-if="room.featured" class="nr-pill-chip"><v-icon>mdi-star-outline</v-icon>Featured</span>
              <span v-if="room.owner_control" class="nr-pill-chip" title="Only the host and admins control the browser"><v-icon>mdi-lock-outline</v-icon>Host controls</span>
              <span v-if="room.hidden" class="nr-pill-chip is-danger">Hidden by moderator</span>
            </div>
            <div v-if="room.now_playing" class="nr-now-playing">
              <span class="nr-eq"><i /><i /><i /></span>
              <span class="text-truncate"><span class="nr-now-label">Now playing</span>{{ room.now_playing }}</span>
            </div>
            <div class="nr-room-host" v-if="room.owner">
              <UserAvatar :user="room.owner" :size="20" />
              <span>Hosted by <strong>{{ room.owner.display_name || room.owner.username }}</strong></span>
            </div>
          </div>
          <v-btn outlined small @click="share">
            <v-icon left small>{{ copied ? 'mdi-check' : 'mdi-link-variant' }}</v-icon>{{ copied ? 'Copied' : 'Share' }}
          </v-btn>
        </div>

        <p v-if="room.description" class="nr-room-desc">{{ room.description }}</p>
      </main>

      <aside class="nr-room-side">
        <section v-if="partyAt" class="nr-panel nr-party">
          <div class="nr-panel-head">
            <span><v-icon small class="mr-1">mdi-calendar-star</v-icon>Watch party</span>
            <span class="nr-panel-count">{{ room.rsvps }} going</span>
          </div>
          <div class="nr-party-when">{{ partyDate }}</div>
          <div class="nr-party-rel">{{ room.ready ? 'Happening now' : 'Starts ' + partyRelative }}</div>
          <div class="d-flex flex-wrap mt-4" style="gap: 8px">
            <v-btn v-if="user" small depressed :color="room.going ? 'primary' : undefined" :outlined="!room.going" :loading="rsvping" @click="rsvp(!room.going)">
              <v-icon left small>{{ room.going ? 'mdi-check' : 'mdi-hand-wave-outline' }}</v-icon>{{ room.going ? 'Going' : 'I am going' }}
            </v-btn>
            <v-btn v-else small outlined :to="{ name: 'login', query: { redirect: $route.fullPath } }">Sign in to RSVP</v-btn>
            <v-btn small text :href="calendarUrl">
              <v-icon left small>mdi-calendar-plus</v-icon>Add to calendar
            </v-btn>
          </div>
          <div v-if="room.guests.length" class="nr-party-guests">
            <UserAvatar v-for="g in room.guests.slice(0, 8)" :key="g" :name="g" :size="24" :title="g" />
            <span v-if="room.guests.length > 8" class="nr-panel-count">+{{ room.guests.length - 8 }}</span>
          </div>
        </section>

        <section class="nr-panel">
          <div class="nr-panel-head">
            <span>In this room</span>
            <span class="nr-panel-count">{{ room.viewers }}<span v-if="room.max_connections">/{{ room.max_connections }}</span></span>
          </div>
          <ul v-if="room.members.length" class="nr-members">
            <li v-for="m in room.members" :key="m">
              <UserAvatar :name="m" :src="room.avatars && room.avatars[m]" :size="26" />
              <span class="text-truncate">{{ m }}</span>
              <span v-if="room.friends_inside.includes(m)" class="nr-pill-chip is-live ml-auto">friend</span>
            </li>
          </ul>
          <p v-else class="nr-panel-empty">
            {{ room.ready ? 'Nobody is here yet. Be the first!' : room.sleeping ? 'The room is asleep. Join to wake it up.' : 'The room is not running right now.' }}
          </p>
        </section>

        <!-- owner / admin tools -->
        <section v-if="room.can_manage" class="nr-panel">
          <div class="nr-panel-head"><span>Manage</span></div>
          <div class="d-flex flex-wrap mb-5" style="gap: 6px">
            <RoomActionBtn action="start" :roomId="room.id" :disabled="room.running" @done="load" />
            <RoomActionBtn action="stop" :roomId="room.id" :disabled="!room.running && !room.paused" @done="load" />
            <RoomActionBtn action="pause" :roomId="room.id" :disabled="!room.running" @done="load" />
            <RoomActionBtn action="restart" :roomId="room.id" :disabled="!room.running" @done="load" />
            <RoomActionBtn action="upgrade" :roomId="room.id" :roomName="room.name" @done="load" />
            <v-spacer />
            <RoomActionBtn action="remove" :roomId="room.id" @done="$router.push({ name: 'home' })" />
          </div>

          <div class="nr-label">Invite link</div>
          <div v-if="room.invite_code" class="d-flex align-center">
            <v-text-field :value="invite" readonly dense outlined hide-details class="mr-2" @focus="$event.target.select()" />
            <v-btn icon small @click="copyInvite" title="Copy"><v-icon small>{{ inviteCopied ? 'mdi-check' : 'mdi-content-copy' }}</v-icon></v-btn>
          </div>
          <div v-else class="nr-panel-empty pa-0">Invite link is disabled.</div>
          <div class="mt-2">
            <v-btn x-small text color="primary" @click="regenerate(false)">{{ room.invite_code ? 'New link' : 'Enable link' }}</v-btn>
            <v-btn v-if="room.invite_code" x-small text color="error" @click="regenerate(true)">Disable</v-btn>
          </div>
        </section>

        <section v-if="room.can_manage" class="nr-panel">
          <div class="nr-panel-head"><span>Details & visibility</span></div>
          <RoomMetaEditor :roomId="room.id" :name="room.name" :meta="meta" :moderation="isAdmin" @saved="onSaved" />
        </section>
      </aside>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.nr-room-page {
  max-width: var(--page-max);
  margin: 0 auto;
  padding: 24px var(--page-pad) 64px;
}
.nr-back {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 24px;
  color: var(--text-tertiary) !important;
  font: 400 12px var(--font-mono);
  text-decoration: none;
  transition: color var(--dur) var(--ease);
  .v-icon { color: inherit !important; font-size: 14px !important; }
  &:hover { color: var(--text-primary) !important; }
}
.nr-room-layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 32px;
  align-items: start;
}
@media (min-width: 1024px) {
  .nr-room-layout { grid-template-columns: minmax(0, 1fr) 320px; }
}
.nr-room-layout > main { display: flex; flex-direction: column; gap: 24px; min-width: 0; }

.nr-preview {
  position: relative;
  aspect-ratio: 16 / 9;
  overflow: hidden;
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-subtle);
  background: #000 !important;
  img {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    object-fit: cover;
    opacity: .38;
  }
}
.nr-preview-fallback {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-surface-1);
  .v-icon { color: var(--text-tertiary) !important; }
}
.nr-preview-fallback:has(+ .nr-preview-overlay) .v-icon { display: none; }
.nr-preview-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 16px;
  ::v-deep .v-btn.v-size--large { height: 44px !important; padding: 0 20px !important; font-size: 14.5px !important; }
}
.nr-app-link {
  text-shadow: 0 1px 4px rgba(0, 0, 0, .6);
}

.nr-preview-note {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font: 400 12px var(--font-mono);
  color: var(--text-primary);
}

.nr-room-head {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}
.nr-room-name {
  margin: 0 0 12px;
  font-size: 32px !important;
  font-weight: 600 !important;
  letter-spacing: -0.035em !important;
  line-height: 1.1;
}
.nr-room-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  .nr-pill-chip { text-transform: lowercase; }
}
.nr-room-tags .nr-pill-chip.is-live, .nr-room-tags .nr-pill-chip.is-accent { text-transform: none; }

.nr-now-playing {
  display: flex;
  align-items: baseline;
  gap: 16px;
  margin-top: 24px;
  padding: 16px 0;
  border-top: 1px solid var(--border-subtle);
  border-bottom: 1px solid var(--border-subtle);
  font-size: 14px;
  font-weight: 500;
  min-width: 0;
}
.nr-now-label {
  display: inline-block;
  width: 112px;
  margin-right: 16px;
  font: 500 11px var(--font-mono);
  letter-spacing: .08em;
  text-transform: uppercase;
  color: var(--text-tertiary);
}
.nr-eq { display: none; }

.nr-room-host {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 16px 0;
  border-bottom: 1px solid var(--border-subtle);
  font-size: 14px;
  color: var(--text-secondary);
  strong { color: var(--text-primary); font-weight: 500; }
}
.nr-room-desc {
  margin: 0;
  font-size: 14px;
  line-height: 1.55;
  color: var(--text-secondary);
  white-space: pre-line;
}

.nr-room-side { display: flex; flex-direction: column; gap: 16px; }
.nr-panel {
  padding: 0 16px 16px;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-lg);
  background: var(--bg-surface-1);
}
.nr-panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  height: 44px;
  margin: 0 -16px 12px;
  padding: 0 16px;
  border-bottom: 1px solid var(--border-subtle);
  font: 500 11px var(--font-mono);
  letter-spacing: .08em;
  text-transform: uppercase;
  color: var(--text-tertiary);
  .v-icon { display: none; }
}
.nr-panel-count { font: 400 12px var(--font-mono); letter-spacing: 0; text-transform: none; color: var(--text-secondary); }
.nr-panel-empty { margin: 0; font-size: 13.5px; line-height: 1.55; color: var(--text-secondary); }
.nr-label { display: block; margin-bottom: 8px; }
.nr-members {
  list-style: none;
  margin: -12px -16px -16px;
  padding: 0;
  li {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 16px;
    border-top: 1px solid var(--border-subtle);
    font-size: 13.5px;
    &:first-child { border-top: 0; }
  }
}

.nr-party-when { font-size: 17px; font-weight: 600; letter-spacing: -0.025em; color: var(--text-primary); }
.nr-party-rel { margin-top: 4px; font: 400 12px var(--font-mono); color: var(--text-tertiary); }
.nr-party-guests {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px;
  margin-top: 16px;
}

.nr-state {
  max-width: 440px;
  margin: 64px auto;
  padding: 48px 24px;
  text-align: center;
  border: 1px dashed var(--border-hover);
  border-radius: var(--radius-lg);
}
.nr-state-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  margin-bottom: 16px;
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-hover);
  background: var(--bg-surface-1);
  .v-icon { color: var(--text-secondary) !important; }
  &.is-warn .v-icon { color: var(--status-provisioning) !important; }
  &.is-danger .v-icon { color: var(--status-danger) !important; }
}
.nr-state-title { font-size: 18px !important; font-weight: 600; letter-spacing: -0.025em; margin: 0 0 8px; }
.nr-state-text { color: var(--text-secondary); margin: 0 0 24px; line-height: 1.55; font-size: 14px; }
</style>

<script lang="ts">
import { Vue, Component, Prop, Watch } from 'vue-property-decorator'
import { api, errorMessage, Room, RoomMeta, User } from '@/api-ext'
import { appLink, copyText, inviteLink, isAndroid, joinRoom, relativeTime, roomLink } from '@/utils/join'
import UserAvatar from '@/components/community/UserAvatar.vue'
import RoomMetaEditor from '@/components/community/RoomMetaEditor.vue'
import RoomActionBtn from '@/components/RoomActionBtn.vue'

@Component({
  components: {
    UserAvatar,
    RoomMetaEditor,
    RoomActionBtn,
  }
})
export default class RoomPage extends Vue {
  @Prop({ required: true }) readonly name!: string

  room: Room | null = null
  meta: RoomMeta | null = null
  loading = false
  error = ''
  errorStatus = 0
  thumbFailed = false
  tick = Date.now()
  copied = false
  inviteCopied = false
  rsvping = false

  private timer = 0

  get user(): User | null {
    return this.$store.getters.user
  }

  get isAdmin(): boolean {
    return this.$store.getters.isAdmin
  }

  get inviteCode(): string {
    return String(this.$route.query.invite || '')
  }

  isAndroid = isAndroid

  get appHref(): string {
    return this.room ? appLink(this.room.name, this.inviteCode || this.room.invite_code || undefined) : ''
  }

  get invite(): string {
    return this.room && this.room.invite_code ? inviteLink(this.room.name, this.room.invite_code) : ''
  }

  get showThumb() {
    return this.room && this.room.has_thumbnail && this.room.ready && !this.thumbFailed
  }

  get thumbUrl() {
    return api.thumbnailUrl(this.name, this.tick, this.inviteCode || undefined)
  }

  get previewStyle() {
    return {}
  }


  get partyAt(): Date | null {
    return this.room && this.room.scheduled_at ? new Date(this.room.scheduled_at) : null
  }

  get partyDate(): string {
    return this.partyAt ? this.partyAt.toLocaleString(undefined, { weekday: 'long', month: 'long', day: 'numeric', hour: 'numeric', minute: '2-digit' }) : ''
  }

  get partyRelative(): string {
    return this.partyAt ? relativeTime(this.partyAt, this.tick) : ''
  }

  // a party that is on can be joined, which starts the room
  get partyOn(): boolean {
    return !!this.partyAt && this.partyAt.getTime() - this.tick < 2 * 60000
  }

  get canJoinNow(): boolean {
    return !!this.room && (this.room.running || this.room.sleeping || this.partyOn)
  }

  get joinLabel(): string {
    if (!this.room) return ''
    if (this.room.running) return 'Join room'
    if (this.room.sleeping || this.partyOn) return 'Wake up & join'
    if (this.partyAt) return 'Starts ' + this.partyRelative
    return 'Room is not running'
  }

  get calendarUrl(): string {
    return this.room ? api.calendarUrl(this.room.name, this.inviteCode || undefined) : ''
  }

  async rsvp(going: boolean) {
    if (!this.room) return
    this.rsvping = true
    try {
      this.room = await api.rsvp(this.room.name, going, this.inviteCode || undefined)
    } catch (e) {
      this.$swal({ icon: 'error', title: 'Unable to RSVP', text: errorMessage(e) })
    } finally {
      this.rsvping = false
    }
  }

  get visibilityIcon() {
    if (!this.room) return ''
    return { public: 'mdi-earth', friends: 'mdi-account-multiple', private: 'mdi-lock' }[this.room.visibility]
  }

  @Watch('name')
  async load() {
    if (document.hidden && this.room) return
    this.loading = true
    try {
      this.room = await api.room(this.name, this.inviteCode || undefined)
      this.error = ''
      document.title = this.room.title + ' · ' + document.title.split(' · ').pop()
      if (this.room.can_manage && !this.meta) {
        this.meta = await api.roomMeta(this.room.id)
      }
    } catch (e) {
      // eslint-disable-next-line
      this.errorStatus = (e as any)?.response?.status || 0
      this.error = errorMessage(e)
    } finally {
      this.loading = false
    }
  }

  join() {
    if (!this.room) return
    joinRoom(this, this.room.name, {
      invite: this.inviteCode || undefined,
      room: this.room,
      from: this.$refs.preview as Element,
      image: this.showThumb ? this.thumbUrl : '',
    })
  }

  async share() {
    if (!this.room) return
    this.copied = await copyText(this.inviteCode ? inviteLink(this.room.name, this.inviteCode) : roomLink(this.room.name))
    setTimeout(() => { this.copied = false }, 2000)
  }

  async copyInvite() {
    this.inviteCopied = await copyText(this.invite)
    setTimeout(() => { this.inviteCopied = false }, 2000)
  }

  async regenerate(disable: boolean) {
    if (!this.room) return
    try {
      this.meta = await api.regenerateInvite(this.room.id, disable)
      await this.load()
    } catch (e) {
      this.$swal({ icon: 'error', title: 'Error', text: errorMessage(e) })
    }
  }

  onSaved(meta: RoomMeta) {
    this.meta = meta
    this.load()
  }

  mounted() {
    this.load()
    this.timer = window.setInterval(() => {
      this.tick = Date.now()
      this.thumbFailed = false
      this.load()
    }, 15000)
  }

  beforeDestroy() {
    window.clearInterval(this.timer)
  }
}
</script>
