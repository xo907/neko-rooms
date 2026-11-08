<template>
  <article
    class="nr-room-card"
    :class="{ 'nr-room-card--offline': !room.ready, 'nr-room-card--compact': compact }"
    tabindex="0"
    role="link"
    :aria-label="room.title"
    @click="open"
    @keydown.enter="open"
  >
    <div ref="thumb" class="nr-thumb">
      <img v-if="showThumb" :src="thumbUrl" @error="thumbFailed = true" alt="" class="nr-thumb-img">
      <div v-else class="nr-thumb-fallback">
        <span v-if="room.sleeping" class="nr-label">Asleep · wakes on join</span>
        <v-icon v-else size="28">{{ categoryIcon }}</v-icon>
      </div>

      <div class="nr-thumb-top">
        <span v-if="room.ready" class="nr-status is-online is-glass">Live</span>
        <span v-else-if="room.paused" class="nr-status is-warn is-glass">Paused</span>
        <span v-else-if="partyAt" class="nr-status is-plain is-glass nr-party">{{ partyLabel }}</span>
        <span v-else-if="room.sleeping" class="nr-status is-glass">Sleeping</span>
        <span v-else class="nr-status is-stopped is-glass">Offline</span>
        <span v-if="room.ready" class="nr-status is-plain is-glass nr-viewers">{{ room.viewers }} watching</span>
        <span class="flex-grow-1" />
        <span v-if="room.featured" class="nr-glyph" title="Featured"><v-icon x-small>mdi-star</v-icon></span>
        <v-tooltip bottom>
          <template v-slot:activator="{ on, attrs }">
            <span v-bind="attrs" v-on="on" class="nr-glyph"><v-icon x-small>{{ visibilityIcon }}</v-icon></span>
          </template>
          <span>{{ visibilityLabel }}</span>
        </v-tooltip>
      </div>

      <div v-if="room.friends_inside.length" class="nr-thumb-bottom">
        <div class="nr-stack">
          <UserAvatar v-for="n in room.friends_inside.slice(0, 3)" :key="'f' + n" :name="n" :src="avatarOf(n)" :size="18" ring />
        </div>
        <span class="text-truncate">
          {{ room.friends_inside.slice(0, 2).join(', ') }}<span v-if="room.friends_inside.length > 2"> +{{ room.friends_inside.length - 2 }}</span>
          {{ room.friends_inside.length === 1 ? 'is' : 'are' }} here
        </span>
      </div>
    </div>

    <div class="nr-body">
      <div class="nr-room-title text-truncate">{{ room.title }}</div>
      <div v-if="room.now_playing" class="nr-room-playing nr-mono text-truncate" :title="room.now_playing">
        <v-icon x-small>mdi-play</v-icon>{{ room.now_playing }}
      </div>
      <div class="nr-room-meta">
        <template v-if="room.owner">
          <UserAvatar :user="room.owner" :size="16" class="mr-2" />
          <span class="text-truncate">{{ room.owner.display_name || room.owner.username }}</span>
        </template>
        <span v-else class="text-truncate nr-mono">/{{ room.name }}</span>
        <template v-if="room.category">
          <span class="nr-sep">·</span>
          <span class="text-truncate">{{ room.category }}</span>
        </template>
      </div>
      <div v-if="!compact && room.description" class="nr-room-desc">{{ room.description }}</div>

      <div class="nr-room-foot">
        <div v-if="room.members.length" class="nr-stack">
          <UserAvatar v-for="m in room.members.slice(0, 5)" :key="'m' + m" :name="m" :src="avatarOf(m)" :size="22" ring tooltip />
          <span v-if="room.members.length > 5" class="nr-more nr-mono">+{{ room.members.length - 5 }}</span>
        </div>
        <span v-else class="nr-empty-note nr-mono">{{ emptyNote }}</span>
        <span class="flex-grow-1" />
        <router-link v-if="room.can_manage" :to="{ name: 'room', params: { name: room.name } }" class="nr-btn is-icon is-sm" title="Manage room" aria-label="Manage room" @click.native.stop>
          <v-icon small>mdi-tune-variant</v-icon>
        </router-link>
        <button
          v-if="room.can_join"
          type="button"
          class="nr-btn is-sm"
          :class="room.ready ? 'is-primary' : 'is-secondary'"
          :disabled="!room.running && !room.sleeping"
          @click.stop="$emit('join', room, { from: $refs.thumb, image: showThumb ? thumbUrl : '' })"
        >
          {{ room.sleeping ? 'Wake & join' : 'Join' }}
        </button>
      </div>
    </div>
  </article>
</template>

<style lang="scss" scoped>
.nr-room-card {
  position: relative;
  display: flex;
  flex-direction: column;
  border-radius: var(--radius-lg);
  border: 1px solid var(--border-subtle);
  background: var(--bg-surface-1);
  cursor: pointer;
  outline: none;
  overflow: hidden;
  transition: border-color var(--dur) var(--ease);
  &:hover { border-color: var(--border-hover); }
  &:focus-visible { border-color: var(--border-focus); box-shadow: var(--focus-ring); }
}
.nr-room-card--offline .nr-thumb-img { filter: grayscale(.6) brightness(.7); }

