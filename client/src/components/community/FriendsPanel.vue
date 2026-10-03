<template>
  <v-card class="nr-friends-panel">
    <v-card-title class="subtitle-1 font-weight-bold pb-2">
      <v-icon left small>mdi-account-multiple</v-icon> Friends
      <v-spacer />
      <v-btn small text color="primary" :to="{ name: 'friends' }">
        <v-badge :value="pending > 0" :content="pending" color="accent" inline>Manage</v-badge>
      </v-btn>
    </v-card-title>

    <v-card-text class="pb-2">
      <v-text-field
        v-model="username"
        dense
        outlined
        hide-details
        placeholder="Add by username"
        prepend-inner-icon="mdi-account-plus-outline"
        :loading="adding"
        @keydown.enter="add"
      />
    </v-card-text>

    <v-list dense class="pt-0" v-if="friends.length">
      <v-subheader v-if="online.length">IN A ROOM — {{ online.length }}</v-subheader>
      <v-list-item v-for="f in online" :key="'on' + f.user.id" @click="$emit('open', f.room)">
        <v-list-item-avatar size="32" class="my-1">
          <v-badge dot bordered color="success" overlap bottom offset-x="10" offset-y="10">
            <UserAvatar :user="f.user" :size="32" />
          </v-badge>
        </v-list-item-avatar>
        <v-list-item-content>
          <v-list-item-title class="font-weight-medium">{{ f.user.display_name || f.user.username }}</v-list-item-title>
          <v-list-item-subtitle>in {{ f.room.title }}</v-list-item-subtitle>
        </v-list-item-content>
        <v-list-item-action v-if="f.room.can_join">
          <v-btn x-small color="primary" depressed @click.stop="$emit('join', f.room)">Join</v-btn>
        </v-list-item-action>
      </v-list-item>

      <v-subheader v-if="offline.length">FRIENDS — {{ offline.length }}</v-subheader>
      <v-list-item v-for="f in offline" :key="'off' + f.user.id">
        <v-list-item-avatar size="32" class="my-1">
          <UserAvatar :user="f.user" :size="32" style="opacity: .6" />
        </v-list-item-avatar>
        <v-list-item-content>
          <v-list-item-title class="text--secondary">{{ f.user.display_name || f.user.username }}</v-list-item-title>
        </v-list-item-content>
      </v-list-item>
    </v-list>
    <v-card-text v-else class="text-center text--secondary caption pt-0">
      No friends yet. Add someone by their username to see when they are hanging out.
    </v-card-text>
  </v-card>
</template>

<script lang="ts">
import { Vue, Component, Prop } from 'vue-property-decorator'
import { api, errorMessage, Friend, Room } from '@/api-ext'
import UserAvatar from '@/components/community/UserAvatar.vue'

@Component({
  components: {
    UserAvatar,
  }
})
export default class FriendsPanel extends Vue {
  @Prop({ type: Array, default: () => [] }) readonly rooms!: Room[]

  username = ''
  adding = false

  get friends(): Friend[] {
    return (this.$store.state.app.friends as Friend[]).filter(f => f.status === 'accepted')
  }

  get pending(): number {
    return this.$store.getters.pendingFriendRequests
  }

  // presence: a friend is "in" a room when a member name matches their name
  roomOf(f: Friend): Room | undefined {
    const names = [f.user.username, f.user.display_name].filter(Boolean).map(n => n.toLowerCase())
    return this.rooms.find(r => r.ready && r.members.some(m => names.includes(m.toLowerCase())))
  }

  get online() {
    return this.friends.map(f => ({ ...f, room: this.roomOf(f) })).filter(f => f.room) as (Friend & { room: Room })[]
  }

  get offline() {
    return this.friends.filter(f => !this.roomOf(f))
  }

  async add() {
    const username = this.username.trim()
    if (!username) return
    this.adding = true
    try {
      const res = await api.addFriend(username)
      this.username = ''
      this.$swal({
        toast: true,
        position: 'bottom-end',
        timer: 2500,
        showConfirmButton: false,
        icon: 'success',
        title: res.status === 'accepted' ? `You are now friends with ${res.user.username}` : `Friend request sent to ${res.user.username}`,
      })
      await this.$store.dispatch('APP_FRIENDS')
    } catch (e) {
      this.$swal({ icon: 'error', title: 'Unable to add friend', text: errorMessage(e) })
    } finally {
      this.adding = false
    }
  }
}
</script>
