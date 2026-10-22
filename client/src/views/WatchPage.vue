<template>
  <!-- the room itself is shown by the player in App.vue -->
  <div class="nr-watch" />
</template>

<style scoped>
.nr-watch { min-height: 100vh; }
</style>

<script lang="ts">
import { Vue, Component, Prop, Watch } from 'vue-property-decorator'
import { isPlaying, openPlayer } from '@/player/state'
import { resolveJoinUrl } from '@/utils/join'

// WatchPage joins the room when it was opened directly, e.g. after a reload.
@Component
export default class WatchPage extends Vue {
  @Prop({ required: true }) readonly name!: string

  get invite(): string {
    return String(this.$route.query.invite || '')
  }

  @Watch('name', { immediate: true })
  async onName() {
    if (isPlaying(this.name)) return

    const url = await resolveJoinUrl(this, this.name, this.invite || undefined)
    if (!url) {
      this.$router.replace({ name: 'room', params: { name: this.name }, query: this.invite ? { invite: this.invite } : {} }).catch(() => { /* ignore */ })
      return
    }

    openPlayer({
      name: this.name,
      invite: this.invite,
      url,
      returnTo: { name: 'room', params: { name: this.name }, query: this.invite ? { invite: this.invite } : {} },
    })
  }
}
</script>
