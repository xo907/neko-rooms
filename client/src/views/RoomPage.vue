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
            <v-btn large color="primary" depressed :disabled="!room.running" @click="join" class="px-8">
              {{ room.running ? 'Join room' : 'Room is not running' }}
            </v-btn>
            <div v-if="room.ready" class="nr-preview-note">
              <span class="nr-dot" />{{ room.viewers }} {{ room.viewers === 1 ? 'person' : 'people' }} watching
            </div>
          </div>
        </div>

        <div class="nr-room-head">
          <div class="flex-grow-1" style="min-width: 0">
            <h1 class="nr-room-name">{{ room.title }}</h1>
            <div class="nr-room-tags">
              <span v-if="room.ready" class="nr-pill-chip is-live"><span class="nr-dot" />Live</span>
              <span v-else-if="room.paused" class="nr-pill-chip is-warn">Paused</span>
              <span v-else class="nr-pill-chip">Offline</span>
              <span class="nr-pill-chip"><v-icon>{{ visibilityIcon }}</v-icon>{{ room.visibility }}</span>
              <span v-if="room.category" class="nr-pill-chip">{{ room.category }}</span>
              <span v-if="room.featured" class="nr-pill-chip"><v-icon>mdi-star-outline</v-icon>Featured</span>
              <span v-if="room.hidden" class="nr-pill-chip is-danger">Hidden by moderator</span>
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
            {{ room.ready ? 'Nobody is here yet. Be the first!' : 'The room is not running right now.' }}
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
  max-width: 1240px;
  margin: 0 auto;
  padding: 28px 16px 64px;
}
.nr-back {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 20px;
  color: var(--nr-muted) !important;
  font-size: 13px;
  font-weight: 500;
  text-decoration: none;
  transition: color .18s;
  .v-icon { color: inherit !important; }
  &:hover { color: var(--nr-text) !important; }
}
.nr-room-layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 32px;
}
@media (min-width: 1024px) {
  .nr-room-layout { grid-template-columns: minmax(0, 1fr) 340px; }
}
.nr-preview {
  position: relative;
  aspect-ratio: 16 / 9;
  overflow: hidden;
  border-radius: calc(var(--nr-radius) * 1.4);
  border: 1px solid var(--nr-line);
  box-shadow: 0 40px 100px -40px rgba(0, 0, 0, .8);
  img {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
}
.nr-preview-fallback {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  .v-icon { color: color-mix(in srgb, var(--nr-text) 30%, transparent) !important; }
}
.nr-preview-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 14px;
  background: radial-gradient(60% 60% at 50% 50%, rgba(0, 0, 0, .65), rgba(0, 0, 0, .3));
  backdrop-filter: blur(3px);
}
.nr-preview-note {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  padding: 4px 12px;
  border-radius: 999px;
  background: rgba(8, 8, 10, .55);
  backdrop-filter: blur(10px);
  color: rgba(255, 255, 255, .9);
  font-size: 12.5px;
  font-weight: 500;
}
.nr-room-head {
  display: flex;
  align-items: flex-start;
  gap: 16px;
  margin-top: 24px;
}
.nr-room-name {
  font-size: clamp(1.5rem, 3vw, 2rem);
  font-weight: 600;
  line-height: 1.15;
  margin: 0 0 12px;
}
.nr-room-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  .nr-pill-chip { text-transform: capitalize; }
}
.nr-room-host {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 14px;
  font-size: 13px;
  color: var(--nr-muted);
  strong { color: var(--nr-text); font-weight: 600; }
}
.nr-room-desc {
  margin: 20px 0 0;
  padding-top: 20px;
  border-top: 1px solid var(--nr-line);
  color: var(--nr-muted);
  font-size: 15px;
  line-height: 1.7;
  white-space: pre-line;
}
.nr-room-side {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.nr-panel {
  padding: 18px;
  border-radius: calc(var(--nr-radius) * 1.3);
  border: 1px solid var(--nr-line);
  background: var(--nr-glass);
  backdrop-filter: blur(14px);
}
.nr-panel-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 14px;
  font-size: 13px;
  font-weight: 600;
}
.nr-panel-count { color: var(--nr-subtle); font-weight: 500; }
.nr-panel-empty {
  margin: 0;
  font-size: 13px;
  color: var(--nr-muted);
}
.nr-label {
  margin-bottom: 8px;
  font-size: 12px;
  font-weight: 600;
  color: var(--nr-muted);
}
.nr-members {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
  li {
    display: flex;
    align-items: center;
    gap: 10px;
    min-width: 0;
    font-size: 13.5px;
  }
}
.nr-state {
  max-width: 440px;
  margin: 48px auto;
  text-align: center;
}
.nr-state-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 52px;
  height: 52px;
  margin-bottom: 18px;
  border-radius: 50%;
  border: 1px solid var(--nr-line);
  background: var(--nr-hover);
  &.is-warn .v-icon { color: #fbbf24 !important; }
  &.is-danger .v-icon { color: #f87171 !important; }
}
.nr-state-title { font-size: 1.5rem; font-weight: 600; margin: 0 0 8px; }
.nr-state-text { color: var(--nr-muted); margin: 0 0 24px; line-height: 1.6; }
</style>

<script lang="ts">
import { Vue, Component, Prop, Watch } from 'vue-property-decorator'
import { api, errorMessage, Room, RoomMeta, User } from '@/api-ext'
import { copyText, inviteLink, joinRoom, roomLink } from '@/utils/join'
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
    return { background: 'radial-gradient(120% 90% at 0% 0%, color-mix(in srgb, var(--nr-primary) 18%, transparent), transparent 60%), linear-gradient(135deg, var(--nr-surface), var(--nr-bg))' }
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
