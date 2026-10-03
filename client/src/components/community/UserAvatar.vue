<template>
  <span class="nr-avatar-wrap">
  <v-tooltip bottom :disabled="!tooltip">
    <template v-slot:activator="{ on, attrs }">
      <v-avatar
        v-bind="attrs"
        v-on="on"
        :size="size"
        :style="{ background: color, border: ring ? '2px solid var(--nr-surface)' : undefined }"
        class="nr-avatar"
      >
        <span class="white--text font-weight-bold" :style="{ fontSize: Math.round(size * 0.42) + 'px' }">{{ initials }}</span>
      </v-avatar>
    </template>
    <span>{{ label }}</span>
  </v-tooltip>
  </span>
</template>

<style scoped>
.nr-avatar-wrap { display: inline-flex; flex-shrink: 0; vertical-align: middle; }
.nr-avatar { user-select: none; }
</style>

<script lang="ts">
import { Vue, Component, Prop } from 'vue-property-decorator'

interface Named {
  username?: string
  display_name?: string
}

// stable, pleasant colors derived from the name
const COLORS = ['#8c5cff', '#ff5ca8', '#3ec5ff', '#2ee59d', '#ffb547', '#ff6b6b', '#5c7cff', '#00c2a8', '#e056fd', '#f78fb3']

@Component
export default class UserAvatar extends Vue {
  @Prop() readonly user!: Named | null
  @Prop(String) readonly name!: string
  @Prop({ type: Number, default: 32 }) readonly size!: number
  @Prop(Boolean) readonly tooltip!: boolean
  @Prop(Boolean) readonly ring!: boolean

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
    return COLORS[h % COLORS.length]
  }
}
</script>
