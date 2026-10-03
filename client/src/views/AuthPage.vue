<template>
  <div class="nr-auth" :style="pageStyle">
    <div class="nr-auth-inner" :class="'nr-auth--' + (login.position || 'center')">
      <v-card class="nr-auth-card pa-6 pa-sm-8" :style="cardStyle" elevation="12">
        <router-link to="/" class="d-flex flex-column align-center text-decoration-none mb-6">
          <img v-if="login.show_logo && logo" :src="logo" :alt="branding.app_name" class="nr-auth-logo mb-3">
          <div class="text-h5 font-weight-bold nr-heading" style="color: var(--nr-text)">{{ branding.app_name }}</div>
        </router-link>

        <div class="text-h6 text-center mb-1">{{ heading }}</div>
        <p v-if="message" class="text-center text--secondary mb-6" style="white-space: pre-line">{{ message }}</p>
        <div v-else class="mb-6" />

        <v-alert v-if="error" type="error" dense text class="mb-4">{{ error }}</v-alert>
        <v-alert v-if="pending" type="success" dense text class="mb-4">
          Your account was created and is waiting for approval by an administrator.
        </v-alert>

        <v-form ref="form" @submit.prevent="submit" v-if="!pending">
          <v-text-field
            v-if="mode === 'setup'"
            v-model="setupToken"
            label="Setup token"
            hint="Printed in the server logs on first start"
            persistent-hint
            outlined
            prepend-inner-icon="mdi-key-outline"
            autocomplete="off"
            class="mb-3"
          />
          <v-text-field
            v-model="username"
            label="Username"
            outlined
            prepend-inner-icon="mdi-account-outline"
            autocomplete="username"
            autofocus
          />
          <v-text-field
            v-if="mode === 'register'"
            v-model="displayName"
            label="Display name (optional)"
            outlined
            prepend-inner-icon="mdi-card-account-details-outline"
          />
          <v-text-field
            v-model="password"
            label="Password"
            :type="showPass ? 'text' : 'password'"
            outlined
            prepend-inner-icon="mdi-lock-outline"
            :append-icon="showPass ? 'mdi-eye' : 'mdi-eye-off'"
            @click:append="showPass = !showPass"
            :autocomplete="mode === 'login' ? 'current-password' : 'new-password'"
            :hint="mode !== 'login' ? `At least ${minLength} characters` : ''"
          />
          <v-text-field
            v-if="mode !== 'login'"
            v-model="password2"
            label="Repeat password"
            :type="showPass ? 'text' : 'password'"
            outlined
            prepend-inner-icon="mdi-lock-check-outline"
            autocomplete="new-password"
          />

          <v-btn type="submit" color="primary" block x-large depressed :loading="loading" class="mt-2">
            {{ mode === 'login' ? 'Sign in' : (mode === 'setup' ? 'Create admin account' : 'Create account') }}
          </v-btn>
        </v-form>

        <div class="text-center mt-6 body-2">
          <template v-if="mode === 'login' && policy && policy.registration_enabled">
            New here? <router-link :to="{ name: 'register', query: $route.query }">Create an account</router-link>
          </template>
          <template v-else-if="mode === 'register'">
            Already have an account? <router-link :to="{ name: 'login', query: $route.query }">Sign in</router-link>
          </template>
          <div v-if="mode !== 'setup' && canBrowse" class="mt-2">
            <router-link to="/">Browse rooms without an account</router-link>
          </div>
        </div>

        <p v-if="login.footer_text" class="caption text-center text--secondary mt-6 mb-0" style="white-space: pre-line">{{ expand(login.footer_text) }}</p>
      </v-card>
    </div>
  </div>
</template>

