<template>
  <v-card>
    <v-data-table
      :headers="headers"
      :items="entries"
      :server-items-length="total"
      :options.sync="options"
      :loading="loading"
      :footer-props="{ itemsPerPageOptions: [25, 50, 100, 200] }"
      dense
    >
      <template v-slot:[`item.created_at`]="{ item }">
        <span :title="item.created_at">{{ item.created_at | datetime }}</span>
      </template>
      <template v-slot:[`item.username`]="{ item }">
        <strong v-if="item.username">{{ item.username }}</strong>
        <span v-else class="text--secondary">anonymous</span>
      </template>
      <template v-slot:[`item.action`]="{ item }">
        <v-chip x-small label :color="color(item.action)" dark>{{ item.action }}</v-chip>
      </template>
    </v-data-table>
  </v-card>
</template>

<script lang="ts">
import { Vue, Component, Watch } from 'vue-property-decorator'
import { api, AuditEntry } from '@/api-ext'

@Component
export default class AdminAudit extends Vue {
  entries: AuditEntry[] = []
  total = 0
  loading = false
  options = { page: 1, itemsPerPage: 50 }

  headers = [
    { text: 'Time', value: 'created_at', sortable: false },
    { text: 'User', value: 'username', sortable: false },
    { text: 'Action', value: 'action', sortable: false },
    { text: 'Target', value: 'target', sortable: false },
    { text: 'Details', value: 'details', sortable: false },
    { text: 'IP', value: 'ip', sortable: false },
  ]

  color(action: string) {
    if (action.includes('failed') || action.includes('delete') || action.includes('remove')) return 'error'
    if (action.startsWith('auth.')) return 'info'
    if (action.startsWith('branding') || action.startsWith('settings')) return 'accent'
    if (action.startsWith('user')) return 'primary'
    return 'grey darken-1'
  }

  @Watch('options', { deep: true, immediate: true })
  async load() {
    this.loading = true
    try {
      const { page, itemsPerPage } = this.options
      const res = await api.audit(itemsPerPage, (page - 1) * itemsPerPage)
      this.entries = res.entries
      this.total = res.total
    } finally {
      this.loading = false
    }
  }
}
</script>
