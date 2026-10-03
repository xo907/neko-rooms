<template>
  <v-card class="nr-room-card" :class="{ 'nr-room-card--offline': !room.ready }" @click="open" :ripple="false">
    <div class="nr-thumb" :style="thumbStyle">
      <img v-if="showThumb" :src="thumbUrl" @error="thumbFailed = true" alt="" class="nr-thumb-img">
      <div v-else class="nr-thumb-fallback">
        <v-icon size="54" color="white" style="opacity: .85">{{ categoryIcon }}</v-icon>
      </div>

      <div class="nr-thumb-shade" />

      <div class="nr-thumb-top d-flex align-center">
        <v-chip v-if="room.ready" x-small color="error" class="font-weight-bold mr-1 px-2" label>
          <span class="nr-live-dot" /> LIVE
        </v-chip>
        <v-chip v-else-if="room.paused" x-small color="warning" label class="px-2">PAUSED</v-chip>
        <v-chip v-else x-small color="grey darken-2" label class="px-2" dark>OFFLINE</v-chip>
        <v-chip v-if="room.ready" x-small label class="px-2 nr-chip-dark" dark>
          <v-icon x-small left>mdi-eye</v-icon>{{ room.viewers }}
        </v-chip>
        <v-spacer />
        <v-icon v-if="room.featured" small color="amber" class="mr-1">mdi-star</v-icon>
        <v-tooltip bottom>
          <template v-slot:activator="{ on, attrs }">
            <v-icon v-bind="attrs" v-on="on" small color="white">{{ visibilityIcon }}</v-icon>
          </template>
          <span>{{ visibilityLabel }}</span>
        </v-tooltip>
      </div>

      <div v-if="room.friends_inside.length" class="nr-thumb-bottom d-flex align-center">
        <div class="nr-stack mr-2">
          <UserAvatar v-for="n in room.friends_inside.slice(0, 3)" :key="'f' + n" :name="n" :size="22" ring />
        </div>
        <span class="caption white--text text-truncate">
          {{ room.friends_inside.slice(0, 2).join(', ') }}<span v-if="room.friends_inside.length > 2"> +{{ room.friends_inside.length - 2 }}</span>
          {{ room.friends_inside.length === 1 ? 'is' : 'are' }} here
        </span>
      </div>
    </div>

    <div class="pa-3">
      <div class="d-flex align-start">
        <div class="flex-grow-1" style="min-width: 0">
          <div class="nr-room-title text-truncate">{{ room.title }}</div>
          <div class="caption text--secondary d-flex align-center mt-1">
            <template v-if="room.owner">
              <UserAvatar :user="room.owner" :size="18" class="mr-1" />
              <span class="text-truncate">{{ room.owner.display_name || room.owner.username }}</span>
            </template>
            <span v-else class="text-truncate">/{{ room.name }}</span>
            <template v-if="room.category">
              <span class="mx-1">·</span>
              <span class="text-truncate">{{ room.category }}</span>
            </template>
          </div>
        </div>
      </div>

      <div v-if="!compact && room.description" class="body-2 text--secondary mt-2 nr-clamp">{{ room.description }}</div>

      <div class="d-flex align-center mt-3">
        <div v-if="room.members.length" class="nr-stack">
          <UserAvatar v-for="m in room.members.slice(0, 5)" :key="'m' + m" :name="m" :size="26" ring tooltip />
          <span v-if="room.members.length > 5" class="caption ml-2 text--secondary">+{{ room.members.length - 5 }}</span>
        </div>
        <span v-else class="caption text--secondary">{{ room.ready ? 'Nobody here yet' : 'Not running' }}</span>
        <v-spacer />
        <v-btn v-if="room.can_manage" icon small :to="{ name: 'room', params: { name: room.name } }" @click.stop>
          <v-icon small>mdi-cog-outline</v-icon>
        </v-btn>
        <v-btn
          v-if="room.can_join"
          color="primary"
          depressed
          small
          :disabled="!room.running"
          @click.stop="$emit('join', room)"
        >
          Join
        </v-btn>
      </div>
    </div>
  </v-card>
</template>

<style lang="scss" scoped>
.nr-room-card {
  overflow: hidden;
  transition: transform .15s ease, box-shadow .15s ease;
  cursor: pointer;
  &:hover {
    transform: translateY(-3px);
    box-shadow: 0 10px 28px rgba(0, 0, 0, .35) !important;
  }
}
.nr-room-card--offline .nr-thumb { filter: grayscale(.6) brightness(.8); }
.nr-thumb {
  position: relative;
  aspect-ratio: 16 / 9;
  overflow: hidden;
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
}
.nr-thumb-shade {
  position: absolute;
  inset: 0;
  background: linear-gradient(180deg, rgba(0,0,0,.45) 0%, rgba(0,0,0,0) 35%, rgba(0,0,0,0) 60%, rgba(0,0,0,.6) 100%);
}
.nr-thumb-top, .nr-thumb-bottom {
  position: absolute;
  left: 8px;
  right: 8px;
}
.nr-thumb-top { top: 8px; }
.nr-thumb-bottom { bottom: 8px; }
.nr-chip-dark { background: rgba(0, 0, 0, .55) !important; }
.nr-live-dot {
  display: inline-block;
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: white;
  margin-right: 4px;
  animation: nr-pulse 1.6s infinite;
}
@keyframes nr-pulse {
  0% { opacity: 1; }
  50% { opacity: .3; }
  100% { opacity: 1; }
}
.nr-room-title {
  font-weight: 700;
  font-size: 1.05rem;
}
.nr-clamp {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.nr-stack {
  display: flex;
  align-items: center;
  > * + * { margin-left: -4px; }
}
</style>

<script lang="ts">
import { Vue, Component, Prop, Watch } from 'vue-property-decorator'
import { api, Room } from '@/api-ext'
import UserAvatar from '@/components/community/UserAvatar.vue'

const GRADIENTS = [
  ['#8c5cff', '#ff5ca8'],
  ['#3ec5ff', '#8c5cff'],
  ['#ff9a5c', '#ff5c8a'],
  ['#2ee59d', '#3ec5ff'],
  ['#5c7cff', '#00c2a8'],
  ['#e056fd', '#686de0'],
  ['#f6d365', '#fda085'],
]

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
    const [a, b] = GRADIENTS[h % GRADIENTS.length]
    return { background: `linear-gradient(135deg, ${a}, ${b})` }
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

  open() {
    this.$router.push({ name: 'room', params: { name: this.room.name }, query: this.invite ? { invite: this.invite } : {} })
  }
}
</script>
