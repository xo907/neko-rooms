<template>
  <div>
    <!-- hero -->
    <section v-if="showHero" class="nr-hero" :style="heroStyle">
      <v-container class="py-12 py-md-16 position-relative">
        <v-row align="center">
          <v-col cols="12" md="7">
            <h1 class="nr-hero-title nr-heading mb-4">{{ expand(home.hero_title) }}</h1>
            <p class="nr-hero-subtitle mb-8">{{ expand(home.hero_subtitle) }}</p>
            <div class="d-flex flex-wrap" style="gap: 12px">
              <v-btn x-large color="white" class="primary--text font-weight-bold" depressed @click="create">
                <v-icon left>mdi-plus-circle</v-icon>{{ home.cta_label || 'Create a room' }}
              </v-btn>
              <v-btn x-large outlined color="white" @click="scrollToRooms">
                <v-icon left>mdi-compass-outline</v-icon>Browse rooms
              </v-btn>
            </div>
          </v-col>
          <v-col cols="12" md="5" class="d-none d-md-flex justify-center">
            <div class="nr-hero-stats">
              <div class="nr-hero-stat">
                <div class="nr-hero-stat-value">{{ liveRooms.length }}</div>
                <div class="nr-hero-stat-label">live rooms</div>
              </div>
              <div class="nr-hero-stat">
                <div class="nr-hero-stat-value">{{ totalViewers }}</div>
                <div class="nr-hero-stat-label">people watching</div>
              </div>
            </div>
          </v-col>
        </v-row>
      </v-container>
    </section>

    <v-container ref="rooms" class="py-6" :class="{ 'pt-8': !showHero }">
      <v-row>
        <v-col cols="12" :md="showFriends ? 9 : 12">
          <!-- toolbar -->
          <div class="d-flex flex-wrap align-center mb-4" style="gap: 12px">
            <h2 v-if="!showHero" class="nr-page-title nr-heading mr-4">{{ branding.tagline || 'Rooms' }}</h2>
            <v-text-field
              v-if="home.show_search"
              v-model="search"
              dense
              solo
              flat
              hide-details
              clearable
              prepend-inner-icon="mdi-magnify"
              placeholder="Search rooms, people, categories"
              class="nr-search"
            />
            <v-spacer />
            <v-btn v-if="user && canCreate" color="primary" depressed @click="create">
              <v-icon left>mdi-plus</v-icon>New room
            </v-btn>
          </div>

          <div v-if="home.show_categories && categories.length" class="mb-6">
            <v-chip-group v-model="category" column active-class="primary white--text">
              <v-chip v-for="c in categories" :key="c" :value="c" small filter outlined>{{ c }}</v-chip>
            </v-chip-group>
          </div>

          <div v-if="!loaded" class="text-center py-16">
            <v-progress-circular indeterminate color="primary" size="48" />
          </div>

          <template v-else>
            <section v-for="section in sections" :key="section.key" class="mb-8">
              <div class="d-flex align-center mb-3">
                <v-icon class="mr-2" :color="section.color">{{ section.icon }}</v-icon>
                <h3 class="text-h6 font-weight-bold nr-heading">{{ section.label }}</h3>
                <span class="ml-2 text--secondary">{{ section.rooms.length }}</span>
              </div>
              <v-row dense>
                <v-col
                  v-for="room in section.rooms"
                  :key="section.key + room.id"
                  cols="12"
                  sm="6"
                  :md="showFriends ? 6 : 4"
                  :lg="home.card_style === 'compact' ? 3 : (showFriends ? 4 : 3)"
                >
                  <RoomCard :room="room" :tick="tick" :compact="home.card_style === 'compact'" @join="join" />
                </v-col>
              </v-row>
            </section>

            <v-card v-if="!sections.length" flat class="text-center pa-12 nr-empty">
              <v-icon size="64" color="primary" class="mb-4">mdi-sofa-outline</v-icon>
              <div class="text-h6 mb-2">{{ search || category ? 'No rooms match your search.' : (home.empty_text || 'No rooms yet.') }}</div>
              <v-btn v-if="!search && !category" color="primary" depressed class="mt-4" @click="create">
                <v-icon left>mdi-plus</v-icon>{{ home.cta_label || 'Create a room' }}
              </v-btn>
            </v-card>
          </template>
        </v-col>

        <v-col v-if="showFriends" cols="12" md="3">
          <div class="nr-sticky">
            <FriendsPanel :rooms="rooms" @join="join" @open="openRoom" />
          </div>
        </v-col>
      </v-row>
    </v-container>
  </div>
</template>

<style lang="scss" scoped>
.nr-hero {
  position: relative;
  color: white;
  overflow: hidden;
  background-size: cover;
  background-position: center;
}
.nr-hero::after {
  content: '';
  position: absolute;
  inset: auto 0 0 0;
  height: 80px;
  background: linear-gradient(180deg, transparent, var(--nr-bg));
  pointer-events: none;
}
.nr-hero-title {
  font-size: clamp(2.2rem, 5vw, 3.6rem);
  line-height: 1.05;
  font-weight: 800;
  letter-spacing: -0.02em;
}
.nr-hero-subtitle {
  font-size: 1.2rem;
  opacity: .92;
  max-width: 560px;
}
.nr-hero-stats {
  display: flex;
  gap: 18px;
}
.nr-hero-stat {
  background: rgba(255, 255, 255, .14);
  backdrop-filter: blur(8px);
  border-radius: calc(var(--nr-radius) * 1.5);
  padding: 22px 28px;
  text-align: center;
  min-width: 150px;
}
.nr-hero-stat-value {
  font-size: 2.6rem;
  font-weight: 800;
  line-height: 1;
}
.nr-hero-stat-label {
  opacity: .85;
  margin-top: 6px;
}
.nr-search {
  max-width: 420px;
  min-width: 220px;
}
.nr-sticky {
  position: sticky;
  top: 80px;
}
.position-relative { position: relative; z-index: 1; }
</style>

