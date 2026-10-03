<template>
  <v-container class="py-8" style="max-width: 1180px">
    <v-btn text small :to="{ name: 'home' }" class="mb-4 px-1">
      <v-icon left small>mdi-arrow-left</v-icon> All rooms
    </v-btn>

    <div v-if="loading && !room" class="text-center py-16">
      <v-progress-circular indeterminate color="primary" size="48" />
    </div>

    <v-card v-else-if="error" class="pa-10 text-center">
      <v-icon size="64" :color="errorStatus === 403 ? 'warning' : 'error'" class="mb-4">
        {{ errorStatus === 403 ? 'mdi-lock' : 'mdi-door-closed' }}
      </v-icon>
      <div class="text-h5 mb-2">{{ errorStatus === 403 ? 'This room is private' : 'Room not found' }}</div>
      <div class="text--secondary mb-6">
        {{ errorStatus === 403 ? 'Ask the owner for an invite link, or sign in if you are one of their friends.' : 'It may have been closed or renamed.' }}
      </div>
      <v-btn v-if="errorStatus === 403 && !user" color="primary" depressed :to="{ name: 'login', query: { redirect: $route.fullPath } }">Sign in</v-btn>
    </v-card>

    <v-row v-else-if="room">
      <v-col cols="12" md="8">
        <v-card class="overflow-hidden">
          <div class="nr-preview" :style="previewStyle">
            <img v-if="showThumb" :src="thumbUrl" @error="thumbFailed = true" alt="">
            <div v-else class="nr-preview-fallback">
              <v-icon size="96" color="white" style="opacity: .8">mdi-play-circle-outline</v-icon>
            </div>
            <div class="nr-preview-overlay d-flex flex-column align-center justify-center" v-if="room.can_join">
              <v-btn x-large color="primary" depressed :disabled="!room.running" @click="join" class="px-8">
                <v-icon left>mdi-login-variant</v-icon>{{ room.running ? 'Join room' : 'Room is not running' }}
              </v-btn>
              <div v-if="room.ready" class="white--text mt-3 font-weight-medium">
                {{ room.viewers }} {{ room.viewers === 1 ? 'person' : 'people' }} watching
              </div>
            </div>
          </div>

          <v-card-text>
            <div class="d-flex align-start">
              <div class="flex-grow-1">
                <div class="d-flex align-center flex-wrap" style="gap: 6px">
                  <h1 class="text-h5 font-weight-bold nr-heading mr-2">{{ room.title }}</h1>
                  <v-chip v-if="room.ready" x-small color="error" label class="font-weight-bold">LIVE</v-chip>
                  <v-chip v-else-if="room.paused" x-small color="warning" label>PAUSED</v-chip>
                  <v-chip v-else x-small label>OFFLINE</v-chip>
                  <v-chip x-small label outlined>
                    <v-icon x-small left>{{ visibilityIcon }}</v-icon>{{ room.visibility }}
                  </v-chip>
                  <v-chip v-if="room.category" x-small label outlined>{{ room.category }}</v-chip>
                  <v-chip v-if="room.featured" x-small label color="amber" text-color="black">
                    <v-icon x-small left>mdi-star</v-icon>featured
                  </v-chip>
                  <v-chip v-if="room.hidden" x-small label color="error" outlined>hidden by moderator</v-chip>
                </div>
                <div class="d-flex align-center mt-2 text--secondary" v-if="room.owner">
                  <UserAvatar :user="room.owner" :size="22" class="mr-2" />
                  hosted by <strong class="ml-1">{{ room.owner.display_name || room.owner.username }}</strong>
                </div>
              </div>
              <v-btn icon @click="share" title="Copy link">
                <v-icon>{{ copied ? 'mdi-check' : 'mdi-share-variant' }}</v-icon>
              </v-btn>
            </div>

            <p v-if="room.description" class="body-1 mt-4 mb-0" style="white-space: pre-line">{{ room.description }}</p>
          </v-card-text>
        </v-card>
      </v-col>

      <v-col cols="12" md="4">
        <v-card class="mb-4">
          <v-card-title class="subtitle-1 font-weight-bold">
            <v-icon left small>mdi-account-group</v-icon> In this room
            <v-spacer />
            <span class="text--secondary">{{ room.viewers }}<span v-if="room.max_connections">/{{ room.max_connections }}</span></span>
          </v-card-title>
          <v-list dense v-if="room.members.length">
            <v-list-item v-for="m in room.members" :key="m">
              <v-list-item-avatar size="28"><UserAvatar :name="m" :size="28" /></v-list-item-avatar>
              <v-list-item-title>{{ m }}</v-list-item-title>
              <v-list-item-action v-if="room.friends_inside.includes(m)">
                <v-chip x-small color="accent">friend</v-chip>
              </v-list-item-action>
            </v-list-item>
          </v-list>
          <v-card-text v-else class="text--secondary">
            {{ room.ready ? 'Nobody is here yet. Be the first!' : 'The room is not running right now.' }}
          </v-card-text>
        </v-card>

        <!-- owner / admin tools -->
        <v-card v-if="room.can_manage" class="mb-4">
          <v-card-title class="subtitle-1 font-weight-bold">
            <v-icon left small>mdi-cog-outline</v-icon> Manage room
          </v-card-title>
          <v-card-text>
            <div class="d-flex flex-wrap mb-4" style="gap: 6px">
              <RoomActionBtn action="start" :roomId="room.id" :disabled="room.running" @done="load" />
              <RoomActionBtn action="stop" :roomId="room.id" :disabled="!room.running && !room.paused" @done="load" />
              <RoomActionBtn action="pause" :roomId="room.id" :disabled="!room.running" @done="load" />
              <RoomActionBtn action="restart" :roomId="room.id" :disabled="!room.running" @done="load" />
              <v-spacer />
              <RoomActionBtn action="remove" :roomId="room.id" @done="$router.push({ name: 'home' })" />
            </div>

            <div class="subtitle-2 mb-1">Invite link</div>
            <div v-if="room.invite_code" class="d-flex align-center">
              <v-text-field :value="invite" readonly dense outlined hide-details class="mr-2" @focus="$event.target.select()" />
              <v-btn icon small @click="copyInvite" title="Copy"><v-icon small>{{ inviteCopied ? 'mdi-check' : 'mdi-content-copy' }}</v-icon></v-btn>
            </div>
            <div v-else class="caption text--secondary">Invite link is disabled.</div>
            <div class="mt-2">
              <v-btn x-small text color="primary" @click="regenerate(false)">{{ room.invite_code ? 'New link' : 'Enable link' }}</v-btn>
              <v-btn v-if="room.invite_code" x-small text color="error" @click="regenerate(true)">Disable</v-btn>
            </div>
          </v-card-text>
        </v-card>

        <v-card v-if="room.can_manage">
          <v-card-title class="subtitle-1 font-weight-bold">
            <v-icon left small>mdi-pencil-outline</v-icon> Details & visibility
          </v-card-title>
          <v-card-text>
            <RoomMetaEditor :roomId="room.id" :name="room.name" :meta="meta" :moderation="isAdmin" @saved="onSaved" />
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>
  </v-container>
</template>

<style lang="scss" scoped>
.nr-preview {
  position: relative;
  aspect-ratio: 16 / 9;
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
}
.nr-preview-overlay {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, .35);
  opacity: 1;
}
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
    return { background: 'linear-gradient(135deg, var(--nr-primary), var(--nr-accent))' }
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
    if (this.room) joinRoom(this, this.room.name, this.inviteCode || undefined)
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
