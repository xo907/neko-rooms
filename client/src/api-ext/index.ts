// Hand-written client for the user, community, branding and admin APIs.
import axios from 'axios'

export const basePath = (location.protocol + '//' + location.host + location.pathname).replace(/\/+$/, '')

// mark all requests as XHR, required by the server's CSRF protection
axios.defaults.headers.common['X-Requested-With'] = 'XMLHttpRequest'
axios.defaults.withCredentials = true

const http = axios.create({ baseURL: basePath + '/api' })
http.defaults.headers.common['X-Requested-With'] = 'XMLHttpRequest'

export type Role = 'admin' | 'user'
export type Visibility = 'public' | 'friends' | 'private'

export interface User {
  id: number
  username: string
  display_name: string
  email: string
  role: Role
  disabled: boolean
  room_limit: number
  created_at: string
  updated_at: string
  last_login_at: string | null
  room_count?: number
}

export interface PublicUser {
  id: number
  username: string
  display_name: string
}

export interface PublicPolicy {
  homepage_enabled: boolean
  guests_can_browse: boolean
  rooms_require_login: boolean
  registration_enabled: boolean
  registration_approval: boolean
  friends_enabled: boolean
  thumbnails_enabled: boolean
  users_can_create_rooms: boolean
  users_can_make_public: boolean
  users_can_pull_images: boolean
  users_can_use_mounts: boolean
  password_min_length: number
}

export interface AuthStatus {
  setup_required: boolean
  user: User | null
  policy: PublicPolicy
  room_limit: number
  room_count: number
}

export interface Policy extends PublicPolicy {
  users_can_set_devices: boolean
  user_neko_images: string[]
  default_room_limit: number
  user_max_connections: number
  show_member_names: boolean
  session_ttl_hours: number
  audit_retention: number
}

export interface Room {
  id: string
  name: string
  title: string
  description: string
  category: string
  visibility: Visibility
  featured: boolean
  hidden?: boolean
  owner: PublicUser | null
  is_owner: boolean
  is_friend: boolean
  can_manage: boolean
  can_join: boolean
  running: boolean
  ready: boolean
  paused: boolean
  viewers: number
  members: string[]
  friends_inside: string[]
  max_connections: number
  has_thumbnail: boolean
  created_at: string
  invite_code?: string
}

export interface RoomMeta {
  name: string
  owner_id: number | null
  visibility: Visibility
  title: string
  description: string
  category: string
  featured: boolean
  hidden: boolean
  invite_code?: string
}

export interface RoomMetaUpdate {
  title?: string
  description?: string
  category?: string
  visibility?: Visibility
  featured?: boolean
  hidden?: boolean
  owner_id?: number
}

export interface Friend {
  user: PublicUser
  status: 'pending' | 'accepted'
  direction: 'incoming' | 'outgoing'
  created_at: string
}

export interface Session {
  id: string
  created_at: string
  last_seen_at: string
  expires_at: string
  ip: string
  user_agent: string
  current: boolean
}

export interface Asset {
  name: string
  mime: string
  size: number
  url: string
  updated_at: string
}

export interface AuditEntry {
  id: number
  created_at: string
  user_id: number | null
  username: string
  action: string
  target: string
  details: string
  ip: string
}

export interface Link {
  label: string
  url: string
  icon: string
  new_tab: boolean
}

export interface Palette {
  primary: string
  secondary: string
  accent: string
  error: string
  info: string
  success: string
  warning: string
  background: string
  surface: string
  text: string
  app_bar: string
  app_bar_text: string
  footer: string
  footer_text: string
  drawer: string
}

export interface LobbyMessage {
  title: string
  message: string
}

export interface Branding {
  app_name: string
  page_title: string
  tagline: string
  description: string
  logo: string
  logo_light: string
  favicon: string
  theme_color: string
  theme: {
    mode: 'dark' | 'light' | 'system'
    allow_user_toggle: boolean
    dark: Palette
    light: Palette
    font_family: string
    font_url: string
    font_file: string
    heading_font_family: string
    font_size: number
    border_radius: number
    dense: boolean
    button_uppercase: boolean
  }
  header: {
    show: boolean
    show_logo: boolean
    show_app_name: boolean
    logo_height: number
    elevation: number
    links: Link[]
    announcement: string
    announcement_type: string
  }
  footer: {
    show: boolean
    text: string
    links: Link[]
    show_powered_by: boolean
    powered_by_label: string
  }
  home: {
    hero_show: 'guests' | 'always' | 'never'
    hero_title: string
    hero_subtitle: string
    hero_image: string
    hero_gradient: boolean
    cta_label: string
    featured_label: string
    friends_label: string
    public_label: string
    mine_label: string
    empty_text: string
    show_search: boolean
    show_categories: boolean
    show_friends: boolean
    show_offline: boolean
    card_style: 'tile' | 'compact'
    categories: string[]
  }
  login: {
    title: string
    message: string
    show_logo: boolean
    background_color: string
    background_image: string
    card_opacity: number
    position: 'center' | 'left' | 'right'
    footer_text: string
  }
  lobby: {
    page_title: string
    show_logo: boolean
    background_color: string
    background_image: string
    popup_color: string
    text_color: string
    button_color: string
    font_family: string
    not_found: LobbyMessage
    not_running: LobbyMessage
    paused: LobbyMessage
    not_ready: LobbyMessage
    ready: LobbyMessage
    login_required: LobbyMessage
    custom_css: string
  }
  rooms: {
    inject: boolean
    page_title: string
    favicon: string
    custom_css: string
    custom_js: string
    custom_head: string
  }
  custom_css: string
  custom_head: string
  custom_js: string
}

