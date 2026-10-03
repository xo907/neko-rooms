import Vue from 'vue'
import { api, errorMessage } from '@/api-ext'

const NAME_KEY = 'neko-rooms-guest-name'

// joinRoom resolves the room link (with credentials) and navigates to it.
export async function joinRoom(vm: Vue, name: string, invite?: string): Promise<void> {
  let displayName: string | undefined

  if (!vm.$store.getters.user) {
    let saved = ''
    try { saved = localStorage.getItem(NAME_KEY) || '' } catch { /* ignore */ }

    const res = await vm.$swal({
      title: 'Pick a nickname',
      input: 'text',
      inputValue: saved,
      inputPlaceholder: 'Your name',
      inputAttributes: { maxlength: '32', autocapitalize: 'off' },
      showCancelButton: true,
      confirmButtonText: 'Join room',
      inputValidator: (v: string) => (!v || !v.trim()) ? 'Please enter a name' : null,
    })
    if (!res.isConfirmed) return

    displayName = String(res.value).trim()
    try { localStorage.setItem(NAME_KEY, displayName) } catch { /* ignore */ }
  }

  try {
    const url = await api.join(name, invite, displayName)
    window.location.href = url
  } catch (e) {
    vm.$swal({
      title: 'Unable to join',
      text: errorMessage(e),
      icon: 'error',
    })
  }
}

export function inviteLink(name: string, code: string): string {
  const base = location.href.split('#')[0]
  return `${base}#/r/${encodeURIComponent(name)}?invite=${encodeURIComponent(code)}`
}

export function roomLink(name: string): string {
  const base = location.href.split('#')[0]
  return `${base}#/r/${encodeURIComponent(name)}`
}

export async function copyText(text: string): Promise<boolean> {
  try {
    await navigator.clipboard.writeText(text)
    return true
  } catch {
    const el = document.createElement('textarea')
    el.value = text
    document.body.appendChild(el)
    el.select()
    const ok = document.execCommand('copy')
    el.remove()
    return ok
  }
}
