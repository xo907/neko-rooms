<template>
  <div class="nr-auth" :style="pageStyle">
    <div class="nr-auth-inner" :class="'nr-auth--' + (login.position || 'center')">
      <div class="nr-auth-stack nr-rise">
        <router-link to="/" class="nr-auth-brand">
          <img v-if="login.show_logo && logo" :src="logo" :alt="branding.app_name" class="nr-auth-logo">
          <span class="nr-auth-name">{{ branding.app_name }}</span>
          <span class="nr-mono nr-auth-host">{{ host }}</span>
        </router-link>

        <section class="nr-auth-card" :style="cardStyle">
          <header class="nr-auth-head">
            <h1 class="nr-auth-heading">{{ heading }}</h1>
            <p v-if="message" class="nr-auth-message">{{ message }}</p>
            <p v-else-if="mode === 'register'" class="nr-auth-message">Rooms, friends and watch parties in one place.</p>
          </header>

          <div class="nr-auth-body">
            <div v-if="error" class="nr-auth-note is-error" role="alert">{{ error }}</div>
            <div v-if="pending" class="nr-auth-note is-ok" role="status">
              Your account was created and is waiting for approval by an administrator.
            </div>

            <v-form v-if="!pending" ref="form" class="nr-auth-form" @submit.prevent="submit">
              <div v-if="mode === 'setup'" class="nr-auth-field">
                <label class="nr-label" for="nr-a-token">Setup token</label>
                <v-text-field id="nr-a-token" v-model="setupToken" outlined dense hide-details class="nr-input-mono" autocomplete="off" />
                <span class="nr-auth-hint">Printed in the server logs on first start</span>
              </div>

              <div class="nr-auth-field">
                <label class="nr-label" for="nr-a-user">Username</label>
                <v-text-field id="nr-a-user" v-model="username" outlined dense hide-details autocomplete="username" autofocus />
              </div>

              <div v-if="mode === 'register'" class="nr-auth-field">
                <label class="nr-label" for="nr-a-display">Display name <span class="nr-auth-optional">optional</span></label>
                <v-text-field id="nr-a-display" v-model="displayName" outlined dense hide-details autocomplete="nickname" />
              </div>

              <div class="nr-auth-field">
                <label class="nr-label" for="nr-a-pass">Password</label>
                <v-text-field
                  id="nr-a-pass"
                  v-model="password"
                  :type="showPass ? 'text' : 'password'"
                  outlined
                  dense
                  hide-details
                  class="nr-input-mono"
                  :autocomplete="mode === 'login' ? 'current-password' : 'new-password'"
                >
                  <template v-slot:append>
                    <button type="button" class="nr-btn is-icon is-sm" :aria-label="showPass ? 'Hide password' : 'Show password'" @click="showPass = !showPass">
                      <v-icon small>{{ showPass ? 'mdi-eye-off-outline' : 'mdi-eye-outline' }}</v-icon>
                    </button>
                  </template>
                </v-text-field>
                <span v-if="mode !== 'login'" class="nr-auth-hint nr-mono">at least {{ minLength }} characters</span>
              </div>

              <div v-if="mode !== 'login'" class="nr-auth-field">
                <label class="nr-label" for="nr-a-pass2">Repeat password</label>
                <v-text-field id="nr-a-pass2" v-model="password2" :type="showPass ? 'text' : 'password'" outlined dense hide-details class="nr-input-mono" autocomplete="new-password" />
              </div>

              <button type="submit" class="nr-btn is-primary nr-auth-submit" :disabled="loading">
                <v-progress-circular v-if="loading" indeterminate size="14" width="2" />
                {{ mode === 'login' ? 'Sign in' : (mode === 'setup' ? 'Create admin account' : 'Create account') }}
              </button>
            </v-form>
          </div>

          <footer v-if="mode !== 'setup'" class="nr-auth-foot">
            <template v-if="mode === 'login' && policy && policy.registration_enabled">
              <span>New here?</span>
              <router-link :to="{ name: 'register', query: $route.query }">Create an account</router-link>
            </template>
            <template v-else-if="mode === 'register'">
              <span>Have an account?</span>
              <router-link :to="{ name: 'login', query: $route.query }">Sign in instead</router-link>
            </template>
            <template v-else-if="canBrowse">
              <span>Just looking?</span>
              <router-link to="/">Browse rooms</router-link>
            </template>
          </footer>
        </section>

        <router-link v-if="mode !== 'setup' && canBrowse && (mode === 'register' || (policy && policy.registration_enabled))" to="/" class="nr-auth-browse nr-mono">
          Browse rooms without an account
        </router-link>
        <p v-if="login.footer_text" class="nr-auth-footer nr-mono">{{ expand(login.footer_text) }}</p>
      </div>
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
.nr-auth-stack {
  width: min(380px, 100%);
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 24px;
}
.nr-auth-brand {
  display: inline-flex;
  align-items: baseline;
  gap: 8px;
  color: var(--text-primary) !important;
  text-decoration: none;
}
.nr-auth-logo {
  align-self: center;
  max-height: 24px;
  max-width: 140px;
  border-radius: var(--radius-sm);
}
.nr-auth-name { font-size: 15px; font-weight: 600; letter-spacing: -0.03em; }
.nr-auth-host { font-size: 11.5px; color: var(--text-tertiary); }

