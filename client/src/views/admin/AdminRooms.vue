<template>
  <div>
    <div class="d-flex flex-wrap align-center mb-4" style="gap: 12px">
      <v-text-field v-model="search" dense solo flat hide-details clearable prepend-inner-icon="mdi-magnify" placeholder="Search rooms or owners" style="max-width: 360px" />
      <v-btn-toggle v-model="filter" dense mandatory>
        <v-btn value="all" small>All</v-btn>
        <v-btn value="public" small>Public</v-btn>
        <v-btn value="featured" small>Featured</v-btn>
        <v-btn value="hidden" small>Hidden</v-btn>
        <v-btn value="unowned" small>No owner</v-btn>
      </v-btn-toggle>
      <v-spacer />
      <v-btn icon @click="load" :loading="loading"><v-icon>mdi-refresh</v-icon></v-btn>
      <v-btn color="primary" depressed :to="{ name: 'my-rooms', query: { create: '1' } }"><v-icon left>mdi-plus</v-icon>New room</v-btn>
    </div>

    <v-card>
      <v-data-table :headers="headers" :items="filtered" :loading="loading" :items-per-page="25" :footer-props="{ itemsPerPageOptions: [10, 25, 50, 100, -1] }" :search="search" :custom-filter="customFilter">
        <template v-slot:[`item.title`]="{ item }">
          <div class="d-flex align-center py-2">
            <div class="nr-mini-thumb mr-3" :style="thumbStyle(item)">
              <img v-if="item.ready && item.has_thumbnail && !failed[item.name]" :src="thumb(item)" @error="$set(failed, item.name, true)" alt="">
            </div>
            <div style="min-width: 0">
              <router-link :to="{ name: 'room', params: { name: item.name } }" class="font-weight-bold text-decoration-none">{{ item.title }}</router-link>
              <div class="caption text--secondary">/{{ item.name }}<span v-if="item.category"> · {{ item.category }}</span></div>
            </div>
          </div>
        </template>
        <template v-slot:[`item.owner`]="{ item }">
          <span v-if="item.owner">{{ item.owner.display_name || item.owner.username }}</span>
          <span v-else class="text--secondary font-italic">none</span>
        </template>
        <template v-slot:[`item.visibility`]="{ item }">
          <v-select
            :value="item.visibility"
            :items="visibilityItems"
            dense
            hide-details
            solo
            flat
            style="max-width: 140px"
            @change="update(item, { visibility: $event })"
          />
        </template>
        <template v-slot:[`item.featured`]="{ item }">
          <v-btn icon small @click="update(item, { featured: !item.featured })" :title="item.featured ? 'Unfeature' : 'Feature on homepage'">
            <v-icon :color="item.featured ? 'amber' : undefined">{{ item.featured ? 'mdi-star' : 'mdi-star-outline' }}</v-icon>
          </v-btn>
        </template>
        <template v-slot:[`item.hidden`]="{ item }">
          <v-btn icon small @click="update(item, { hidden: !item.hidden })" :title="item.hidden ? 'Unhide' : 'Hide from listings'">
            <v-icon :color="item.hidden ? 'error' : undefined">{{ item.hidden ? 'mdi-eye-off' : 'mdi-eye-outline' }}</v-icon>
          </v-btn>
        </template>
        <template v-slot:[`item.status`]="{ item }">
          <v-chip x-small :color="item.ready ? 'success' : (item.paused ? 'warning' : (item.running ? 'info' : 'grey'))" dark label>
            {{ item.ready ? 'live' : (item.paused ? 'paused' : (item.running ? 'starting' : 'stopped')) }}
          </v-chip>
          <span v-if="item.ready" class="caption ml-2"><v-icon x-small>mdi-eye</v-icon> {{ item.viewers }}</span>
        </template>
        <template v-slot:[`item.actions`]="{ item }">
          <div class="d-flex justify-end">
            <RoomActionBtn action="start" :roomId="item.id" :disabled="item.running" @done="load" />
            <RoomActionBtn action="stop" :roomId="item.id" :disabled="!item.running && !item.paused" @done="load" />
            <RoomActionBtn action="restart" :roomId="item.id" :disabled="!item.running" @done="load" />
            <v-btn icon @click="edit(item)" title="Edit details & owner"><v-icon>mdi-pencil-outline</v-icon></v-btn>
            <v-btn icon :disabled="!item.running" @click="join(item)" title="Join as admin"><v-icon>mdi-login-variant</v-icon></v-btn>
            <RoomActionBtn action="remove" :roomId="item.id" @done="load" />
          </div>
        </template>
      </v-data-table>
    </v-card>

    <v-dialog v-model="dialog" max-width="560">
      <v-card v-if="editing">
        <v-card-title>Edit {{ editing.title }}</v-card-title>
        <v-card-text>
          <RoomMetaEditor :roomId="editing.id" :name="editing.name" :meta="editingMeta" moderation cancelable @saved="onSaved" @cancel="dialog = false" />
        </v-card-text>
      </v-card>
    </v-dialog>
  </div>
