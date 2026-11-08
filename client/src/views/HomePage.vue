<template>
  <div class="nr-home">
    <!-- hero -->
    <section v-if="showHero" class="nr-hero nr-container">
      <div v-if="home.hero_image" class="nr-hero-image" :style="heroImageStyle" />
      <div class="nr-hero-grid" :class="{ 'has-parties': upcoming.length }">
        <div class="nr-hero-copy nr-rise">
          <span class="nr-eyebrow">
            <span class="nr-dot" />{{ liveRooms.length }} live {{ liveRooms.length === 1 ? 'room' : 'rooms' }} · {{ totalViewers }} watching
          </span>
          <h1 class="nr-hero-title">{{ expand(home.hero_title) }}</h1>
          <p class="nr-hero-subtitle">{{ expand(home.hero_subtitle) }}</p>
          <div class="nr-hero-actions">
            <button type="button" class="nr-btn is-primary is-lg" @click="create">{{ home.cta_label || 'Create a room' }}</button>
            <button type="button" class="nr-btn is-secondary is-lg" @click="scrollToRooms">Browse rooms</button>
          </div>
        </div>

        <aside v-if="upcoming.length" class="nr-upnext nr-surface nr-rise" aria-label="Upcoming watch parties">
          <div class="nr-upnext-head"><span class="nr-label">Up next</span></div>
          <router-link
            v-for="r in upcoming.slice(0, 3)"
            :key="'up' + r.id"
            :to="{ name: 'room', params: { name: r.name } }"
            class="nr-upnext-row"
          >
            <span class="nr-upnext-date">
              <span class="nr-label">{{ partyDay(r) }}</span>
              <span class="nr-mono">{{ partyDate(r) }}</span>
            </span>
            <span class="nr-upnext-copy">
              <span class="nr-upnext-title">{{ r.title }}</span>
              <span class="nr-mono nr-upnext-meta">{{ partyTime(r) }}<template v-if="r.category"> · {{ r.category }}</template> · {{ r.rsvps }} going</span>
            </span>
            <span class="nr-status" :class="r.going ? 'is-online' : 'is-plain'">{{ r.going ? 'Going' : 'RSVP' }}</span>
          </router-link>
        </aside>
      </div>
    </section>

    <div ref="rooms" class="nr-wrap nr-container" :class="{ 'is-headless': !showHero }">
      <!-- page heading for signed in users -->
      <div v-if="!showHero" class="nr-page-head nr-rise">
        <h1 class="nr-page-title">{{ branding.tagline || 'Rooms' }}</h1>
        <span class="nr-eyebrow"><span class="nr-dot" />{{ liveRooms.length }} live · {{ totalViewers }} watching</span>
      </div>

      <!-- toolbar -->
      <div class="nr-toolbar">
        <div class="nr-filters" v-if="home.show_categories && categories.length" role="group" aria-label="Category">
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
        </div>
      </div>

      <div v-if="!loaded" class="nr-grid" aria-busy="true">
        <div v-for="n in 3" :key="'sk' + n" class="nr-skeleton">
          <div class="nr-skeleton-thumb" />
          <div class="nr-skeleton-line" />
          <div class="nr-skeleton-line is-short" />
        </div>
      </div>

      <template v-else>
        <section v-for="section in sections" :key="section.key" class="nr-section">
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

        <div v-if="!sections.length" class="nr-empty-state">
          <div class="nr-empty-icon"><v-icon>{{ search || category ? 'mdi-magnify' : 'mdi-television-play' }}</v-icon></div>
          <h2>{{ search || category ? 'No rooms match your search' : (home.empty_text || 'No watch-party rooms running') }}</h2>
          <p v-if="!search && !category">A room is a shared browser in its own container. Spin one up and send the link around.</p>
          <button v-if="!search && !category" type="button" class="nr-btn is-primary" @click="create">
            <v-icon small>mdi-plus</v-icon>{{ home.cta_label || 'Create a room' }}
          </button>
          <button v-else type="button" class="nr-btn is-secondary" @click="search = ''; category = null">Clear filters</button>
        </div>
      </template>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.nr-hero {
  position: relative;
  padding-top: clamp(48px, 10vh, 96px);
}
.nr-hero-image {
  position: absolute;
  inset: 0;
  background-size: cover;
  background-position: center;
  opacity: .16;
  pointer-events: none;
}
.nr-hero-grid {
  position: relative;
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 48px;
  align-items: end;
  padding-bottom: clamp(40px, 7vh, 64px);
  border-bottom: 1px solid var(--border-subtle);
  &.has-parties { grid-template-columns: minmax(0, 1.4fr) minmax(0, 1fr); }
}
.nr-hero-copy {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 24px;
}
.nr-hero-title {
  margin: 0;
  max-width: 14ch;
  font-size: clamp(44px, 6vw, 72px) !important;
  font-weight: 600 !important;
  line-height: .95 !important;
  letter-spacing: -0.045em !important;
  color: var(--text-primary);
}
.nr-hero-subtitle {
  margin: 0;
  max-width: 460px;
  font-size: 16px;
  line-height: 1.55;
  color: var(--text-secondary);
}
.nr-hero-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.nr-upnext { display: flex; flex-direction: column; overflow: hidden; }
.nr-upnext-head { padding: 12px 16px; }
.nr-upnext-row {
  display: grid;
  grid-template-columns: 56px minmax(0, 1fr) auto;
  align-items: center;
  gap: 16px;
  padding: 12px 16px;
  border-top: 1px solid var(--border-subtle);
  color: var(--text-primary) !important;
  text-decoration: none;
  transition: background-color var(--dur) var(--ease);
  &:hover { background: rgba(255, 255, 255, .02); }
}
.nr-upnext-date {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 8px 0;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-md);
  .nr-label { font-size: 10px !important; }
  .nr-mono { font-size: 18px; font-weight: 500; }
}
.nr-upnext-copy { display: flex; flex-direction: column; gap: 4px; min-width: 0; }
.nr-upnext-title { font-size: 14px; font-weight: 500; letter-spacing: -0.02em; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.nr-upnext-meta { font-size: 11.5px; color: var(--text-tertiary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }

.nr-wrap {
  padding-top: 32px;
  padding-bottom: 64px;
  &.is-headless { padding-top: 48px; }
}
.nr-page-head {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  justify-content: space-between;
  gap: 8px 24px;
  margin-bottom: 32px;
}

.nr-toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 32px;
}
.nr-filters {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.nr-filter {
  height: 30px;
  padding: 0 12px;
  border-radius: 7px;
  border: 1px solid var(--border-subtle);
  background: transparent;
  color: var(--text-secondary);
  font: 500 13px var(--font-sans);
  cursor: pointer;
  transition: color var(--dur) var(--ease), background-color var(--dur) var(--ease), border-color var(--dur) var(--ease), transform var(--dur) var(--ease);
  &:hover { color: var(--text-primary); border-color: var(--border-hover); }
  &:active { transform: scale(.985); }
  &.is-active { background: var(--bg-surface-3); border-color: var(--border-hover); color: var(--text-primary); }
}
.nr-toolbar-end {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-left: auto;
}
.nr-search {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 38px;
  width: min(320px, 60vw);
  min-width: 0;
  padding: 0 12px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--border-subtle);
  background: var(--bg-surface-2);
  transition: border-color var(--dur) var(--ease), box-shadow var(--dur) var(--ease);
  cursor: text;
  .v-icon { color: var(--text-tertiary) !important; }
  input {
    flex: 1;
    width: 100%;
    min-width: 0;
    border: 0;
    outline: 0;
    background: transparent;
    color: var(--text-primary);
    font: 400 13.5px var(--font-sans);
    &::placeholder { color: var(--text-tertiary); }
    &::-webkit-search-cancel-button { filter: grayscale(1); opacity: .5; }
  }
  &:hover { border-color: var(--border-hover); }
  &:focus-within { border-color: var(--border-focus); box-shadow: var(--focus-ring); }
}

.nr-section { margin-bottom: 48px; }
.nr-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 16px;
  &.nr-grid--compact { grid-template-columns: repeat(auto-fill, minmax(220px, 1fr)); }
}

