<template>
  <div class="nr-home">
    <!-- hero -->
    <section v-if="showHero" class="nr-hero">
      <div v-if="home.hero_image" class="nr-hero-image" :style="heroImageStyle" />
      <div class="nr-hero-inner nr-rise">
        <div class="nr-eyebrow mb-6">
          <span class="nr-dot" />
          <span>{{ liveRooms.length }} live {{ liveRooms.length === 1 ? 'room' : 'rooms' }}</span>
          <span class="nr-eyebrow-sep" />
          <span>{{ totalViewers }} watching</span>
        </div>
        <h1 class="nr-hero-title nr-display">{{ expand(home.hero_title) }}</h1>
        <p class="nr-hero-subtitle">{{ expand(home.hero_subtitle) }}</p>
        <div class="nr-hero-actions">
          <v-btn large color="primary" depressed @click="create">
            {{ home.cta_label || 'Create a room' }}
          </v-btn>
          <v-btn large outlined @click="scrollToRooms">
            Browse rooms
          </v-btn>
        </div>
      </div>
    </section>

    <div ref="rooms" class="nr-wrap" :class="showHero ? 'pt-2' : 'pt-10'">
      <div class="nr-layout">
        <div class="nr-main">
          <!-- page heading for signed in users -->
          <div v-if="!showHero" class="nr-page-head nr-rise">
            <h1 class="nr-page-title nr-heading">{{ branding.tagline || 'Rooms' }}</h1>
            <div class="nr-page-meta">
              <span class="nr-dot" />{{ liveRooms.length }} live · {{ totalViewers }} watching
            </div>
          </div>

          <!-- toolbar -->
          <div class="nr-toolbar">
            <div class="nr-filters" v-if="home.show_categories && categories.length">
              <button type="button" class="nr-filter" :class="{ 'is-active': !category }" @click="category = null">All</button>
              <button
                v-for="c in categories"
                :key="c"
                type="button"
                class="nr-filter"
                :class="{ 'is-active': category === c }"
                @click="category = category === c ? null : c"
              >{{ c }}</button>
            </div>
            <div class="nr-toolbar-end">
              <label v-if="home.show_search" class="nr-search">
                <v-icon small>mdi-magnify</v-icon>
                <input v-model="search" type="search" placeholder="Search rooms, people…" aria-label="Search rooms">
              </label>
              <v-btn v-if="user && canCreate" color="primary" depressed @click="create">
                <v-icon left small>mdi-plus</v-icon>New room
              </v-btn>
            </div>
          </div>

          <div v-if="!loaded" class="text-center py-16">
            <v-progress-circular indeterminate color="primary" size="32" width="2" />
          </div>

          <template v-else>
            <section v-for="section in sections" :key="section.key" class="mb-10">
              <h2 class="nr-section-title">{{ section.label }} <span>{{ section.rooms.length }}</span></h2>
              <div class="nr-grid" :class="{ 'nr-grid--compact': home.card_style === 'compact' }">
                <RoomCard
                  v-for="room in section.rooms"
                  :key="section.key + room.id"
                  :room="room"
                  :tick="tick"
                  :compact="home.card_style === 'compact'"
                  @join="join"
                />
              </div>
            </section>

            <div v-if="!sections.length" class="nr-empty">
              <div class="nr-empty-icon"><v-icon>mdi-sofa-outline</v-icon></div>
              <div class="nr-empty-title">{{ search || category ? 'No rooms match your search.' : (home.empty_text || 'No rooms yet.') }}</div>
              <v-btn v-if="!search && !category" color="primary" depressed class="mt-5" @click="create">
                {{ home.cta_label || 'Create a room' }}
              </v-btn>
              <v-btn v-else text class="mt-3" @click="search = ''; category = null">Clear filters</v-btn>
            </div>
          </template>
        </div>

      </div>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.nr-hero {
  position: relative;
  padding: clamp(64px, 12vh, 128px) 16px clamp(48px, 8vh, 88px);
  text-align: center;
  overflow: hidden;
}
.nr-hero-image {
  position: absolute;
  inset: 0;
  background-size: cover;
  background-position: center;
  opacity: .28;
  mask-image: radial-gradient(ellipse 70% 70% at 50% 30%, #000 30%, transparent 75%);
  -webkit-mask-image: radial-gradient(ellipse 70% 70% at 50% 30%, #000 30%, transparent 75%);
}
.nr-hero-inner {
  position: relative;
  max-width: 760px;
  margin: 0 auto;
}
.nr-eyebrow-sep {
  width: 1px;
  height: 12px;
  background: var(--nr-line-strong);
}
.nr-hero-title {
  font-size: clamp(2.8rem, 7.5vw, 5.2rem);
  line-height: .98;
  margin: 0 0 20px;
  color: var(--nr-text);
}
.nr-hero-subtitle {
  font-size: clamp(1rem, 1.6vw, 1.125rem);
  line-height: 1.6;
  color: var(--nr-muted);
  max-width: 520px;
  margin: 0 auto 32px;
}
.nr-hero-actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 10px;
}

.nr-wrap {
  max-width: 1240px;
  margin: 0 auto;
  padding: 0 16px 64px;
}
.nr-layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 32px;
}