<script lang="ts">
import { Vue, Component, Watch } from 'vue-property-decorator'
import { Branding, Room, User } from '@/api-ext'
import { expand } from '@/branding/theme'
import { joinRoom } from '@/utils/join'
import RoomCard from '@/components/community/RoomCard.vue'
import FriendsPanel from '@/components/community/FriendsPanel.vue'

interface Section {
  key: string
  label: string
  icon: string
  color: string
  rooms: Room[]
}

@Component({
  components: {
    RoomCard,
    FriendsPanel,
  }
})
export default class HomePage extends Vue {
  search = ''
  category: string | null = null
  tick = Math.floor(Date.now() / 30000)

  private refreshTimer = 0
  private tickTimer = 0

  get branding(): Branding {
    return this.$store.state.app.branding
  }

  get home() {
    return this.branding.home
  }

  get user(): User | null {
    return this.$store.getters.user
  }

  get canCreate(): boolean {
    return this.$store.getters.canCreateRooms
  }

  get rooms(): Room[] {
    return this.$store.state.app.directory
  }

  get loaded(): boolean {
    return this.$store.state.app.directoryLoaded
  }

  get showHero(): boolean {
    switch (this.home.hero_show) {
      case 'always': return true
      case 'never': return false
      default: return !this.user
    }
  }

  get heroStyle() {
    const layers: string[] = []
    if (this.home.hero_gradient) {
      layers.push('linear-gradient(120deg, color-mix(in srgb, var(--nr-primary) 88%, transparent), color-mix(in srgb, var(--nr-accent) 80%, transparent))')
    }
    if (this.home.hero_image) {
      layers.push(`url("${this.home.hero_image.replace(/"/g, '')}")`)
    }
    if (!layers.length) {
      layers.push('linear-gradient(120deg, var(--nr-primary), var(--nr-accent))')
    }
    return { backgroundImage: layers.join(', ') }
  }

  get showFriends(): boolean {
    const policy = this.$store.getters.policy
    return !!this.user && !!policy && policy.friends_enabled && this.home.show_friends
  }

  get liveRooms(): Room[] {
    return this.rooms.filter(r => r.ready)
  }

  get totalViewers(): number {
    return this.rooms.reduce((sum, r) => sum + (r.viewers || 0), 0)
  }

  get categories(): string[] {
    const set = new Set<string>(this.home.categories || [])
    this.rooms.forEach(r => { if (r.category) set.add(r.category) })
    return Array.from(set)
  }

  get filtered(): Room[] {
    const q = (this.search || '').trim().toLowerCase()
    return this.rooms.filter(r => {
      if (!r.running && !r.is_owner && !this.home.show_offline) return false
      if (this.category && r.category !== this.category) return false
      if (!q) return true
      return [r.title, r.name, r.description, r.category, r.owner?.username, r.owner?.display_name, ...r.members]
        .some(v => v && v.toLowerCase().includes(q))
    })
  }

  get sections(): Section[] {
    const used = new Set<string>()
    const take = (pred: (r: Room) => boolean) => {
      const list = this.filtered.filter(r => !used.has(r.id) && pred(r))
      list.forEach(r => used.add(r.id))
      return list
    }

    const sections: Section[] = [
      { key: 'featured', label: this.home.featured_label || 'Featured', icon: 'mdi-star', color: 'amber', rooms: take(r => r.featured && r.visibility === 'public') },
      { key: 'friends', label: this.home.friends_label || "Friends' rooms", icon: 'mdi-account-heart', color: 'accent', rooms: take(r => r.is_friend || r.friends_inside.length > 0) },
      { key: 'public', label: this.home.public_label || 'Public rooms', icon: 'mdi-earth', color: 'info', rooms: take(r => r.visibility === 'public' && !r.is_owner) },
      { key: 'mine', label: this.home.mine_label || 'Your rooms', icon: 'mdi-television-play', color: 'primary', rooms: take(r => r.is_owner) },
    ]
    return sections.filter(s => s.rooms.length)
  }

  expand(s: string) {
    return expand(this.branding, s)
  }

  async load() {
    if (document.hidden) return
    try {
      await this.$store.dispatch('APP_DIRECTORY')
    } catch (e) {
      console.error('unable to load rooms', e)
    }
  }

  join(room: Room) {
    joinRoom(this, room.name)
  }

  openRoom(room: Room) {
    this.$router.push({ name: 'room', params: { name: room.name } })
  }

  create() {
    if (!this.user) {
      const policy = this.$store.getters.policy
      this.$router.push({ name: policy && policy.registration_enabled ? 'register' : 'login', query: { redirect: '/my/rooms?create=1' } })
      return
    }
    this.$router.push({ name: 'my-rooms', query: { create: '1' } })
  }

  scrollToRooms() {
    const el = (this.$refs.rooms as Vue).$el as HTMLElement
    el.scrollIntoView({ behavior: 'smooth' })
  }

  @Watch('user')
  onUser() {
    this.load()
  }

  mounted() {
    this.load()
    this.refreshTimer = window.setInterval(this.load, 10000)
    this.tickTimer = window.setInterval(() => { this.tick = Math.floor(Date.now() / 30000) }, 30000)
  }

  beforeDestroy() {
    window.clearInterval(this.refreshTimer)
    window.clearInterval(this.tickTimer)
  }
}
</script>
