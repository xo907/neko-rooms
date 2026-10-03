<template>
  <div v-if="policy">
    <v-row>
      <v-col cols="12" md="6">
        <v-card class="mb-6">
          <v-card-title class="subtitle-1 font-weight-bold"><v-icon left small>mdi-home-outline</v-icon>Homepage & community</v-card-title>
          <v-card-text>
            <v-switch v-model="policy.homepage_enabled" inset dense label="Enable the public homepage with the room directory" />
            <v-switch v-model="policy.guests_can_browse" inset dense label="Signed-out visitors can browse public rooms" />
            <v-switch v-model="policy.rooms_require_login" inset dense label="Require signing in to join any room" hint="Also blocks direct room links for signed-out visitors" persistent-hint />
            <v-switch v-model="policy.friends_enabled" inset dense label="Enable friends" />
            <v-switch v-model="policy.thumbnails_enabled" inset dense label="Show live room thumbnails" />
            <v-switch v-model="policy.show_member_names" inset dense label="Show who is inside rooms" />
          </v-card-text>
        </v-card>

        <v-card class="mb-6">
          <v-card-title class="subtitle-1 font-weight-bold"><v-icon left small>mdi-account-plus-outline</v-icon>Registration</v-card-title>
          <v-card-text>
            <v-switch v-model="policy.registration_enabled" inset dense label="Allow people to sign up" />
            <v-switch v-model="policy.registration_approval" inset dense :disabled="!policy.registration_enabled" label="New accounts need admin approval" />
            <v-text-field v-model.number="policy.password_min_length" type="number" min="6" max="128" label="Minimum password length" outlined dense class="mt-4" />
          </v-card-text>
        </v-card>

        <v-card class="mb-6">
          <v-card-title class="subtitle-1 font-weight-bold"><v-icon left small>mdi-shield-lock-outline</v-icon>Security</v-card-title>
          <v-card-text>
            <v-text-field v-model.number="policy.session_ttl_hours" type="number" min="1" label="Session lifetime (hours)" hint="Sessions are extended while in use" persistent-hint outlined dense class="mb-4" />
            <v-text-field v-model.number="policy.audit_retention" type="number" min="100" label="Audit log entries to keep" outlined dense />
          </v-card-text>
        </v-card>
      </v-col>

      <v-col cols="12" md="6">
        <v-card class="mb-6">
          <v-card-title class="subtitle-1 font-weight-bold"><v-icon left small>mdi-account-outline</v-icon>Regular users</v-card-title>
          <v-card-text>
            <v-switch v-model="policy.users_can_create_rooms" inset dense label="Can create rooms" />
            <v-switch v-model="policy.users_can_make_public" inset dense label="Can make rooms public" />
            <v-switch v-model="policy.users_can_pull_images" inset dense label="Can pull neko images" />
            <v-switch v-model="policy.users_can_use_mounts" inset dense label="Can use public/protected host mounts" hint="Private and template storage is always allowed" persistent-hint />
            <v-switch v-model="policy.users_can_set_devices" inset dense label="Can attach devices & GPUs" />

            <v-row class="mt-4" dense>
              <v-col cols="6">
                <v-text-field v-model.number="policy.default_room_limit" type="number" min="0" label="Rooms per user" hint="0 = unlimited, can be overridden per user" persistent-hint outlined dense />
              </v-col>
              <v-col cols="6">
                <v-text-field v-model.number="policy.user_max_connections" type="number" min="0" label="Max connections per room" hint="0 = unlimited" persistent-hint outlined dense />
              </v-col>
            </v-row>

            <v-select
              v-model="policy.user_neko_images"
              :items="images"
              label="Images users may use"
              hint="Leave empty to allow all configured images"
              persistent-hint
              multiple
              chips
              small-chips
              deletable-chips
              outlined
              dense
              class="mt-6"
            />
          </v-card-text>
        </v-card>
      </v-col>
    </v-row>

    <SaveBar v-if="dirty">
      <span class="mr-4">You have unsaved changes</span>
      <v-btn text @click="load">Discard</v-btn>
      <v-btn color="primary" depressed :loading="saving" @click="save">Save settings</v-btn>
    </SaveBar>
  </div>
</template>

<script lang="ts">
import { Vue, Component } from 'vue-property-decorator'
import { api, errorMessage, Policy } from '@/api-ext'
import SaveBar from '@/components/admin/SaveBar.vue'

@Component({
  components: {
    SaveBar,
  }
})
export default class AdminSettings extends Vue {
  policy: Policy | null = null
  original = ''
  saving = false

  get dirty() {
    return this.policy !== null && JSON.stringify(this.policy) !== this.original
  }

  get images(): string[] {
    return this.$store.state.roomsConfig.neko_images || []
  }

  async load() {
    this.policy = await api.policy()
    this.original = JSON.stringify(this.policy)
  }

  async save() {
    if (!this.policy) return
    this.saving = true
    try {
      this.policy = await api.updatePolicy({
        ...this.policy,
        // eslint-disable-next-line
        password_min_length: Number(this.policy.password_min_length),
        // eslint-disable-next-line
        session_ttl_hours: Number(this.policy.session_ttl_hours),
        // eslint-disable-next-line
        audit_retention: Number(this.policy.audit_retention),
        // eslint-disable-next-line
        default_room_limit: Number(this.policy.default_room_limit),
        // eslint-disable-next-line
        user_max_connections: Number(this.policy.user_max_connections),
      })
      this.original = JSON.stringify(this.policy)
      await this.$store.dispatch('APP_STATUS')
      this.$swal({ toast: true, position: 'bottom-end', timer: 2000, showConfirmButton: false, icon: 'success', title: 'Settings saved' })
    } catch (e) {
      this.$swal({ icon: 'error', title: 'Unable to save', text: errorMessage(e) })
    } finally {
      this.saving = false
    }
  }

  mounted() {
    this.load()
    this.$store.dispatch('ROOMS_CONFIG').catch(() => { /* ignore */ })
  }
}
</script>