.nr-page-head {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  justify-content: space-between;
  gap: 8px 24px;
  margin-bottom: 24px;
}
.nr-page-title {
  font-size: clamp(1.8rem, 3.5vw, 2.4rem);
  line-height: 1.1;
  margin: 0;
}
.nr-page-meta {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 13px;
  color: var(--nr-muted);
}

.nr-toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 28px;
  padding-bottom: 20px;
  border-bottom: 1px solid var(--nr-line);
}
.nr-filters {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.nr-filter {
  height: 32px;
  padding: 0 14px;
  border-radius: 999px;
  border: 1px solid var(--nr-line);
  background: var(--nr-hover);
  color: var(--nr-muted);
  font: inherit;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: color .18s, background-color .18s, border-color .18s, transform .18s;
  &:hover { color: var(--nr-text); border-color: var(--nr-line-strong); }
  &:active { transform: scale(.97); }
  &.is-active {
    background: var(--nr-primary);
    border-color: var(--nr-primary);
    color: var(--nr-on-primary);
  }
}
.nr-toolbar-end {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-left: auto;
}
.nr-search {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 36px;
  width: min(300px, 60vw);
  min-width: 0;
  padding: 0 14px;
  border-radius: 999px;
  border: 1px solid var(--nr-line);
  background: var(--nr-hover);
  transition: border-color .18s, box-shadow .18s;
  cursor: text;
  .v-icon { color: var(--nr-subtle) !important; }
  input {
    flex: 1;
    width: 100%;
    min-width: 0;
    border: 0;
    outline: 0;
    background: transparent;
    color: var(--nr-text);
    font: inherit;
    font-size: 13.5px;
    &::placeholder { color: var(--nr-subtle); }
    &::-webkit-search-cancel-button { filter: grayscale(1); opacity: .5; }
  }
  &:focus-within {
    border-color: color-mix(in srgb, var(--nr-primary) 55%, transparent);
    box-shadow: 0 0 0 4px color-mix(in srgb, var(--nr-primary) 12%, transparent);
  }
}

.nr-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 16px;
  &.nr-grid--compact { grid-template-columns: repeat(auto-fill, minmax(210px, 1fr)); }
}

.nr-empty {
  text-align: center;
  padding: 64px 24px;
  border: 1px dashed var(--nr-line-strong);
  border-radius: calc(var(--nr-radius) * 1.5);
}
.nr-empty-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  margin-bottom: 16px;
  border-radius: 50%;
  border: 1px solid var(--nr-line);
  background: var(--nr-hover);
  .v-icon { color: var(--nr-muted) !important; }
}
.nr-empty-title {
  color: var(--nr-muted);
  font-size: 15px;
}
@media (max-width: 600px) {
  .nr-toolbar-end { width: 100%; }
  .nr-search { flex: 1; width: auto; }
}
</style>

<script lang="ts">
import { Vue, Component, Watch } from 'vue-property-decorator'
import { Branding, Room, User } from '@/api-ext'
import { expand } from '@/branding/theme'
import { joinRoom, JoinOptions } from '@/utils/join'
import RoomCard from '@/components/community/RoomCard.vue'

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

  get heroImageStyle() {
    return { backgroundImage: `url("${this.home.hero_image.replace(/"/g, "")}")` }
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
    // background tabs only skip refreshes, never the first load
    if (document.hidden && this.loaded) return
    try {
      await this.$store.dispatch('APP_DIRECTORY')
    } catch (e) {
      console.error('unable to load rooms', e)
    }
  }

  join(room: Room, opts: JoinOptions = {}) {
    joinRoom(this, room.name, { ...opts, room })
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
    const el = this.$refs.rooms as HTMLElement
    el.scrollIntoView({ behavior: 'smooth' })
  }

  @Watch('user')
  onUser() {
    this.load()
  }

  onVisible() {
    if (!document.hidden) this.load()
  }

  mounted() {
    this.load()
    document.addEventListener('visibilitychange', this.onVisible)
    this.refreshTimer = window.setInterval(this.load, 10000)
    this.tickTimer = window.setInterval(() => { this.tick = Math.floor(Date.now() / 30000) }, 30000)
  }

  beforeDestroy() {
    document.removeEventListener('visibilitychange', this.onVisible)
    window.clearInterval(this.refreshTimer)
    window.clearInterval(this.tickTimer)
  }
}
</script>
