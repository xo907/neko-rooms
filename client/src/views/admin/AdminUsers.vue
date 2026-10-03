<template>
  <div>
    <div class="d-flex flex-wrap align-center mb-4" style="gap: 12px">
      <v-text-field v-model="search" dense solo flat hide-details clearable prepend-inner-icon="mdi-magnify" placeholder="Search users" style="max-width: 360px" />
      <v-btn-toggle v-model="filter" dense mandatory>
        <v-btn value="all" small>All</v-btn>
        <v-btn value="admin" small>Admins</v-btn>
        <v-btn value="disabled" small>
          Disabled
          <v-chip v-if="disabledCount" x-small color="warning" class="ml-1">{{ disabledCount }}</v-chip>
        </v-btn>
      </v-btn-toggle>
      <v-spacer />
      <v-btn color="primary" depressed @click="openCreate"><v-icon left>mdi-account-plus</v-icon>Add user</v-btn>
    </div>

    <v-card>
      <v-data-table :headers="headers" :items="filtered" :search="search" :loading="loading" :items-per-page="25" :footer-props="{ itemsPerPageOptions: [10, 25, 50, 100, -1] }">
        <template v-slot:[`item.username`]="{ item }">
          <div class="d-flex align-center py-2">
            <UserAvatar :user="item" :size="34" class="mr-3" />
            <div>
              <div class="font-weight-bold">{{ item.display_name || item.username }}</div>
              <div class="caption text--secondary">@{{ item.username }}<span v-if="item.email"> · {{ item.email }}</span></div>
            </div>
          </div>
        </template>
        <template v-slot:[`item.role`]="{ item }">
          <v-chip x-small label :color="item.role === 'admin' ? 'accent' : undefined">{{ item.role }}</v-chip>
        </template>
        <template v-slot:[`item.disabled`]="{ item }">
          <v-chip v-if="item.disabled" x-small label color="warning">disabled</v-chip>
          <v-chip v-else x-small label color="success" outlined>active</v-chip>
        </template>
        <template v-slot:[`item.room_count`]="{ item }">
          {{ item.room_count }}<span class="text--secondary"> / {{ limitLabel(item) }}</span>
        </template>
        <template v-slot:[`item.last_login_at`]="{ item }">
          <span v-if="item.last_login_at">{{ item.last_login_at | timeago }}</span>
          <span v-else class="text--secondary">never</span>
        </template>
        <template v-slot:[`item.created_at`]="{ item }">
          <span :title="item.created_at">{{ item.created_at | datetime }}</span>
        </template>
        <template v-slot:[`item.actions`]="{ item }">
          <div class="d-flex justify-end">
            <v-btn v-if="item.disabled" small text color="success" @click="setDisabled(item, false)">Approve</v-btn>
            <v-btn icon @click="openEdit(item)" title="Edit"><v-icon>mdi-pencil-outline</v-icon></v-btn>
            <v-menu offset-y left>
              <template v-slot:activator="{ on, attrs }">
                <v-btn icon v-bind="attrs" v-on="on"><v-icon>mdi-dots-vertical</v-icon></v-btn>
              </template>
              <v-list dense>
                <v-list-item @click="setDisabled(item, !item.disabled)" :disabled="item.id === me.id">
                  <v-list-item-icon><v-icon>{{ item.disabled ? 'mdi-account-check' : 'mdi-account-cancel' }}</v-icon></v-list-item-icon>
                  <v-list-item-title>{{ item.disabled ? 'Enable account' : 'Disable account' }}</v-list-item-title>
                </v-list-item>
                <v-list-item @click="setRole(item, item.role === 'admin' ? 'user' : 'admin')" :disabled="item.id === me.id">
                  <v-list-item-icon><v-icon>mdi-shield-crown-outline</v-icon></v-list-item-icon>
                  <v-list-item-title>{{ item.role === 'admin' ? 'Make regular user' : 'Make admin' }}</v-list-item-title>
                </v-list-item>
                <v-list-item @click="revokeSessions(item)">
                  <v-list-item-icon><v-icon>mdi-logout-variant</v-icon></v-list-item-icon>
                  <v-list-item-title>Sign out everywhere</v-list-item-title>
                </v-list-item>
                <v-divider />
                <v-list-item @click="remove(item)" :disabled="item.id === me.id">
                  <v-list-item-icon><v-icon color="error">mdi-delete-outline</v-icon></v-list-item-icon>
                  <v-list-item-title class="error--text">Delete user</v-list-item-title>
                </v-list-item>
              </v-list>
            </v-menu>
          </div>
        </template>
      </v-data-table>
    </v-card>

    <v-dialog v-model="dialog" max-width="520" persistent>
      <v-card>
        <v-card-title>{{ editing ? 'Edit ' + editing.username : 'Add user' }}</v-card-title>
        <v-card-text>
          <v-alert v-if="formError" type="error" dense text>{{ formError }}</v-alert>
          <v-text-field v-model="form.username" label="Username" outlined dense />
          <v-text-field v-model="form.display_name" label="Display name" outlined dense />
          <v-text-field v-model="form.email" label="Email" outlined dense />
          <v-text-field
            v-model="form.password"
            :label="editing ? 'New password (leave empty to keep)' : 'Password'"
            outlined
            dense
            :type="showPass ? 'text' : 'password'"
            :append-icon="showPass ? 'mdi-eye' : 'mdi-eye-off'"
            @click:append="showPass = !showPass"
            autocomplete="new-password"
          >
            <template v-slot:append-outer>
              <v-btn icon small @click="generate" title="Generate password"><v-icon small>mdi-dice-5-outline</v-icon></v-btn>
            </template>
          </v-text-field>
          <v-row dense>
            <v-col cols="6">
              <v-select v-model="form.role" :items="['user', 'admin']" label="Role" outlined dense :disabled="editing && editing.id === me.id" />
            </v-col>
            <v-col cols="6">
              <v-text-field
                v-model.number="form.room_limit"
                type="number"
                min="-1"
                label="Room limit"
                hint="-1 = default, 0 = unlimited"
                persistent-hint
                outlined
                dense
              />
            </v-col>
          </v-row>
          <v-switch v-model="form.disabled" label="Account disabled" inset hide-details :disabled="editing && editing.id === me.id" />
        </v-card-text>
        <v-card-actions>
          <v-spacer />
          <v-btn text @click="dialog = false">Cancel</v-btn>
          <v-btn color="primary" depressed :loading="saving" @click="save">{{ editing ? 'Save' : 'Create' }}</v-btn>
        </v-card-actions>
      </v-card>
    </v-dialog>
  </div>
