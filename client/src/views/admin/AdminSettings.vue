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

        <v-card v-if="integrations" class="mb-6">
          <v-card-title class="subtitle-1 font-weight-bold"><v-icon left small>mdi-robot-outline</v-icon>Room automation</v-card-title>
          <v-card-text>
            <v-switch v-model="integrations.now_playing" inset dense label="Show what is playing in each room" hint="The title of the page open in the room's browser" persistent-hint />
            <v-switch v-model="sleepEnabled" inset dense label="Put empty rooms to sleep" hint="Frees memory and CPU. A sleeping room starts again when someone joins it." persistent-hint />
            <v-text-field
              v-if="sleepEnabled"
              v-model.number="integrations.auto_sleep_minutes"
              type="number"
              min="5"
              max="1440"
              label="Minutes a room may stay empty"
              outlined
              dense
              class="mt-6"
              hide-details
            />
          </v-card-text>
        </v-card>

        <v-card v-if="integrations" class="mb-6">
          <v-card-title class="subtitle-1 font-weight-bold"><v-icon left small>mdi-bell-ring-outline</v-icon>Discord</v-card-title>
          <v-card-text>
            <p class="caption text--secondary mt-0">
              Post public rooms to a Discord channel: in the channel settings open Integrations, Webhooks, New Webhook and copy its URL.
            </p>
            <v-text-field v-model="integrations.discord_webhook" label="Webhook URL" placeholder="https://discord.com/api/webhooks/…" outlined dense clearable />
            <v-text-field
              v-model="integrations.public_url"
              label="Site address"
              :placeholder="siteAddress"
              hint="Used for links and images in posts. Leave empty to use the address admins open the site on."
              persistent-hint
              outlined
              dense
              clearable
            />
            <v-switch v-model="integrations.notify_live" inset dense hide-details label="A room goes live" class="mt-4" />
            <v-switch v-model="integrations.notify_watching" inset dense hide-details label="People start watching an empty room" />
            <v-switch v-model="integrations.notify_schedule" inset dense hide-details label="Watch parties are planned and start" />
            <div class="mt-5">
              <v-btn small outlined :disabled="!integrations.discord_webhook" :loading="testing" @click="test">
                <v-icon left small>mdi-send-outline</v-icon>Send a test post
              </v-btn>
            </div>
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
import { api, errorMessage, Integrations, Policy } from '@/api-ext'
import SaveBar from '@/components/admin/SaveBar.vue'

@Component({
  components: {
    SaveBar,
  }
})
export default class AdminSettings extends Vue {
  policy: Policy | null = null
  integrations: Integrations | null = null
  original = ''
  originalIntegrations = ''
  saving = false
  testing = false

  get dirty() {
    return (this.policy !== null && JSON.stringify(this.policy) !== this.original) ||
      (this.integrations !== null && JSON.stringify(this.integrations) !== this.originalIntegrations)
  }

  get sleepEnabled(): boolean {
    return !!this.integrations && this.integrations.auto_sleep_minutes > 0
  }

  set sleepEnabled(on: boolean) {
    // eslint-disable-next-line
    if (this.integrations) this.integrations.auto_sleep_minutes = on ? 30 : 0
  }

  get siteAddress(): string {
    return location.href.split(/[?#]/)[0].replace(/[^/]*$/, '')
  }

  integrationsPayload(): Integrations {
    const i = this.integrations as Integrations
    // eslint-disable-next-line
    return { ...i, public_url: i.public_url || '', discord_webhook: i.discord_webhook || '', auto_sleep_minutes: Number(i.auto_sleep_minutes) || 0 }
  }

  async test() {
    if (!this.integrations) return
    this.testing = true
    try {
      await api.testIntegrations(this.integrationsPayload())
      this.$swal({ toast: true, position: 'bottom-end', timer: 2500, showConfirmButton: false, icon: 'success', title: 'Posted to Discord' })
    } catch (e) {
      this.$swal({ icon: 'error', title: 'Discord did not take it', text: errorMessage(e) })
    } finally {
      this.testing = false
    }
  }

  get images(): string[] {
    return this.$store.state.roomsConfig.neko_images || []
  }

  async load() {
    this.policy = await api.policy()
    this.original = JSON.stringify(this.policy)
    this.integrations = await api.integrations()
    this.originalIntegrations = JSON.stringify(this.integrations)
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
      if (this.integrations) {
        this.integrations = await api.updateIntegrations(this.integrationsPayload())
        this.originalIntegrations = JSON.stringify(this.integrations)
      }
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
