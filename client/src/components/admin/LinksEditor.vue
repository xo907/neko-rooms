<template>
  <div>
    <div v-for="(link, i) in value" :key="i" class="d-flex align-center mb-2" style="gap: 8px">
      <v-text-field :value="link.label" @input="set(i, 'label', $event)" label="Label" outlined dense hide-details style="max-width: 160px" />
      <v-text-field :value="link.url" @input="set(i, 'url', $event)" label="URL" outlined dense hide-details />
      <v-text-field :value="link.icon" @input="set(i, 'icon', $event)" label="Icon" placeholder="mdi-github" outlined dense hide-details style="max-width: 150px">
        <template v-slot:prepend-inner><v-icon small v-if="link.icon">{{ link.icon }}</v-icon></template>
      </v-text-field>
      <v-checkbox :input-value="link.new_tab" @change="set(i, 'new_tab', !!$event)" hide-details dense class="mt-0 pt-0" title="Open in new tab" off-icon="mdi-open-in-new" on-icon="mdi-open-in-new" color="primary" />
      <v-btn icon small :disabled="i === 0" @click="move(i, -1)"><v-icon small>mdi-arrow-up</v-icon></v-btn>
      <v-btn icon small @click="remove(i)"><v-icon small>mdi-close</v-icon></v-btn>
    </div>
    <v-btn small text color="primary" @click="add"><v-icon left small>mdi-plus</v-icon>Add link</v-btn>
    <span class="caption text--secondary ml-2">Icons: any <a href="https://pictogrammers.com/library/mdi/" target="_blank" rel="noopener">Material Design Icon</a> name</span>
  </div>
</template>

<script lang="ts">
import { Vue, Component, Prop } from 'vue-property-decorator'
import { Link } from '@/api-ext'

@Component
export default class LinksEditor extends Vue {
  @Prop({ type: Array, default: () => [] }) readonly value!: Link[]

  emit(list: Link[]) {
    this.$emit('input', list)
  }

  set(i: number, key: keyof Link, v: string | boolean) {
    const list = this.value.map((l, j) => (j === i ? { ...l, [key]: v } : l))
    this.emit(list)
  }

  add() {
    // eslint-disable-next-line
    this.emit([...this.value, { label: '', url: '', icon: '', new_tab: true }])
  }

  remove(i: number) {
    this.emit(this.value.filter((_, j) => j !== i))
  }

  move(i: number, d: number) {
    const list = [...this.value]
    const [item] = list.splice(i, 1)
    list.splice(i + d, 0, item)
    this.emit(list)
  }
}
</script>
