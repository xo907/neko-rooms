import './class-component-hooks'
import Vue from 'vue'
import App from './App.vue'
import './plugins/filters.ts'
import sweetalert from './plugins/sweetalert'
import store from './store'
import router from './router'
import vuetify from './plugins/vuetify'

import '@/api-ext'
import '@/assets/styles/main.scss'

Vue.config.productionTip = false

Vue.use(sweetalert)

async function boot() {
  // branding is normally inlined by the server, fetch it otherwise (dev server)
  // eslint-disable-next-line
  if (!(store.state as any).app.branding) {
    try {
      await store.dispatch('APP_BRANDING')
    } catch (e) {
      console.error('unable to load branding', e)
    }
  }

  try {
    await store.dispatch('APP_STATUS')
  } catch (e) {
    console.error('unable to load status', e)
  }

  new Vue({
    store,
    router,
    vuetify,
    render: h => h(App)
  }).$mount('#app')
}

boot()
