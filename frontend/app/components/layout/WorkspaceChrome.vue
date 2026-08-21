<template>
  <header
    class="flex items-center gap-2 px-4 h-12 shrink-0 border-b"
    style="background: var(--color-surface-1); border-color: var(--color-border); height: var(--toolbar-height)"
  >
    <PostchiLogo variant="mark" :height="16" class="text-muted" title="Postchi" />

    <div class="h-4 w-px shrink-0" style="background: var(--color-border)" />

    <NuxtLink
      to="/workspaces"
      class="inline-flex items-center gap-1.5 text-xs font-medium tracking-tight text-muted hover:text-default transition shrink-0"
      title="Back to workspaces"
    >
      <ArrowLeft :size="14" :stroke-width="2" aria-hidden="true" />
      <span>Workspaces</span>
    </NuxtLink>

    <div class="h-4 w-px shrink-0" style="background: var(--color-border)" />

    <h1 class="font-semibold text-sm tracking-tight truncate max-w-[200px]">{{ workspaceName }}</h1>

    <span
      v-if="typeBadge"
      class="text-[10px] uppercase tracking-wide px-2 py-0.5 rounded border shrink-0"
      style="border-color: var(--color-border); color: var(--color-text-muted)"
    >{{ typeBadge }}</span>

    <nav v-if="navItems.length" class="flex items-center gap-1 ml-2">
      <NuxtLink
        v-for="item in navItems"
        :key="item.to"
        :to="item.to"
        class="text-xs px-2.5 py-1 rounded transition"
        :class="isNavActive(item) ? 'font-medium' : 'text-muted hover:text-default'"
        :style="isNavActive(item) ? { background: 'var(--color-surface-2)' } : undefined"
      >
        {{ item.label }}
      </NuxtLink>
    </nav>

    <div class="flex-1 min-w-2" />

    <slot name="actions" />

    <Button class="text-xs shrink-0 inline-flex items-center gap-1.5" title="Team members" @click="openSettings('team')">
      <Users :size="14" :stroke-width="2" aria-hidden="true" />
      <span>Team</span>
      <span v-if="memberCount" class="opacity-70">({{ memberCount }})</span>
    </Button>

    <div class="inline-flex items-center gap-1.5 shrink-0">
      <Palette :size="14" :stroke-width="2" class="opacity-60 shrink-0" aria-hidden="true" />
      <select
        :value="themeStore.themeId"
        class="ui-input text-xs max-w-[168px]"
        title="Theme"
        aria-label="Theme"
        @change="themeStore.setTheme(($event.target as HTMLSelectElement).value as ThemeId)"
      >
        <option v-for="t in THEMES" :key="t.id" :value="t.id">{{ t.label }}</option>
      </select>
    </div>

    <Button class="text-xs shrink-0 inline-flex items-center gap-1.5" title="Workspace settings" @click="openSettings('general')">
      <Settings :size="14" :stroke-width="2" aria-hidden="true" />
      <span>Settings</span>
    </Button>
  </header>

  <WorkspaceSettingsModal
    v-if="showSettings"
    :workspace-id="workspaceId"
    :initial-tab="settingsTab"
    @close="showSettings = false"
  />
</template>

<script setup lang="ts">
import { ArrowLeft, Palette, Settings, Users } from 'lucide-vue-next'
import { THEMES, type ThemeId } from '~/constants/theme'
import { workspaceTypeLabel } from '~/utils/workspaceType'

export type ChromeNavItem = {
  label: string
  to: string
  match?: 'exact' | 'prefix'
}

const props = defineProps<{
  workspaceId: string
  workspaceName: string
  workspaceType?: string
  navItems?: ChromeNavItem[]
}>()

const route = useRoute()
const wsStore = useWorkspaceStore()
const themeStore = useThemeStore()

const showSettings = ref(false)
const settingsTab = ref<'general' | 'team' | 'shares' | 'api-sync' | 'documentation'>('general')

const memberCount = computed(() => wsStore.members.length)
const typeBadge = computed(() => props.workspaceType ? workspaceTypeLabel(props.workspaceType) : '')
const navItems = computed(() => props.navItems ?? [])

onMounted(() => {
  void wsStore.fetchMembers(props.workspaceId)
})

function openSettings(tab: typeof settingsTab.value) {
  settingsTab.value = tab
  showSettings.value = true
}

function isNavActive(item: ChromeNavItem) {
  if (item.match === 'exact') return route.path === item.to
  return route.path === item.to || route.path.startsWith(`${item.to}/`)
}
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}
</style>
