<template>
  <v-container class="py-8" style="max-width: 900px">
    <h1 class="nr-page-title nr-heading text-h4 mb-6">Friends</h1>

    <v-card class="mb-6">
      <v-card-text>
        <v-autocomplete
          v-model="selected"
          :search-input.sync="query"
          :items="results"
          :loading="searching"
          item-text="username"
          item-value="username"
          placeholder="Find people by username"
          prepend-inner-icon="mdi-account-search-outline"
          outlined
          hide-details
          hide-no-data
          no-filter
          return-object
          @change="add"
        >
          <template v-slot:item="{ item }">
            <v-list-item-avatar size="30"><UserAvatar :user="item" :size="30" /></v-list-item-avatar>
            <v-list-item-content>
              <v-list-item-title>{{ item.display_name || item.username }}</v-list-item-title>
              <v-list-item-subtitle>@{{ item.username }}</v-list-item-subtitle>
            </v-list-item-content>
            <v-list-item-action><v-icon>mdi-account-plus</v-icon></v-list-item-action>
          </template>
        </v-autocomplete>
      </v-card-text>
    </v-card>

    <v-card v-if="incoming.length" class="mb-6">
      <v-card-title class="subtitle-1 font-weight-bold">Friend requests</v-card-title>
      <v-list>
        <v-list-item v-for="f in incoming" :key="'in' + f.user.id">
          <v-list-item-avatar><UserAvatar :user="f.user" :size="40" /></v-list-item-avatar>
          <v-list-item-content>
            <v-list-item-title>{{ f.user.display_name || f.user.username }}</v-list-item-title>
            <v-list-item-subtitle>@{{ f.user.username }} wants to be your friend</v-list-item-subtitle>
          </v-list-item-content>
          <v-list-item-action class="flex-row">
            <v-btn small color="primary" depressed class="mr-2" @click="accept(f)">Accept</v-btn>
            <v-btn small text @click="remove(f)">Decline</v-btn>
          </v-list-item-action>
        </v-list-item>
      </v-list>
    </v-card>

    <v-card class="mb-6">
      <v-card-title class="subtitle-1 font-weight-bold">Your friends <span class="ml-2 text--secondary">{{ accepted.length }}</span></v-card-title>
      <v-list v-if="accepted.length">
        <v-list-item v-for="f in accepted" :key="'ac' + f.user.id">
          <v-list-item-avatar><UserAvatar :user="f.user" :size="40" /></v-list-item-avatar>
          <v-list-item-content>
            <v-list-item-title>{{ f.user.display_name || f.user.username }}</v-list-item-title>
            <v-list-item-subtitle>@{{ f.user.username }} · friends since {{ f.created_at | datetime }}</v-list-item-subtitle>
          </v-list-item-content>
          <v-list-item-action>
            <v-btn icon @click="remove(f)" title="Remove friend"><v-icon>mdi-account-remove-outline</v-icon></v-btn>
          </v-list-item-action>
        </v-list-item>
      </v-list>
      <v-card-text v-else class="text--secondary">Search for people above to add friends. Friends see your friends-only rooms and can join them.</v-card-text>
    </v-card>

    <v-card v-if="outgoing.length">
      <v-card-title class="subtitle-1 font-weight-bold">Sent requests</v-card-title>
      <v-list>
        <v-list-item v-for="f in outgoing" :key="'out' + f.user.id">
          <v-list-item-avatar><UserAvatar :user="f.user" :size="40" /></v-list-item-avatar>
          <v-list-item-content>
            <v-list-item-title>{{ f.user.display_name || f.user.username }}</v-list-item-title>
            <v-list-item-subtitle>pending</v-list-item-subtitle>
          </v-list-item-content>
          <v-list-item-action>
            <v-btn small text @click="remove(f)">Cancel</v-btn>
          </v-list-item-action>
        </v-list-item>
      </v-list>
    </v-card>
  </v-container>
</template>

<script lang="ts">
import { Vue, Component, Watch } from 'vue-property-decorator'
import { api, errorMessage, Friend, PublicUser } from '@/api-ext'
import UserAvatar from '@/components/community/UserAvatar.vue'

@Component({
  components: {
    UserAvatar,
  }
})
export default class FriendsPage extends Vue {
  query = ''
  selected: PublicUser | null = null
  results: PublicUser[] = []
  searching = false
  private searchTimer = 0

  get friends(): Friend[] {
    return this.$store.state.app.friends
  }

  get incoming() {
    return this.friends.filter(f => f.status === 'pending' && f.direction === 'incoming')
  }

  get outgoing() {
    return this.friends.filter(f => f.status === 'pending' && f.direction === 'outgoing')
  }

  get accepted() {
    return this.friends.filter(f => f.status === 'accepted')
  }

  @Watch('query')
  onQuery(q: string) {
    window.clearTimeout(this.searchTimer)
    if (!q || q.length < 2) {
      this.results = []
      return
    }
    this.searchTimer = window.setTimeout(async () => {
      this.searching = true
      try {
        const me = this.$store.getters.user
        this.results = (await api.searchUsers(q)).filter(u => !me || u.id !== me.id)
      } finally {
        this.searching = false
      }
    }, 250)
  }

  async refresh() {
    await this.$store.dispatch('APP_FRIENDS')
  }

  async add(user: PublicUser | null) {
    if (!user) return
    try {
      const res = await api.addFriend(user.username)
      this.$swal({ toast: true, position: 'bottom-end', timer: 2500, showConfirmButton: false, icon: 'success', title: res.status === 'accepted' ? 'You are now friends!' : 'Friend request sent' })
      await this.refresh()
    } catch (e) {
      this.$swal({ icon: 'error', title: 'Unable to add friend', text: errorMessage(e) })
    } finally {
      this.$nextTick(() => {
        this.selected = null
        this.query = ''
      })
    }
  }

  async accept(f: Friend) {
    try {
      await api.acceptFriend(f.user.id)
      await this.refresh()
    } catch (e) {
      this.$swal({ icon: 'error', title: 'Error', text: errorMessage(e) })
    }
  }

  async remove(f: Friend) {
    if (f.status === 'accepted') {
      const { value } = await this.$swal({ title: `Remove ${f.user.username}?`, icon: 'warning', showCancelButton: true, confirmButtonText: 'Remove' })
      if (!value) return
    }
    try {
      await api.removeFriend(f.user.id)
      await this.refresh()
    } catch (e) {
      this.$swal({ icon: 'error', title: 'Error', text: errorMessage(e) })
    }
  }

  mounted() {
    this.refresh()
  }
}
</script>
