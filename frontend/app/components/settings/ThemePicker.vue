<template>
  <div class="space-y-2">
    <label class="text-sm block">Theme</label>
    <p class="text-xs text-muted mb-2">Choose the color palette for the app interface and code editors.</p>
    <div class="grid gap-2 sm:grid-cols-2">
      <button
        v-for="theme in THEMES"
        :key="theme.id"
        type="button"
        class="theme-option rounded-md p-3 text-left transition"
        :class="themeStore.themeId === theme.id ? 'theme-option-active' : ''"
        @click="themeStore.setTheme(theme.id)"
      >
        <div class="flex items-center gap-2 mb-2">
          <span class="theme-swatch" aria-hidden="true">
            <span class="theme-swatch-bg" :style="{ background: theme.swatch.bg }" />
            <span class="theme-swatch-accent" :style="{ background: theme.swatch.accent }" />
            <span class="theme-swatch-text" :style="{ background: theme.swatch.text }" />
          </span>
          <span class="text-sm font-medium">{{ theme.label }}</span>
          <span class="theme-mode-badge">{{ theme.mode === 'light' ? 'Light' : 'Dark' }}</span>
        </div>
        <p class="text-xs text-muted">{{ theme.description }}</p>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { THEMES } from '~/constants/theme'

const themeStore = useThemeStore()
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}

.theme-option {
  border: 1px solid var(--color-border);
  background: var(--color-surface-2);
}

.theme-option:hover {
  border-color: var(--color-accent);
}

.theme-option-active {
  border-color: var(--color-accent);
  box-shadow: 0 0 0 1px var(--color-accent-muted);
}

.theme-swatch {
  display: inline-grid;
  grid-template-columns: repeat(3, 10px);
  gap: 2px;
  width: 34px;
  height: 18px;
  border-radius: var(--radius-sm);
  overflow: hidden;
  border: 1px solid var(--color-border);
}

.theme-swatch-bg,
.theme-swatch-accent,
.theme-swatch-text {
  display: block;
  height: 100%;
}

.theme-mode-badge {
  margin-left: auto;
  font-size: 0.625rem;
  font-weight: 500;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  color: var(--color-text-muted);
  opacity: 0.8;
}
</style>
