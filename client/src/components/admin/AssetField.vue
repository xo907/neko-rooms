<template>
  <div class="nr-asset-field">
    <div class="d-flex align-center">
      <div v-if="preview" class="nr-asset-preview mr-3" :class="{ 'nr-asset-preview--font': isFont }">
        <img v-if="value && !isFont" :src="value" alt="">
        <v-icon v-else-if="isFont">mdi-format-font</v-icon>
        <v-icon v-else small class="text--disabled">mdi-image-off-outline</v-icon>
      </div>
      <v-text-field
        :value="value"
        @input="$emit('input', $event)"
        :label="label"
        :hint="hint"
        :persistent-hint="!!hint"
        placeholder="https://… or upload"
        outlined
        dense
        hide-details="auto"
        clearable
        @click:clear="$emit('input', '')"
      />
      <v-btn icon class="ml-1" :loading="uploading" @click="pick" title="Upload">
        <v-icon>mdi-upload</v-icon>
      </v-btn>
      <input ref="file" type="file" :accept="accept" class="d-none" @change="upload">
    </div>
  </div>
</template>

<style scoped>
.nr-asset-preview {
  width: 56px;
  min-width: 56px;
  height: 40px;
  border-radius: 6px;
  display: flex;
  align-items: center;
  justify-content: center;
  overflow: hidden;
  background: repeating-conic-gradient(rgba(128,128,128,.25) 0% 25%, transparent 0% 50%) 50% / 12px 12px;
}
.nr-asset-preview img {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
}
</style>

<script lang="ts">
import { Vue, Component, Prop } from 'vue-property-decorator'
import { api, errorMessage } from '@/api-ext'

@Component
export default class AssetField extends Vue {
  @Prop(String) readonly value!: string
  @Prop(String) readonly label!: string
  @Prop(String) readonly hint!: string
  @Prop({ type: String, required: true }) readonly asset!: string
  @Prop({ type: String, default: 'image/*' }) readonly accept!: string
  @Prop({ type: Boolean, default: true }) readonly preview!: boolean

  uploading = false

  get isFont() {
    return this.accept.includes('font') || this.accept.includes('woff')
  }

  pick() {
    (this.$refs.file as HTMLInputElement).click()
  }

  async upload(e: Event) {
    const input = e.target as HTMLInputElement
    const file = input.files && input.files[0]
    input.value = ''
    if (!file) return

    this.uploading = true
    try {
      const asset = await api.uploadAsset(this.asset, file)
      this.$emit('input', asset.url)
    } catch (err) {
      this.$swal({ icon: 'error', title: 'Upload failed', text: errorMessage(err) })
    } finally {
      this.uploading = false
    }
  }
}
</script>
