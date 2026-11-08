<template>
  <!-- a row in a menu -->
  <v-list-item v-if="tmpl && variant === 'item'" :disabled="disabled || loading" @click="Action">
    <v-list-item-title :class="{ 'nr-action-warn': highlight, 'nr-action-danger': action === 'remove' }">{{ tmpl.label }}</v-list-item-title>
    <v-progress-circular v-if="loading" indeterminate size="14" width="2" class="ml-3" />
  </v-list-item>

  <!-- a labelled button -->
  <button
    v-else-if="tmpl && variant === 'button'"
    type="button"
    class="nr-btn is-sm is-secondary"
    :disabled="disabled || loading"
    @click="Action"
  >
    <v-progress-circular v-if="loading" indeterminate size="12" width="2" />
    <v-icon v-else small>{{ tmpl.icon }}</v-icon>{{ tmpl.label }}
  </button>

  <!-- an icon with a tooltip -->
  <v-tooltip bottom v-else-if="tmpl" open-delay="300">
    <template v-slot:activator="{ on, attrs }">
      <button
        v-bind="attrs"
        v-on="on"
        type="button"
        class="nr-btn is-icon is-sm"
        :class="{ 'is-danger': action === 'remove' }"
        :disabled="disabled || loading"
        :aria-label="tmpl.tooltip"
        @click="Action"
      >
        <v-progress-circular v-if="loading" indeterminate size="14" width="2" />
        <v-icon v-else small>{{ tmpl.icon }}</v-icon>
      </button>
    </template>
    <span>{{ tmpl.tooltip }}</span>
  </v-tooltip>
</template>

<style lang="scss" scoped>
.nr-action-warn { color: var(--status-provisioning-ink) !important; }
.nr-action-danger { color: var(--status-danger-ink) !important; }
</style>

<script lang="ts">
import { Vue, Component, Prop } from 'vue-property-decorator'
import { AxiosError } from 'axios'
import { api, errorMessage } from '@/api-ext'

@Component
export default class RoomActionBtn extends Vue {
  @Prop(String) readonly action!: string
  @Prop(String) readonly roomId!: string
  @Prop(String) readonly roomName!: string
  @Prop(Boolean) readonly disabled!: boolean
  // icon (default), item for a menu row or button for a labelled button
  @Prop({ type: String, default: 'icon' }) readonly variant!: string
  @Prop(Boolean) readonly highlight!: boolean

  public loading = false

  // eslint-disable-next-line
  get tmpl(): any | undefined {
    switch (this.action) {
      case 'start':
        return {
          dispatch: 'ROOMS_START',
          msg: 'Room started!',
          tooltip: 'Start',
          label: 'Start',
          color: 'green',
          icon: 'mdi-power',
        }
      case 'stop':
        return {
          dispatch: 'ROOMS_STOP',
          msg: 'Room stopped!',
          tooltip: 'Stop',
          label: 'Stop',
          color: 'warning',
          icon: 'mdi-stop',
        }
      case 'pause':
        return {
          dispatch: 'ROOMS_PAUSE',
          msg: 'Room paused!',
          tooltip: 'Pause',
          label: 'Pause',
          color: 'orange',
          icon: 'mdi-pause',
        }
      case 'restart':
        return {
          dispatch: 'ROOMS_RESTART',
          msg: 'Room restarted!',
          tooltip: 'Restart',
          label: 'Restart',
          color: 'blue',
          icon: 'mdi-refresh',
        }
      case 'recreate':
        return {
          dispatch: 'ROOMS_RECREATE',
          msg: 'Room recreated!',
          tooltip: 'Recreate',
          label: 'Recreate',
          color: 'blue',
          icon: 'mdi-backup-restore',
        }
      case 'upgrade':
        return {
          msg: 'Room updated!',
          tooltip: 'Update the image: pull the newest version and recreate the room on it',
          label: 'Update image',
          color: 'primary',
          icon: 'mdi-update',
        }
      case 'remove':
        return {
          dispatch: 'ROOMS_REMOVE',
          msg: 'Room removed!',
          tooltip: 'Destroy room',
          label: 'Destroy room',
          color: 'red',
          icon: 'mdi-trash-can-outline',
        }
    }

    return undefined
  }

  // upgrade pulls in the background on the server and waits for it here
  async upgrade() {
    this.loading = true
    try {
      // eslint-disable-next-line
      let state: any = await api.upgradeRoom(this.roomId)
      while (state.state === 'pulling' || state.state === 'recreating') {
        await new Promise(resolve => setTimeout(resolve, 2000))
        state = await api.upgradeStatus(this.roomName)
      }
      if (state.state === 'failed') throw new Error(state.error || 'The update failed')

      this.$emit('done', this.action)
      this.$swal({
        title: state.up_to_date ? 'Already up to date' : 'Room updated!',
        text: state.up_to_date ? 'The room runs the newest version of its image.' : 'The room was recreated on the newest image.',
        icon: 'success',
      })
    } catch (e) {
      this.$swal({ title: 'Unable to update', text: errorMessage(e), icon: 'error' })
    } finally {
      this.loading = false
    }
  }

  async Action() {
    if (!this.tmpl) return

    if (this.action === 'remove') {
      const { value } = await this.$swal({
        title: "Destroy room",
        text: "The room and its container are removed. This can not be undone.",
        icon: 'warning',
        showCancelButton: true,
        confirmButtonText: "Destroy",
        cancelButtonText: "Cancel",
      })

      if (!value) return
    } else if (this.action === 'upgrade') {
      const { value } = await this.$swal({
        title: 'Update room',
        text: 'Pull the newest version of the image and recreate the room on it, when there is one. Data that is not kept in storage is lost.',
        icon: 'question',
        showCancelButton: true,
        confirmButtonText: 'Update',
        cancelButtonText: 'Cancel',
      })

      if (!value) return
      await this.upgrade()
      return
    } else if (this.action === 'recreate') {
      const { value } = await this.$swal({
        title: "Recreate room",
        text: "Do you really want to recreate this room? It will delete all your non-persistent data.",
        icon: 'warning',
        showCancelButton: true,
        confirmButtonText: "Yes",
        cancelButtonText: "No",
      })

      if (!value) return
    }

    this.loading = true

    try {
      await this.$store.dispatch(this.tmpl.dispatch, this.roomId)
      this.$emit("done", this.action)
      this.$swal({
        title: this.tmpl.msg,
        icon: 'success',
      })
    } catch (e) {
      const response = (e as AxiosError).response
      if (response) {
        this.$swal({
          title: 'Server error',
          text: String(response.data),
          icon: 'error',
        })
      } else {
        this.$swal({
          title: 'Network error',
          text: String(e),
          icon: 'error',
        })
      }
    } finally {
      this.loading = false
    }
  }
}
</script>
