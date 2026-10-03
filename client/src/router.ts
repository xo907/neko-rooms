import Vue from 'vue'
import VueRouter, { RouteConfig } from 'vue-router'
import store from './store'
import { AuthStatus } from '@/api-ext'

Vue.use(VueRouter)

const routes: RouteConfig[] = [
  { path: '/', name: 'home', component: () => import('./views/HomePage.vue') },
  { path: '/r/:name', name: 'room', component: () => import('./views/RoomPage.vue'), props: true },

  { path: '/login', name: 'login', component: () => import('./views/AuthPage.vue'), props: { mode: 'login' }, meta: { guestOnly: true, bare: true } },
  { path: '/register', name: 'register', component: () => import('./views/AuthPage.vue'), props: { mode: 'register' }, meta: { guestOnly: true, bare: true } },
  { path: '/setup', name: 'setup', component: () => import('./views/AuthPage.vue'), props: { mode: 'setup' }, meta: { bare: true } },

  { path: '/my/rooms', name: 'my-rooms', component: () => import('./views/Home.vue'), meta: { requiresAuth: true, title: 'My rooms' } },
  { path: '/friends', name: 'friends', component: () => import('./views/FriendsPage.vue'), meta: { requiresAuth: true, title: 'Friends' } },
  { path: '/account', name: 'account', component: () => import('./views/AccountPage.vue'), meta: { requiresAuth: true, title: 'Account' } },

  {
    path: '/admin',
    component: () => import('./views/admin/AdminLayout.vue'),
    meta: { requiresAdmin: true },
    children: [
      { path: '', name: 'admin', component: () => import('./views/admin/AdminOverview.vue'), meta: { title: 'Admin' } },
      { path: 'rooms', name: 'admin-rooms', component: () => import('./views/admin/AdminRooms.vue'), meta: { title: 'Rooms' } },
      { path: 'users', name: 'admin-users', component: () => import('./views/admin/AdminUsers.vue'), meta: { title: 'Users' } },
      { path: 'branding', name: 'admin-branding', component: () => import('./views/admin/AdminBranding.vue'), meta: { title: 'Branding' } },
      { path: 'settings', name: 'admin-settings', component: () => import('./views/admin/AdminSettings.vue'), meta: { title: 'Settings' } },
      { path: 'images', name: 'admin-images', component: () => import('./views/admin/AdminImages.vue'), meta: { title: 'Images' } },
      { path: 'audit', name: 'admin-audit', component: () => import('./views/admin/AdminAudit.vue'), meta: { title: 'Audit log' } },
    ],
  },

  { path: '*', redirect: '/' },
]

const router = new VueRouter({
  mode: 'hash',
  routes,
  scrollBehavior(to, from, saved) {
    return saved || { x: 0, y: 0 }
  },
})

// only allow same-origin relative redirects
export function safeNext(next: unknown): string | null {
  if (typeof next !== 'string' || !next.startsWith('/') || next.startsWith('//') || next.includes('\\')) return null
  return next
}

router.beforeEach(async (to, from, next) => {
  // eslint-disable-next-line
  let status = (store.state as any).app.status as AuthStatus | null
  if (!status) {
    try {
      status = await store.dispatch('APP_STATUS') as AuthStatus
    } catch {
      return next()
    }
  }

  if (status.setup_required) {
    return to.name === 'setup' ? next() : next({ name: 'setup' })
  }
  if (to.name === 'setup') {
    return next({ name: 'home' })
  }

  const user = status.user
  if (to.matched.some(r => r.meta.guestOnly) && user) {
    return next({ name: 'home' })
  }
  if (to.matched.some(r => r.meta.requiresAuth || r.meta.requiresAdmin) && !user) {
    return next({ name: 'login', query: { redirect: to.fullPath } })
  }
  if (to.matched.some(r => r.meta.requiresAdmin) && user && user.role !== 'admin') {
    return next({ name: 'home' })
  }
  if (to.name === 'home' && !user && (!status.policy.homepage_enabled || !status.policy.guests_can_browse)) {
    return next({ name: 'login' })
  }

  next()
})

export default router