// loading skeletons keep the grid's shape, so nothing jumps when rooms arrive
.nr-skeleton {
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-lg);
  background: var(--bg-surface-1);
  overflow: hidden;
  padding-bottom: 16px;
}
.nr-skeleton-thumb { aspect-ratio: 16 / 9; background: var(--bg-surface-2); border-bottom: 1px solid var(--border-subtle); animation: nr-pulse 1.6s ease-in-out infinite; }
.nr-skeleton-line {
  height: 10px;
  margin: 16px 16px 0;
  width: 60%;
  border-radius: 4px;
  background: var(--bg-surface-2);
  animation: nr-pulse 1.6s ease-in-out infinite;
  &.is-short { width: 35%; margin-top: 10px; }
}

@media (max-width: 960px) {
  .nr-hero-grid.has-parties { grid-template-columns: minmax(0, 1fr); }
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
      // sleeping rooms and upcoming parties are as good as running ones
      if (!r.running && !r.sleeping && !r.scheduled_at && !r.is_owner && !this.home.show_offline) return false
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

    // upcoming parties get their own section, after the rooms that are on now
    const parties: Section = { key: 'parties', label: 'Upcoming watch parties', icon: 'mdi-calendar-star', color: 'primary', rooms: take(r => !!r.scheduled_at && !r.ready).sort((a, b) => (a.scheduled_at || '').localeCompare(b.scheduled_at || '')) }
    const sections: Section[] = [
      { key: 'featured', label: this.home.featured_label || 'Featured', icon: 'mdi-star', color: 'amber', rooms: take(r => r.featured && r.visibility === 'public') },
      { key: 'friends', label: this.home.friends_label || "Friends' rooms", icon: 'mdi-account-heart', color: 'accent', rooms: take(r => r.is_friend || r.friends_inside.length > 0) },
      { key: 'public', label: this.home.public_label || 'Public rooms', icon: 'mdi-earth', color: 'info', rooms: take(r => r.visibility === 'public' && !r.is_owner) },
      { key: 'mine', label: this.home.mine_label || 'Your rooms', icon: 'mdi-television-play', color: 'primary', rooms: take(r => r.is_owner) },
    ]
    sections.push(parties)
    return sections.filter(s => s.rooms.length)
  }

  // watch parties that have not started, soonest first, for the hero
  get upcoming(): Room[] {
    return this.rooms
      .filter(r => !!r.scheduled_at && !r.ready)
      .sort((a, b) => (a.scheduled_at || '').localeCompare(b.scheduled_at || ''))
  }

  partyDay(r: Room): string {
    return r.scheduled_at ? new Date(r.scheduled_at).toLocaleDateString(undefined, { weekday: 'short' }) : ''
  }

  partyDate(r: Room): string {
    return r.scheduled_at ? String(new Date(r.scheduled_at).getDate()) : ''
  }

  partyTime(r: Room): string {
    return r.scheduled_at ? new Date(r.scheduled_at).toLocaleTimeString(undefined, { hour: '2-digit', minute: '2-digit' }) : ''
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