.nr-thumb {
  position: relative;
  aspect-ratio: 16 / 9;
  overflow: hidden;
  background: var(--bg-surface-2);
  border-bottom: 1px solid var(--border-subtle);
}
.nr-thumb-img {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.nr-thumb-fallback {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  .v-icon { color: var(--text-tertiary) !important; }
}
.nr-thumb-top, .nr-thumb-bottom {
  position: absolute;
  left: 12px;
  right: 12px;
  display: flex;
  align-items: center;
  gap: 8px;
}
.nr-thumb-top { top: 12px; }
.nr-thumb-bottom {
  bottom: 12px;
  height: 26px;
  padding: 0 8px 0 4px;
  right: auto;
  max-width: calc(100% - 24px);
  border-radius: var(--radius-sm);
  background: rgba(9, 9, 11, .78);
  color: var(--text-primary);
  font-size: 12px;
  font-weight: 500;
  .nr-stack { margin-right: 8px; }
}
.nr-viewers { color: var(--text-primary) !important; }
.nr-party { color: var(--text-primary) !important; }
.nr-glyph {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: var(--radius-sm);
  background: rgba(9, 9, 11, .78);
  .v-icon { color: var(--text-secondary) !important; font-size: 12px !important; }
}

.nr-body {
  display: flex;
  flex-direction: column;
  flex: 1;
  gap: 8px;
  padding: 16px;
}
.nr-room-title {
  font-size: 16px;
  font-weight: 600;
  letter-spacing: -0.025em;
  color: var(--text-primary);
}
.nr-room-playing {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  color: var(--text-secondary);
  .v-icon { color: var(--status-online) !important; flex: none; }
}
.nr-room-meta {
  display: flex;
  align-items: center;
  min-width: 0;
  font-size: 13px;
  color: var(--text-tertiary);
}
.nr-sep { margin: 0 6px; }
.nr-room-desc {
  font-size: 13px;
  line-height: 1.5;
  color: var(--text-secondary);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.nr-room-foot {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: auto;
  padding-top: 8px;
}
.nr-stack {
  display: flex;
  align-items: center;
  > * + * { margin-left: -6px; }
}
.nr-more, .nr-empty-note {
  font-size: 11.5px;
  color: var(--text-tertiary);
}
.nr-more { margin-left: 8px; }
</style>

<script lang="ts">
import { Vue, Component, Prop, Watch } from 'vue-property-decorator'
import { api, Room } from '@/api-ext'
import UserAvatar from '@/components/community/UserAvatar.vue'
import { relativeTime } from '@/utils/join'

const CATEGORY_ICONS: Record<string, string> = {
  movies: 'mdi-movie-open-outline',
  music: 'mdi-music',
  gaming: 'mdi-controller-classic-outline',
  sports: 'mdi-basketball',
  study: 'mdi-school-outline',
  chill: 'mdi-sofa-outline',
  tv: 'mdi-television-classic',
  anime: 'mdi-star-face',
}

@Component({
  components: {
    UserAvatar,
  }
})
export default class RoomCard extends Vue {
  @Prop({ required: true }) readonly room!: Room
  @Prop(Boolean) readonly compact!: boolean
  @Prop({ type: Number, default: 0 }) readonly tick!: number
  @Prop(String) readonly invite!: string

  thumbFailed = false

  @Watch('tick')
  onTick() {
    this.thumbFailed = false
  }

  get showThumb() {
    return this.room.has_thumbnail && this.room.ready && !this.thumbFailed
  }

  get thumbUrl() {
    return api.thumbnailUrl(this.room.name, this.tick, this.invite)
  }

  get partyAt(): Date | null {
    if (!this.room.scheduled_at || this.room.ready) return null
    return new Date(this.room.scheduled_at)
  }

  get partyLabel(): string {
    // tick is a counter that changes every so often, it keeps this fresh
    void this.tick
    return this.partyAt ? relativeTime(this.partyAt) : ''
  }

  get emptyNote(): string {
    if (this.room.ready) return 'Nobody here yet'
    if (this.partyAt && this.room.rsvps) return this.room.rsvps + ' going'
    if (this.room.sleeping) return 'Asleep, joining wakes it'
    return 'Not running'
  }

  get categoryIcon() {
    return CATEGORY_ICONS[(this.room.category || '').toLowerCase()] || 'mdi-play-circle-outline'
  }

  get visibilityIcon() {
    switch (this.room.visibility) {
      case 'public': return 'mdi-earth'
      case 'friends': return 'mdi-account-multiple'
      default: return 'mdi-lock'
    }
  }

  get visibilityLabel() {
    switch (this.room.visibility) {
      case 'public': return 'Public room'
      case 'friends': return 'Friends only'
      default: return 'Private, invite only'
    }
  }

  avatarOf(name: string): string | undefined {
    return this.room.avatars ? this.room.avatars[name] : undefined
  }

  open() {
    this.$router.push({ name: 'room', params: { name: this.room.name }, query: this.invite ? { invite: this.invite } : {} })
  }
}
</script>
