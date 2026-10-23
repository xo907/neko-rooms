<template>
  <article
    class="nr-room-card"
    :class="{ 'nr-room-card--offline': !room.ready, 'nr-room-card--compact': compact }"
    tabindex="0"
    role="link"
    :aria-label="room.title"
    @click="open"
    @keydown.enter="open"
    @mousemove="spot"
  >
    <div ref="thumb" class="nr-thumb" :style="thumbStyle">
      <img v-if="showThumb" :src="thumbUrl" @error="thumbFailed = true" alt="" class="nr-thumb-img">
      <div v-else class="nr-thumb-fallback">
        <v-icon size="34">{{ categoryIcon }}</v-icon>
      </div>

      <div class="nr-thumb-shade" />

      <div class="nr-thumb-top">
        <span v-if="room.ready" class="nr-glass-chip nr-glass-chip--live"><span class="nr-live-dot" />Live</span>
        <span v-else-if="room.paused" class="nr-glass-chip">Paused</span>
        <span v-else-if="partyAt" class="nr-glass-chip nr-glass-chip--party"><v-icon x-small>mdi-calendar-star</v-icon>{{ partyLabel }}</span>
        <span v-else-if="room.sleeping" class="nr-glass-chip"><v-icon x-small>mdi-weather-night</v-icon>Sleeping</span>
        <span v-else class="nr-glass-chip">Offline</span>
        <span v-if="room.ready" class="nr-glass-chip">
          <v-icon x-small>mdi-eye-outline</v-icon>{{ room.viewers }}
        </span>
        <span class="flex-grow-1" />
        <span v-if="room.featured" class="nr-glass-chip nr-glass-chip--icon" title="Featured"><v-icon x-small>mdi-star</v-icon></span>
        <v-tooltip bottom>
          <template v-slot:activator="{ on, attrs }">
            <span v-bind="attrs" v-on="on" class="nr-glass-chip nr-glass-chip--icon"><v-icon x-small>{{ visibilityIcon }}</v-icon></span>
          </template>
          <span>{{ visibilityLabel }}</span>
        </v-tooltip>
      </div>

      <div v-if="room.friends_inside.length" class="nr-thumb-bottom">
        <div class="nr-stack mr-2">
          <UserAvatar v-for="n in room.friends_inside.slice(0, 3)" :key="'f' + n" :name="n" :src="avatarOf(n)" :size="20" ring />
        </div>
        <span class="text-truncate">
          {{ room.friends_inside.slice(0, 2).join(', ') }}<span v-if="room.friends_inside.length > 2"> +{{ room.friends_inside.length - 2 }}</span>
          {{ room.friends_inside.length === 1 ? 'is' : 'are' }} here
        </span>
      </div>
    </div>

    <div class="nr-body">
      <div class="nr-room-title text-truncate">{{ room.title }}</div>
      <div class="nr-room-meta">
        <template v-if="room.owner">
          <UserAvatar :user="room.owner" :size="16" class="mr-1" />
          <span class="text-truncate">{{ room.owner.display_name || room.owner.username }}</span>
        </template>
        <span v-else class="text-truncate">/{{ room.name }}</span>
        <template v-if="room.category">
          <span class="nr-sep">·</span>
          <span class="text-truncate">{{ room.category }}</span>
        </template>
      </div>

      <div v-if="room.now_playing" class="nr-room-playing text-truncate" :title="room.now_playing">
        <v-icon x-small>mdi-play</v-icon>{{ room.now_playing }}
      </div>
      <div v-if="!compact && room.description" class="nr-room-desc">{{ room.description }}</div>

      <div class="nr-room-foot">
        <div v-if="room.members.length" class="nr-stack">
          <UserAvatar v-for="m in room.members.slice(0, 5)" :key="'m' + m" :name="m" :src="avatarOf(m)" :size="24" ring tooltip />
          <span v-if="room.members.length > 5" class="nr-more">+{{ room.members.length - 5 }}</span>
        </div>
        <span v-else class="nr-empty-note">{{ emptyNote }}</span>
        <span class="flex-grow-1" />
        <v-btn v-if="room.can_manage" icon small class="nr-manage" :to="{ name: 'room', params: { name: room.name } }" @click.stop title="Manage room">
          <v-icon small>mdi-tune-variant</v-icon>
        </v-btn>
        <v-btn
          v-if="room.can_join"
          color="primary"
          depressed
          small
          :disabled="!room.running && !room.sleeping"
          @click.stop="$emit('join', room, { from: $refs.thumb, image: showThumb ? thumbUrl : '' })"
        >
          {{ room.sleeping ? 'Wake & join' : 'Join' }}
        </v-btn>
      </div>
    </div>
  </article>
</template>

<style lang="scss" scoped>
.nr-room-card {
  --mx: 50%;
  --my: 50%;
  position: relative;
  display: flex;
  flex-direction: column;
  padding: 6px;
  border-radius: calc(var(--nr-radius) * 1.3);
  border: 1px solid var(--nr-line);
  background: var(--nr-glass);
  backdrop-filter: blur(14px);
  cursor: pointer;
  outline: none;
  overflow: hidden;
  transition: transform .32s cubic-bezier(.2, .8, .2, 1), border-color .32s, box-shadow .32s;
  &::after {
    content: '';
    position: absolute;
    inset: 0;
    border-radius: inherit;
    pointer-events: none;
    opacity: 0;
    background: radial-gradient(240px circle at var(--mx) var(--my), color-mix(in srgb, var(--nr-primary) 14%, transparent), transparent 65%);
    transition: opacity .32s;
  }
  &:hover, &:focus-visible {
    transform: translateY(-2px);
    border-color: var(--nr-line-strong);
    box-shadow: 0 24px 60px -30px rgba(0, 0, 0, .7);
    &::after { opacity: 1; }
    .nr-thumb-img { transform: scale(1.03); }
  }
  &:focus-visible { border-color: var(--nr-primary); }
}
.nr-room-card--offline .nr-thumb { filter: grayscale(.7) brightness(.75); }

