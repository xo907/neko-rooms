<template>
  <div>
    <v-card class="mb-6">
      <v-card-title class="subtitle-1 font-weight-bold">Configured neko images</v-card-title>
      <v-card-text>
        <p class="text--secondary">
          Images are configured with <code>NEKO_ROOMS_NEKO_IMAGES</code>. Pull an image to download or update it.
          You can restrict which images regular users may use in <router-link :to="{ name: 'admin-settings' }">Settings</router-link>.
        </p>
        <v-chip v-for="img in images" :key="img" class="mr-2 mb-2" label>{{ img }}</v-chip>
      </v-card-text>
    </v-card>
    <div class="text-center">
      <Pull />
    </div>
  </div>
</template>

<script lang="ts">
import { Vue, Component } from 'vue-property-decorator'
import Pull from '@/components/Pull.vue'

@Component({
  components: {
    Pull,
  }
})
export default class AdminImages extends Vue {
  get images(): string[] {
    return this.$store.state.roomsConfig.neko_images || []
  }

  mounted() {
    this.$store.dispatch('ROOMS_CONFIG')
  }
}
</script>
