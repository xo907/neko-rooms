<template>
  <span class="nr-avatar-wrap">
  <v-tooltip bottom :disabled="!tooltip">
    <template v-slot:activator="{ on, attrs }">
      <v-avatar
        v-bind="attrs"
        v-on="on"
        :size="size"
        :style="{ background: color, boxShadow: ring ? '0 0 0 2px var(--nr-bg)' : 'inset 0 0 0 1px rgba(255, 255, 255, .12)' }"
        class="nr-avatar"
      >
        <img v-if="image" :src="image" :alt="label" class="nr-photo" @error="failed = image">
        <span v-else class="nr-initials" :style="{ fontSize: Math.round(size * 0.42) + 'px' }">{{ initials }}</span>
      </v-avatar>
    </template>
    <span>{{ label }}</span>
  </v-tooltip>
  </span>
</template>

<style scoped>
.nr-avatar-wrap { display: inline-flex; flex-shrink: 0; vertical-align: middle; }
.nr-avatar { user-select: none; }
.nr-photo { width: 100%; height: 100%; object-fit: cover; }
.nr-initials { color: rgba(255, 255, 255, .95); font-weight: 600; letter-spacing: -0.02em; }
</style>

<script lang="ts">
import { Vue, Component, Prop } from 'vue-property-decorator'
import { api } from '@/api-ext'

interface Named {
  id?: number
  avatar?: number
  username?: string
  display_name?: string
}

// stable, muted colors derived from the name
const COLORS = ['#0f766e', '#0369a1', '#6d28d9', '#be185d', '#b45309', '#be123c', '#15803d', '#4338ca', '#a21caf', '#3f3f46']

@Component
export default class UserAvatar extends Vue {
  @Prop() readonly user!: Named | null
  @Prop(String) readonly name!: string
  @Prop({ type: Number, default: 32 }) readonly size!: number
  @Prop(Boolean) readonly tooltip!: boolean
  @Prop(Boolean) readonly ring!: boolean
  // avatar path relative to the api root, for names without a user (room members)
  @Prop(String) readonly src!: string

  failed = ''

  get image(): string {
    let url = ''
    if (this.src) url = api.apiUrl(this.src)
    else if (this.user && this.user.id && this.user.avatar) url = api.avatarUrl({ id: this.user.id, avatar: this.user.avatar })
    return url !== this.failed ? url : ''
  }

  get label(): string {
    if (this.name) return this.name
    if (!this.user) return '?'
    return this.user.display_name || this.user.username || '?'
  }

  get initials(): string {
    const parts = this.label.trim().split(/[\s._-]+/).filter(Boolean)
    if (parts.length === 0) return '?'
    if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase()
    return (parts[0][0] + parts[1][0]).toUpperCase()
  }

  get color(): string {
    const key = ((this.user && this.user.username) || this.label).toLowerCase()
    let h = 0
    for (let i = 0; i < key.length; i++) h = (h * 31 + key.charCodeAt(i)) >>> 0
    const c = COLORS[h % COLORS.length]
    return `linear-gradient(135deg, ${c}, color-mix(in srgb, ${c} 55%, #000))`
  }
}
</script>