.nr-thumb {
  position: relative;
  aspect-ratio: 16 / 9;
  overflow: hidden;
  border-radius: var(--nr-radius);
  box-shadow: inset 0 0 0 1px var(--nr-line);
}
.nr-thumb-img {
  position: absolute;
  inset: 0;
  width: 100%;
  height: 100%;
  object-fit: cover;
  transition: transform .6s cubic-bezier(.2, .8, .2, 1);
}
.nr-thumb-fallback {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  .v-icon { color: color-mix(in srgb, var(--nr-text) 35%, transparent) !important; }
}
.nr-thumb-shade {
  position: absolute;
  inset: 0;
  background: linear-gradient(180deg, rgba(0, 0, 0, .35) 0%, transparent 30%, transparent 60%, rgba(0, 0, 0, .55) 100%);
  pointer-events: none;
}
.nr-thumb-top, .nr-thumb-bottom {
  position: absolute;
  left: 8px;
  right: 8px;
  display: flex;
  align-items: center;
  gap: 5px;
}
.nr-thumb-top { top: 8px; }
.nr-thumb-bottom {
  bottom: 8px;
  color: #fff;
  font-size: 12px;
  font-weight: 500;
}
.nr-glass-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  height: 22px;
  padding: 0 8px;
  border-radius: 999px;
  background: rgba(8, 8, 10, .55);
  backdrop-filter: blur(10px);
  box-shadow: inset 0 0 0 1px rgba(255, 255, 255, .1);
  color: rgba(255, 255, 255, .92);
  font-size: 11px;
  font-weight: 600;
  .v-icon { color: inherit !important; font-size: 12px !important; }
  &.nr-glass-chip--icon { width: 22px; padding: 0; justify-content: center; }
  &.nr-glass-chip--live { box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--nr-primary) 45%, transparent); }
  &.nr-glass-chip--party { color: var(--nr-primary); box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--nr-primary) 45%, transparent); .v-icon { color: inherit !important; } }
}
.nr-live-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--nr-primary);
  box-shadow: 0 0 8px var(--nr-primary);
  animation: nr-pulse 1.8s ease-in-out infinite;
}

.nr-body {
  position: relative;
  z-index: 1;
  display: flex;
  flex-direction: column;
  flex: 1;
  padding: 12px 8px 6px;
}
.nr-room-title {
  font-weight: 600;
  font-size: 15px;
  letter-spacing: -0.01em;
  color: var(--nr-text);
}
.nr-room-meta {
  display: flex;
  align-items: center;
  min-width: 0;
  margin-top: 4px;
  font-size: 12.5px;
  color: var(--nr-muted);
}
.nr-sep { margin: 0 6px; color: var(--nr-subtle); }
.nr-room-playing {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-top: 6px;
  font-size: 12.5px;
  font-weight: 500;
  color: var(--nr-text);
  .v-icon { color: var(--nr-primary) !important; flex: none; }
}
.nr-room-desc {
  margin-top: 8px;
  font-size: 13px;
  line-height: 1.5;
  color: var(--nr-muted);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.nr-room-foot {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: auto;
  padding-top: 14px;
}
.nr-stack {
  display: flex;
  align-items: center;
  > * + * { margin-left: -6px; }
}
.nr-more, .nr-empty-note {
  font-size: 12px;
  color: var(--nr-subtle);
}
.nr-more { margin-left: 8px; }
.nr-manage .v-icon { color: var(--nr-muted) !important; }
</style>

<script lang="ts">
import { Vue, Component, Prop, Watch } from 'vue-property-decorator'
import { api, Room } from '@/api-ext'
import UserAvatar from '@/components/community/UserAvatar.vue'
import { relativeTime } from '@/utils/join'

// subtle hue shifts, so placeholder thumbnails are told apart without
// turning the grid into a rainbow
const TINTS = [0, 22, 40, 60, 80]

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

  get thumbStyle() {
    let h = 0
    for (let i = 0; i < this.room.name.length; i++) h = (h * 31 + this.room.name.charCodeAt(i)) >>> 0
    const tint = TINTS[h % TINTS.length]
    const angle = 120 + (h % 5) * 30
    return {
      background: `radial-gradient(120% 90% at 0% 0%, color-mix(in srgb, var(--nr-primary) ${10 + tint / 4}%, transparent), transparent 60%),
        linear-gradient(${angle}deg, color-mix(in srgb, var(--nr-surface) 94%, var(--nr-text)), var(--nr-bg))`,
    }
  }

  spot(e: MouseEvent) {
    const el = e.currentTarget as HTMLElement
    const r = el.getBoundingClientRect()
    el.style.setProperty('--mx', `${e.clientX - r.left}px`)
    el.style.setProperty('--my', `${e.clientY - r.top}px`)
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