</template>

<style lang="scss" scoped>
.nr-mini-thumb {
  width: 80px;
  min-width: 80px;
  height: 45px;
  border-radius: 6px;
  overflow: hidden;
  img { width: 100%; height: 100%; object-fit: cover; }
}
</style>

<script lang="ts">
import { Vue, Component } from 'vue-property-decorator'
import { api, errorMessage, Room, RoomMeta, RoomMetaUpdate } from '@/api-ext'
import { joinRoom } from '@/utils/join'
import RoomActionBtn from '@/components/RoomActionBtn.vue'
import RoomMetaEditor from '@/components/community/RoomMetaEditor.vue'

@Component({
  components: {
    RoomActionBtn,
    RoomMetaEditor,
  }
})
export default class AdminRooms extends Vue {
  rooms: Room[] = []
  loading = false
  search = ''
  filter = 'all'
  failed: Record<string, boolean> = {}
  tick = Date.now()
  dialog = false
  editing: Room | null = null
  editingMeta: RoomMeta | null = null

  headers = [
    { text: 'Room', value: 'title' },
    { text: 'Owner', value: 'owner', sort: (a: Room['owner'], b: Room['owner']) => (a?.username || '').localeCompare(b?.username || '') },
    { text: 'Visibility', value: 'visibility' },
    { text: 'Featured', value: 'featured', align: 'center' },
    { text: 'Listed', value: 'hidden', align: 'center' },
    { text: 'Status', value: 'status', sortable: false },
    { text: '', value: 'actions', sortable: false, align: 'end' },
  ]

  visibilityItems = [
    { text: 'Public', value: 'public' },
    { text: 'Friends', value: 'friends' },
    { text: 'Private', value: 'private' },
  ]

  get filtered(): Room[] {
    switch (this.filter) {
      case 'public': return this.rooms.filter(r => r.visibility === 'public')
      case 'featured': return this.rooms.filter(r => r.featured)
      case 'hidden': return this.rooms.filter(r => r.hidden)
      case 'unowned': return this.rooms.filter(r => !r.owner)
      default: return this.rooms
    }
  }

  customFilter(value: unknown, search: string, item: Room) {
    const q = search.toLowerCase()
    return [item.title, item.name, item.category, item.owner?.username, item.owner?.display_name].some(v => v && v.toLowerCase().includes(q))
  }

  thumb(room: Room) {
    return api.thumbnailUrl(room.name, this.tick)
  }

  thumbStyle(room: Room) {
    return { background: room.ready ? 'linear-gradient(135deg, var(--nr-primary), var(--nr-accent))' : 'var(--nr-secondary)' }
  }

  async load() {
    this.loading = true
    try {
      this.rooms = await api.adminRooms()
      this.tick = Date.now()
      this.failed = {}
    } catch (e) {
      this.$swal({ icon: 'error', title: 'Unable to load rooms', text: errorMessage(e) })
    } finally {
      this.loading = false
    }
  }

  async update(room: Room, data: RoomMetaUpdate) {
    try {
      await api.updateRoomMeta(room.id, data)
      await this.load()
    } catch (e) {
      this.$swal({ icon: 'error', title: 'Unable to update room', text: errorMessage(e) })
    }
  }

  async edit(room: Room) {
    this.editing = room
    this.editingMeta = await api.roomMeta(room.id)
    this.dialog = true
  }

  onSaved() {
    this.dialog = false
    this.load()
  }

  join(room: Room) {
    joinRoom(this, room.name)
  }

  mounted() {
    this.load()
  }
}
</script>