.nr-auth-card {
  width: 100%;
  border: 1px solid var(--border-subtle);
  border-radius: var(--radius-lg);
  background: var(--bg-surface-1);
  overflow: hidden;
}
.nr-auth-head { padding: 24px 24px 0; display: flex; flex-direction: column; gap: 8px; }
.nr-auth-heading {
  margin: 0;
  font-size: 22px !important;
  font-weight: 600 !important;
  line-height: 1.2;
  letter-spacing: -0.03em !important;
  color: var(--text-primary);
}
.nr-auth-message { margin: 0; font-size: 13.5px; line-height: 1.55; color: var(--text-secondary); white-space: pre-line; }
.nr-auth-body { padding: 24px; display: flex; flex-direction: column; gap: 16px; }
.nr-auth-form { display: flex; flex-direction: column; gap: 16px; }
.nr-auth-field { display: flex; flex-direction: column; gap: 8px; }
.nr-auth-optional { margin-left: 4px; text-transform: none; letter-spacing: 0; color: var(--text-disabled); }
.nr-auth-hint { font-size: 11.5px; color: var(--text-tertiary); }
.nr-auth-submit { width: 100%; height: 40px; margin-top: 8px; font-size: 14px; }
.nr-auth-note {
  padding: 10px 12px;
  border-radius: var(--radius-md);
  border: 1px solid var(--border-subtle);
  background: var(--bg-surface-2);
  font-size: 13px;
  line-height: 1.5;
  &.is-error { border-color: rgba(239, 68, 68, .35); color: var(--status-danger-ink); }
  &.is-ok { border-color: rgba(16, 185, 129, .35); color: var(--status-online-ink); }
}
.nr-auth-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 24px;
  border-top: 1px solid var(--border-subtle);
  font-size: 12.5px;
  color: var(--text-tertiary);
  a { color: var(--text-primary) !important; font-weight: 500; text-decoration: none; }
  a:hover { text-decoration: underline; text-underline-offset: 3px; text-decoration-color: var(--border-focus); }
}
.nr-auth-browse {
  font-size: 12px;
  color: var(--text-tertiary) !important;
  text-decoration: none;
  &:hover { color: var(--text-primary) !important; }
}
.nr-auth-footer { margin: 0; font-size: 11.5px; line-height: 1.6; color: var(--text-tertiary); text-align: center; white-space: pre-line; }
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

  get host() {
    return window.location.host
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
      backgroundColor: this.login.background_color || 'transparent',
    }
    if (this.login.background_image) {
      style.backgroundImage = `url("${this.login.background_image.replace(/"/g, '')}")`
    }
    return style
  }

  get cardStyle() {
    const opacity = this.login.card_opacity ?? 100
    return opacity < 100 ? { backgroundColor: `color-mix(in srgb, var(--bg-surface-1) ${opacity}%, transparent)` } : {}
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