<style lang="scss" scoped>
.nr-auth {
  min-height: 100vh;
  background-size: cover;
  background-position: center;
  display: flex;
}
.nr-auth-inner {
  flex: 1;
  display: flex;
  align-items: center;
  padding: 24px 16px;
  &.nr-auth--center { justify-content: center; }
  &.nr-auth--left { justify-content: flex-start; padding-left: 6vw; }
  &.nr-auth--right { justify-content: flex-end; padding-right: 6vw; }
}
.nr-auth-card {
  width: 100%;
  max-width: 430px;
}
.nr-auth-logo {
  max-height: 64px;
  max-width: 220px;
}
</style>

<script lang="ts">
import { Vue, Component, Prop } from 'vue-property-decorator'
import { api, Branding, errorMessage } from '@/api-ext'
import { expand, logoFor } from '@/branding/theme'
import { safeNext } from '@/router'

@Component
export default class AuthPage extends Vue {
  @Prop({ required: true }) readonly mode!: 'login' | 'register' | 'setup'

  username = ''
  password = ''
  password2 = ''
  displayName = ''
  setupToken = ''
  showPass = false
  loading = false
  error = ''
  pending = false

  get branding(): Branding {
    return this.$store.state.app.branding
  }

  get login() {
    return this.branding.login
  }

  get policy() {
    return this.$store.getters.policy
  }

  get canBrowse() {
    return this.policy && this.policy.homepage_enabled && this.policy.guests_can_browse
  }

  get minLength() {
    return this.policy ? this.policy.password_min_length : 8
  }

  get logo() {
    return logoFor(this.branding, this.$store.state.app.dark)
  }

  get heading() {
    if (this.mode === 'setup') return 'Welcome! Create the first admin account'
    if (this.mode === 'register') return 'Create your account'
    return this.login.title || 'Sign in'
  }

  get message() {
    return this.mode === 'login' ? this.login.message : ''
  }

  get pageStyle() {
    const style: Record<string, string> = {
      backgroundColor: this.login.background_color || 'var(--nr-bg)',
    }
    if (this.login.background_image) {
      style.backgroundImage = `url("${this.login.background_image.replace(/"/g, '')}")`
    } else {
      style.backgroundImage = 'radial-gradient(circle at 20% 20%, color-mix(in srgb, var(--nr-primary) 35%, transparent), transparent 50%), radial-gradient(circle at 80% 80%, color-mix(in srgb, var(--nr-accent) 30%, transparent), transparent 50%)'
    }
    return style
  }

  get cardStyle() {
    const opacity = this.login.card_opacity ?? 100
    return opacity < 100 ? { backgroundColor: `color-mix(in srgb, var(--nr-surface) ${opacity}%, transparent) !important`, backdropFilter: 'blur(12px)' } : {}
  }

  expand(s: string) {
    return expand(this.branding, s)
  }

  async submit() {
    this.error = ''
    if (this.mode !== 'login' && this.password !== this.password2) {
      this.error = 'Passwords do not match'
      return
    }

    this.loading = true
    try {
      if (this.mode === 'login') {
        await this.$store.dispatch('APP_LOGIN', { username: this.username, password: this.password })
      } else if (this.mode === 'register') {
        // eslint-disable-next-line
        const res = await api.register({ username: this.username, password: this.password, display_name: this.displayName })
        if ('pending_approval' in res) {
          this.pending = true
          return
        }
        await this.$store.dispatch('APP_STATUS')
      } else {
        // eslint-disable-next-line
        await api.setup({ setup_token: this.setupToken.trim(), username: this.username, password: this.password })
        await this.$store.dispatch('APP_STATUS')
      }
      this.redirect()
    } catch (e) {
      this.error = errorMessage(e)
    } finally {
      this.loading = false
    }
  }

  redirect() {
    // "next" is a full path outside the app (e.g. a room), "redirect" an app route
    const next = safeNext(this.$route.query.next)
    if (next) {
      window.location.href = next
      return
    }
    const redirect = safeNext(this.$route.query.redirect)
    this.$router.replace(redirect || (this.mode === 'setup' ? '/admin' : '/')).catch(() => { /* ignore */ })
  }
}
</script>