</template>

<script lang="ts">
import { Vue, Component } from 'vue-property-decorator'
import { api, errorMessage, Role, User } from '@/api-ext'
import { randomPassword } from '@/utils/random'
import UserAvatar from '@/components/community/UserAvatar.vue'

@Component({
  components: {
    UserAvatar,
  }
})
export default class AdminUsers extends Vue {
  users: User[] = []
  loading = false
  search = ''
  filter = 'all'
  defaultLimit = 0

  dialog = false
  editing: User | null = null
  saving = false
  showPass = false
  formError = ''
  form = this.emptyForm()

  headers = [
    { text: 'User', value: 'username' },
    { text: 'Role', value: 'role' },
    { text: 'Status', value: 'disabled' },
    { text: 'Rooms', value: 'room_count' },
    { text: 'Last login', value: 'last_login_at' },
    { text: 'Joined', value: 'created_at' },
    { text: '', value: 'actions', sortable: false, align: 'end' },
  ]

  get me(): User {
    return this.$store.getters.user
  }

  get filtered(): User[] {
    if (this.filter === 'admin') return this.users.filter(u => u.role === 'admin')
    if (this.filter === 'disabled') return this.users.filter(u => u.disabled)
    return this.users
  }

  get disabledCount(): number {
    return this.users.filter(u => u.disabled).length
  }

  emptyForm() {
    // eslint-disable-next-line
    return { username: '', display_name: '', email: '', password: '', role: 'user' as Role, room_limit: -1, disabled: false }
  }

  limitLabel(u: User) {
    if (u.role === 'admin') return '∞'
    const limit = u.room_limit >= 0 ? u.room_limit : this.defaultLimit
    return limit === 0 ? '∞' : String(limit)
  }

  async load() {
    this.loading = true
    try {
      const [users, policy] = await Promise.all([api.users(), api.policy()])
      this.users = users
      this.defaultLimit = policy.default_room_limit
    } finally {
      this.loading = false
    }
  }

  openCreate() {
    this.editing = null
    this.form = this.emptyForm()
    this.formError = ''
    this.dialog = true
  }

  openEdit(u: User) {
    this.editing = u
    // eslint-disable-next-line
    this.form = { username: u.username, display_name: u.display_name, email: u.email, password: '', role: u.role, room_limit: u.room_limit, disabled: u.disabled }
    this.formError = ''
    this.dialog = true
  }

  generate() {
    this.form.password = randomPassword()
    this.showPass = true
  }

  async save() {
    this.saving = true
    this.formError = ''
    try {
      const data = { ...this.form, room_limit: Number(this.form.room_limit) }
      if (this.editing) {
        await api.updateUser(this.editing.id, data)
      } else {
        await api.createUser(data)
      }
      this.dialog = false
      await this.load()
    } catch (e) {
      this.formError = errorMessage(e)
    } finally {
      this.saving = false
    }
  }

  async setDisabled(u: User, disabled: boolean) {
    try {
      await api.updateUser(u.id, { disabled })
      await this.load()
    } catch (e) {
      this.$swal({ icon: 'error', title: 'Error', text: errorMessage(e) })
    }
  }

  async setRole(u: User, role: Role) {
    try {
      await api.updateUser(u.id, { role })
      await this.load()
    } catch (e) {
      this.$swal({ icon: 'error', title: 'Error', text: errorMessage(e) })
    }
  }

  async revokeSessions(u: User) {
    try {
      await api.revokeUserSessions(u.id)
      this.$swal({ toast: true, position: 'bottom-end', timer: 2000, showConfirmButton: false, icon: 'success', title: `${u.username} was signed out` })
    } catch (e) {
      this.$swal({ icon: 'error', title: 'Error', text: errorMessage(e) })
    }
  }

  async remove(u: User) {
    const { value } = await this.$swal({
      title: `Delete ${u.username}?`,
      text: 'Their rooms stay, but will have no owner. This can not be undone.',
      icon: 'warning',
      showCancelButton: true,
      confirmButtonText: 'Delete',
    })
    if (!value) return
    try {
      await api.deleteUser(u.id)
      await this.load()
    } catch (e) {
      this.$swal({ icon: 'error', title: 'Error', text: errorMessage(e) })
    }
  }

  mounted() {
    this.load()
  }
}
</script>
