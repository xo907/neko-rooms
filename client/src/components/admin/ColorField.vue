<template>
  <v-text-field
    :value="value"
    @input="$emit('input', $event)"
    :label="label"
    :hint="hint"
    :persistent-hint="!!hint"
    outlined
    dense
    hide-details="auto"
    class="nr-color-field"
  >
    <template v-slot:prepend-inner>
      <v-menu offset-y :close-on-content-click="false" min-width="300">
        <template v-slot:activator="{ on, attrs }">
          <div v-bind="attrs" v-on="on" class="nr-swatch" :style="{ background: value || 'transparent' }" />
        </template>
        <v-color-picker
          :value="pickerValue"
          @input="onPick"
          mode="hexa"
          show-swatches
          :swatches="swatches"
          swatches-max-height="120"
        />
      </v-menu>
    </template>
  </v-text-field>
</template>

<style scoped>
.nr-swatch {
  width: 22px;
  height: 22px;
  border-radius: 6px;
  border: 1px solid rgba(128, 128, 128, .5);
  cursor: pointer;
  margin-right: 6px;
  margin-top: 1px;
}
</style>

<script lang="ts">
import { Vue, Component, Prop } from 'vue-property-decorator'

@Component
export default class ColorField extends Vue {
  @Prop(String) readonly value!: string
  @Prop(String) readonly label!: string
  @Prop(String) readonly hint!: string

  swatches = [
    ['#8c5cff', '#6c3cf0', '#3f51b5'],
    ['#ff5ca8', '#e83e8c', '#f44336'],
    ['#3ec5ff', '#2196f3', '#00bcd4'],
    ['#2ee59d', '#4caf50', '#009688'],
    ['#ffb547', '#ff9800', '#ffc107'],
    ['#13111c', '#1d1a2b', '#ffffff'],
  ]

  get pickerValue() {
    return /^#[0-9a-fA-F]{6}([0-9a-fA-F]{2})?$/.test(this.value || '') ? this.value : '#000000'
  }

  onPick(v: string | { hexa: string }) {
    let hex = typeof v === 'string' ? v : v.hexa
    // drop alpha when fully opaque
    if (/^#[0-9a-fA-F]{8}$/.test(hex) && hex.toLowerCase().endsWith('ff')) hex = hex.slice(0, 7)
    if (hex !== this.value) this.$emit('input', hex)
  }
}
</script>
