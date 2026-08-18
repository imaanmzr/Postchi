import { defineStore } from 'pinia'
import { DEFAULT_THEME, getThemeMode, isThemeId, THEME_STORAGE_KEY, type ThemeId } from '~/constants/theme'

export const useThemeStore = defineStore('theme', {
  state: () => ({
    themeId: DEFAULT_THEME as ThemeId,
  }),
  actions: {
    init() {
      if (!import.meta.client) return
      const stored = localStorage.getItem(THEME_STORAGE_KEY)
      if (isThemeId(stored)) this.themeId = stored
      this.apply()
    },
    setTheme(themeId: ThemeId) {
      this.themeId = themeId
      if (import.meta.client) {
        localStorage.setItem(THEME_STORAGE_KEY, themeId)
        this.apply()
      }
    },
    apply() {
      if (!import.meta.client) return
      document.documentElement.dataset.theme = this.themeId
      document.documentElement.style.colorScheme = getThemeMode(this.themeId)
    },
  },
})
