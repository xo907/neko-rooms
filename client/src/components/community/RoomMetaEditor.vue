<template>
  <v-form ref="form" @submit.prevent="save">
    <v-text-field v-model="data.title" label="Title" :placeholder="name" counter="80" maxlength="80" outlined dense />
    <v-textarea v-model="data.description" label="Description" counter="500" maxlength="500" rows="2" auto-grow outlined dense />
    <v-combobox v-model="data.category" :items="categories" label="Category" outlined dense clearable />

    <div class="subtitle-2 mb-2">Who can find and join this room?</div>
    <v-radio-group v-model="data.visibility" class="mt-0" hide-details>
      <v-radio value="public" :disabled="!canMakePublic">
        <template v-slot:label>
          <div>
            <div><v-icon small class="mr-1">mdi-earth</v-icon><strong>Public</strong></div>
            <div class="caption text--secondary">Listed on the homepage, anyone can join.<span v-if="!canMakePublic"> Not allowed for your account.</span></div>
          </div>
        </template>
      </v-radio>
      <v-radio value="friends" :disabled="!friendsEnabled">
        <template v-slot:label>
          <div>
            <div><v-icon small class="mr-1">mdi-account-multiple</v-icon><strong>Friends</strong></div>
            <div class="caption text--secondary">Only your friends see it and can join.</div>
          </div>
        </template>
      </v-radio>
      <v-radio value="private">
        <template v-slot:label>
          <div>
            <div><v-icon small class="mr-1">mdi-lock</v-icon><strong>Private</strong></div>
            <div class="caption text--secondary">Hidden. Only people with the invite link can join.</div>
          </div>
        </template>
      </v-radio>
    </v-radio-group>

    <template v-if="isAdmin && moderation">
      <v-divider class="my-4" />
      <div class="subtitle-2 mb-2"><v-icon small class="mr-1">mdi-shield-crown-outline</v-icon>Moderation</div>
      <v-switch v-model="data.featured" inset dense hide-details label="Featured on the homepage" class="mt-0" />
      <v-switch v-model="data.hidden" inset dense hide-details label="Hidden from listings (moderated)" class="mb-4" />
      <v-autocomplete
        v-model="data.owner_id"
        :items="userItems"
        label="Owner"
        outlined
        dense
        hide-details
        clearable
        :loading="usersLoading"
      />
    </template>

    <div class="d-flex mt-4">
      <v-spacer />
      <v-btn text @click="$emit('cancel')" v-if="cancelable">Cancel</v-btn>
      <v-btn color="primary" depressed type="submit" :loading="saving">Save</v-btn>
    </div>
  </v-form>
</template>

<script lang="ts">
import { Vue, Component, Prop, Watch } from 'vue-property-decorator'
import { api, errorMessage, RoomMeta, RoomMetaUpdate, User, Visibility } from '@/api-ext'

@Component
export default class RoomMetaEditor extends Vue {
  @Prop({ required: true }) readonly roomId!: string
  @Prop({ required: true }) readonly name!: string
  @Prop() readonly meta!: RoomMeta | null
  @Prop(Boolean) readonly moderation!: boolean
  @Prop(Boolean) readonly cancelable!: boolean

  data = {
    title: '',
    description: '',
    category: '' as string | null,
    visibility: 'private' as Visibility,
    featured: false,
    hidden: false,
    // eslint-disable-next-line
    owner_id: null as number | null,
  }

  saving = false
  users: User[] = []
  usersLoading = false

  get isAdmin(): boolean {
    return this.$store.getters.isAdmin
  }

  get canMakePublic(): boolean {
    const p = this.$store.getters.policy
    return this.isAdmin || (p && p.users_can_make_public)
  }

  get friendsEnabled(): boolean {
    const p = this.$store.getters.policy
    return p && p.friends_enabled
  }

  get categories(): string[] {
    return this.$store.state.app.branding.home.categories || []
  }

  get userItems() {
    return this.users.map(u => ({ text: `${u.display_name || u.username} (@${u.username})`, value: u.id }))
  }

  @Watch('meta', { immediate: true })
  onMeta() {
    const m = this.meta
    this.data = {
      title: m ? m.title : '',
      description: m ? m.description : '',
      category: m ? m.category : '',
      visibility: m ? m.visibility : 'private',
      featured: m ? m.featured : false,
      hidden: m ? m.hidden : false,
      // eslint-disable-next-line
      owner_id: m ? m.owner_id : null,
    }
  }

  async mounted() {
    if (this.isAdmin && this.moderation) {
      this.usersLoading = true
      try {
        this.users = await api.users()
      } finally {
        this.usersLoading = false
      }
    }
  }

  async save() {
    this.saving = true
    try {
      const update: RoomMetaUpdate = {
        title: this.data.title,
        description: this.data.description,
        category: this.data.category || '',
        visibility: this.data.visibility,
      }
      if (this.isAdmin && this.moderation) {
        update.featured = this.data.featured
        update.hidden = this.data.hidden
        // eslint-disable-next-line
        update.owner_id = this.data.owner_id || 0
      }
      const meta = await api.updateRoomMeta(this.roomId, update)
      this.$emit('saved', meta)
      this.$swal({ toast: true, position: 'bottom-end', timer: 2000, showConfirmButton: false, icon: 'success', title: 'Room updated' })
    } catch (e) {
      this.$swal({ icon: 'error', title: 'Unable to save', text: errorMessage(e) })
    } finally {
      this.saving = false
    }
  }
}
</script>
