import { Palette } from '@/api-ext'

export interface ThemePreset {
  name: string
  mode: 'dark' | 'light' | 'system'
  dark: Palette
  light: Palette
  font_family: string
  font_url: string
  border_radius: number
  button_uppercase: boolean
}

const light = (primary: string, accent: string): Palette => ({
  primary,
  secondary: '#e9e4fb',
  accent,
  error: '#e53935',
  info: '#0091ea',
  success: '#00b974',
  warning: '#f59e0b',
  background: '#f6f6fb',
  surface: '#ffffff',
  text: '#1e1b2e',
  app_bar: '#ffffff',
  app_bar_text: '#1e1b2e',
  footer: '#efeff6',
  footer_text: '#5d5875',
  drawer: '#ffffff',
})

export const presets: ThemePreset[] = [
  {
    name: 'Night Party',
    mode: 'dark',
    dark: {
      primary: '#8c5cff', secondary: '#2a2540', accent: '#ff5ca8', error: '#ff5252', info: '#3ec5ff', success: '#2ee59d', warning: '#ffb547',
      background: '#13111c', surface: '#1d1a2b', text: '#ece9f7', app_bar: '#1d1a2b', app_bar_text: '#ffffff', footer: '#13111c', footer_text: '#a29dbb', drawer: '#1a1726',
    },
    light: light('#6c3cf0', '#e83e8c'),
    font_family: 'Nunito, Roboto, Helvetica Neue, Arial, sans-serif',
    font_url: 'https://fonts.googleapis.com/css2?family=Nunito:wght@400;600;700;800&display=swap',
    border_radius: 12,
    button_uppercase: false,
  },
  {
    name: 'Classic neko',
    mode: 'dark',
    dark: {
      primary: '#3f51b5', secondary: '#424242', accent: '#ff4081', error: '#ff5252', info: '#2196f3', success: '#4caf50', warning: '#fb8c00',
      background: '#121212', surface: '#1e1e1e', text: '#ffffff', app_bar: '#3f51b5', app_bar_text: '#ffffff', footer: '#272727', footer_text: '#ffffff', drawer: '#1e1e1e',
    },
    light: { ...light('#3f51b5', '#ff4081'), app_bar: '#3f51b5', app_bar_text: '#ffffff' },
    font_family: 'Roboto, Helvetica Neue, Helvetica, Arial, sans-serif',
    font_url: 'https://fonts.googleapis.com/css?family=Roboto:100,300,400,500,700,900',
    border_radius: 4,
    button_uppercase: true,
  },
  {
    name: 'Midnight Ocean',
    mode: 'dark',
    dark: {
      primary: '#3ea6ff', secondary: '#1c2a3a', accent: '#00e5c0', error: '#ff5c5c', info: '#4fc3f7', success: '#2ee59d', warning: '#ffca28',
      background: '#0b1220', surface: '#121c2e', text: '#e6edf7', app_bar: '#0f1828', app_bar_text: '#e6edf7', footer: '#0b1220', footer_text: '#8899b0', drawer: '#0f1828',
    },
    light: light('#1976d2', '#00bfa5'),
    font_family: 'Inter, Roboto, Helvetica Neue, Arial, sans-serif',
    font_url: 'https://fonts.googleapis.com/css2?family=Inter:wght@400;500;600;800&display=swap',
    border_radius: 10,
    button_uppercase: false,
  },
  {
    name: 'Sunset',
    mode: 'dark',
    dark: {
      primary: '#ff7a45', secondary: '#3a2530', accent: '#ffcc4d', error: '#ff4d6d', info: '#4dabf7', success: '#51cf66', warning: '#ffd43b',
      background: '#1a1015', surface: '#26171f', text: '#fbeee6', app_bar: '#26171f', app_bar_text: '#fbeee6', footer: '#1a1015', footer_text: '#c9a99a', drawer: '#26171f',
    },
    light: light('#f4511e', '#ffb300'),
    font_family: 'Poppins, Roboto, Helvetica Neue, Arial, sans-serif',
    font_url: 'https://fonts.googleapis.com/css2?family=Poppins:wght@400;500;600;800&display=swap',
    border_radius: 16,
    button_uppercase: false,
  },
  {
    name: 'Forest',
    mode: 'dark',
    dark: {
      primary: '#3ddc84', secondary: '#1f2e26', accent: '#a3e635', error: '#ff6b6b', info: '#38bdf8', success: '#4ade80', warning: '#facc15',
      background: '#0d1511', surface: '#14211a', text: '#e7f5ec', app_bar: '#14211a', app_bar_text: '#e7f5ec', footer: '#0d1511', footer_text: '#8fb39c', drawer: '#14211a',
    },
    light: light('#16a34a', '#65a30d'),
    font_family: 'Rubik, Roboto, Helvetica Neue, Arial, sans-serif',
    font_url: 'https://fonts.googleapis.com/css2?family=Rubik:wght@400;500;700;800&display=swap',
    border_radius: 8,
    button_uppercase: false,
  },
  {
    name: 'Clean Light',
    mode: 'light',
    dark: {
      primary: '#7c5cff', secondary: '#2b2b33', accent: '#ff4f8b', error: '#ff5252', info: '#3ec5ff', success: '#2ee59d', warning: '#ffb547',
      background: '#111114', surface: '#1b1b20', text: '#f2f2f5', app_bar: '#1b1b20', app_bar_text: '#ffffff', footer: '#111114', footer_text: '#9a9aa6', drawer: '#1b1b20',
    },
    light: light('#5b3df5', '#ff4f8b'),
    font_family: 'Manrope, Roboto, Helvetica Neue, Arial, sans-serif',
    font_url: 'https://fonts.googleapis.com/css2?family=Manrope:wght@400;600;700;800&display=swap',
    border_radius: 14,
    button_uppercase: false,
  },
]
