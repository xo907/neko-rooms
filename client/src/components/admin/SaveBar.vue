<template>
  <div class="nr-savebar-wrap">
    <v-sheet class="nr-savebar d-flex align-center px-4 py-2" elevation="12">
      <slot />
    </v-sheet>
  </div>
</template>

<style scoped>
.nr-savebar-wrap {
  position: sticky;
  bottom: 16px;
  display: flex;
  justify-content: flex-end;
  z-index: 5;
  pointer-events: none;
  margin-top: 16px;
}
.nr-savebar {
  pointer-events: auto;
  border-radius: var(--nr-radius) !important;
  border: 1px solid var(--nr-primary);
  gap: 8px;
}
</style>

<script lang="ts">
import { Vue, Component } from 'vue-property-decorator'

@Component
export default class SaveBar extends Vue {}
</script>