export function errorMessage(e: unknown): string {
  // eslint-disable-next-line
  const err = e as any
  if (err && err.response) {
    const data = err.response.data
    if (typeof data === 'string' && data.trim()) return data.trim()
    return 'Server error (' + err.response.status + ')'
  }
  return String(e)
}

export const api = {
  // auth
  status: () => http.get<AuthStatus>('/auth/status').then(r => r.data),
  login: (username: string, password: string) => http.post<User>('/auth/login', { username, password }).then(r => r.data),
  logout: () => http.post('/auth/logout'),
  register: (data: { username: string; password: string; display_name?: string; email?: string }) =>
    http.post<User | { pending_approval: boolean }>('/auth/register', data).then(r => r.data),
  setup: (data: { setup_token: string; username: string; password: string }) => http.post<User>('/auth/setup', data).then(r => r.data),

  // account
  updateAccount: (data: { display_name?: string; email?: string }) => http.put<User>('/account', data).then(r => r.data),
  changePassword: (current: string, next: string) =>
    // eslint-disable-next-line
    http.post('/account/password', { current_password: current, new_password: next }),
  sessions: () => http.get<Session[]>('/account/sessions').then(r => r.data),
  revokeOtherSessions: () => http.delete('/account/sessions'),

  // branding
  branding: () => http.get<Branding>('/branding').then(r => r.data),

  // community
  directory: () => http.get<Room[]>('/public/rooms').then(r => r.data),
  room: (name: string, invite?: string) => http.get<Room>('/public/rooms/' + encodeURIComponent(name), { params: invite ? { invite } : {} }).then(r => r.data),
  // eslint-disable-next-line
  join: (name: string, invite?: string, displayName?: string) => http.post<{ url: string }>('/public/rooms/' + encodeURIComponent(name) + '/join', { invite, display_name: displayName }).then(r => r.data.url),
  thumbnailUrl: (name: string, bust: number, invite?: string) =>
    basePath + '/api/public/rooms/' + encodeURIComponent(name) + '/thumbnail.jpg?t=' + bust + (invite ? '&invite=' + encodeURIComponent(invite) : ''),

  roomMeta: (id: string) => http.get<RoomMeta>('/rooms/' + id + '/meta').then(r => r.data),
  updateRoomMeta: (id: string, data: RoomMetaUpdate) => http.put<RoomMeta>('/rooms/' + id + '/meta', data).then(r => r.data),
  regenerateInvite: (id: string, disable = false) => http.post<RoomMeta>('/rooms/' + id + '/invite', null, { params: disable ? { disable: 'true' } : {} }).then(r => r.data),

  // friends
  friends: () => http.get<Friend[]>('/friends').then(r => r.data),
  addFriend: (username: string) => http.post<{ status: string; user: PublicUser }>('/friends', { username }).then(r => r.data),
  acceptFriend: (id: number) => http.post('/friends/' + id + '/accept'),
  removeFriend: (id: number) => http.delete('/friends/' + id),
  searchUsers: (q: string) => http.get<PublicUser[]>('/users/search', { params: { q } }).then(r => r.data),

  // admin
  overview: () => http.get<Record<string, number>>('/admin/overview').then(r => r.data),
  users: () => http.get<User[]>('/admin/users').then(r => r.data),
  createUser: (data: Partial<User> & { password: string }) => http.post<User>('/admin/users', data).then(r => r.data),
  updateUser: (id: number, data: Partial<User> & { password?: string }) => http.put<User>('/admin/users/' + id, data).then(r => r.data),
  deleteUser: (id: number) => http.delete('/admin/users/' + id),
  userSessions: (id: number) => http.get<Session[]>('/admin/users/' + id + '/sessions').then(r => r.data),
  revokeUserSessions: (id: number) => http.delete('/admin/users/' + id + '/sessions'),
  adminRooms: () => http.get<Room[]>('/admin/rooms').then(r => r.data),
  policy: () => http.get<Policy>('/admin/policy').then(r => r.data),
  updatePolicy: (p: Policy) => http.put<Policy>('/admin/policy', p).then(r => r.data),
  brandingDefaults: () => http.get<Branding>('/admin/branding/defaults').then(r => r.data),
  updateBranding: (b: Branding) => http.put<Branding>('/admin/branding', b).then(r => r.data),
  resetBranding: () => http.post<Branding>('/admin/branding/reset').then(r => r.data),
  assets: () => http.get<Asset[]>('/admin/branding/assets').then(r => r.data),
  uploadAsset: (name: string, file: File) => {
    const form = new FormData()
    form.append('file', file)
    return http.post<Asset>('/admin/branding/assets/' + name, form).then(r => r.data)
  },
  deleteAsset: (name: string) => http.delete('/admin/branding/assets/' + name),
  audit: (limit: number, offset: number) => http.get<{ entries: AuditEntry[]; total: number }>('/admin/audit', { params: { limit, offset } }).then(r => r.data),
}
