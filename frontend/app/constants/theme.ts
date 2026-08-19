export const THEME_STORAGE_KEY = 'postchi:theme'

export type ThemeMode = 'dark' | 'light'

export type ThemeId =
  | 'tokyo-night'
  | 'tokyo-night-day'
  | 'nord'
  | 'nord-light'
  | 'github-dark'
  | 'github-light'
  | 'rose-pine'
  | 'rose-pine-dawn'
  | 'kanagawa'
  | 'one-dark'

export interface ThemeDefinition {
  id: ThemeId
  label: string
  description: string
  mode: ThemeMode
  swatch: { bg: string; accent: string; text: string }
}

export const DEFAULT_THEME: ThemeId = 'github-light'

export const THEMES: ThemeDefinition[] = [
  {
    id: 'tokyo-night',
    label: 'Tokyo Night',
    description: 'Deep blue-purple with vibrant accents',
    mode: 'dark',
    swatch: { bg: '#1a1b26', accent: '#7aa2f7', text: '#a9b1d6' },
  },
  {
    id: 'tokyo-night-day',
    label: 'Tokyo Night Day',
    description: 'Soft daylight variant with muted blues',
    mode: 'light',
    swatch: { bg: '#e1e2e7', accent: '#34548a', text: '#343b58' },
  },
  {
    id: 'nord',
    label: 'Nord',
    description: 'Soft arctic palette with pastel accents',
    mode: 'dark',
    swatch: { bg: '#2e3440', accent: '#88c0d0', text: '#eceff4' },
  },
  {
    id: 'nord-light',
    label: 'Nord Light',
    description: 'Snowstorm tones with cool frost accents',
    mode: 'light',
    swatch: { bg: '#eceff4', accent: '#5e81ac', text: '#2e3440' },
  },
  {
    id: 'github-dark',
    label: 'GitHub Dark',
    description: 'GitHub default dark with Primer colors',
    mode: 'dark',
    swatch: { bg: '#0d1117', accent: '#2f81f7', text: '#e6edf3' },
  },
  {
    id: 'github-light',
    label: 'GitHub Light',
    description: 'Clean GitHub light with blue accents',
    mode: 'light',
    swatch: { bg: '#ffffff', accent: '#0969da', text: '#1f2328' },
  },
  {
    id: 'rose-pine',
    label: 'Rosé Pine',
    description: 'Elegant muted violet with warm accents',
    mode: 'dark',
    swatch: { bg: '#191724', accent: '#9ccfd8', text: '#e0def4' },
  },
  {
    id: 'rose-pine-dawn',
    label: 'Rosé Pine Dawn',
    description: 'Warm light palette with soft pastels',
    mode: 'light',
    swatch: { bg: '#faf4ed', accent: '#56949f', text: '#575279' },
  },
  {
    id: 'kanagawa',
    label: 'Kanagawa',
    description: 'Japanese ink painting inspired dark tones',
    mode: 'dark',
    swatch: { bg: '#1f1f28', accent: '#7e9cd8', text: '#dcd7ba' },
  },
  {
    id: 'one-dark',
    label: 'One Dark',
    description: 'Atom One Dark with balanced syntax colors',
    mode: 'dark',
    swatch: { bg: '#282c34', accent: '#61afef', text: '#abb2bf' },
  },
]

const themeById = new Map(THEMES.map((theme) => [theme.id, theme]))

export function isThemeId(value: string | null | undefined): value is ThemeId {
  return themeById.has(value as ThemeId)
}

export function getThemeMode(themeId: ThemeId): ThemeMode {
  return themeById.get(themeId)?.mode ?? 'dark'
}
