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

    <Button class="text-xs shrink-0 inline-flex items-center gap-1.5" title="Search requests" @click="emit('open-collections')">
      <Search :size="14" :stroke-width="2" aria-hidden="true" />
      <span>Search</span>
    </Button>

    <div class="flex-1 min-w-2" />

    <Button class="text-xs shrink-0 inline-flex items-center gap-1.5" title="Open collection runner" @click="emit('open-runner')">
      <Play :size="14" :stroke-width="2" aria-hidden="true" />
      <span>Runner</span>
    </Button>

    <div class="inline-flex items-center gap-1.5 shrink-0" title="Active environment">
      <Globe :size="14" :stroke-width="2" class="opacity-60 shrink-0" aria-hidden="true" />
      <select
        :value="envStore.activeId || ''"
        class="ui-input text-xs max-w-[160px]"
        aria-label="Active environment"
        @change="envStore.setActive(($event.target as HTMLSelectElement).value || null)"
      >
      <option value="">No environment</option>
      <optgroup v-for="group in envGroups" :key="group.stage" :label="group.label">
        <option v-for="env in group.envs" :key="env.id" :value="env.id">{{ env.name }}</option>
      </optgroup>
    </select>
    </div>

    <Button class="text-xs shrink-0 inline-flex items-center gap-1.5" title="Manage environments" @click="showEnvManager = true">
      <Layers :size="14" :stroke-width="2" aria-hidden="true" />
      <span>Env</span>
    </Button>
    <Button class="text-xs shrink-0 inline-flex items-center gap-1.5" title="API reference catalog" @click="navigateTo(`/workspaces/${workspaceId}/catalog`)">
      <BookOpen :size="14" :stroke-width="2" aria-hidden="true" />
      <span>API Docs</span>
    </Button>
    <Button class="text-xs shrink-0 inline-flex items-center gap-1.5" title="Documentation workspace" @click="navigateTo(`/workspaces/${workspaceId}/docs`)">
      <FileText :size="14" :stroke-width="2" aria-hidden="true" />
      <span>Docs</span>
    </Button>

    <Button class="text-xs shrink-0 inline-flex items-center gap-1.5" title="Import / export" @click="showImport = true">
      <Upload :size="14" :stroke-width="2" aria-hidden="true" />
      <span>Import</span>
    </Button>
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
        <option v-for="theme in THEMES" :key="theme.id" :value="theme.id">{{ theme.label }}</option>
      </select>
    </div>

    <Button class="text-xs shrink-0 inline-flex items-center gap-1.5" title="Workspace settings" @click="openSettings('general')">
      <Settings :size="14" :stroke-width="2" aria-hidden="true" />
      <span>Settings</span>
    </Button>

    <Button class="text-xs shrink-0 inline-flex items-center gap-1.5" title="Account settings" @click="showAccount = true">
      <User :size="14" :stroke-width="2" aria-hidden="true" />
      <span>Account</span>
    </Button>

    <Button class="text-xs shrink-0 inline-flex items-center gap-1.5" title="Sign out" @click="signOut">
      <LogOut :size="14" :stroke-width="2" aria-hidden="true" />
      <span>Sign out</span>
    </Button>

    <CollectionMenu
      :has-collection="hasActiveCollection"
      @history="emit('open-history')"
      @variables="emit('open-variables')"
      @openapi="emit('open-openapi')"
      @collection-settings="emit('open-collection-settings')"
    />

    <EnvironmentManager v-if="showEnvManager" :workspace-id="workspaceId" @close="showEnvManager = false" />
    <ImportExportModal v-if="showImport" :workspace-id="workspaceId" @close="showImport = false" />
    <WorkspaceSettingsModal
      v-if="showSettings"
      :workspace-id="workspaceId"
      :initial-tab="settingsTab"
      @close="showSettings = false"
    />
    <AccountSettingsModal v-if="showAccount" @close="showAccount = false" />
  </header>
</template>

<script setup lang="ts">
import {
  ArrowLeft,
  BookOpen,
  FileText,
  Globe,
  Layers,
  LogOut,
  Palette,
  Play,
  Search,
  Settings,
  Upload,
  User,
  Users,
} from 'lucide-vue-next'
import { THEMES, type ThemeId } from '~/constants/theme'

const props = defineProps<{ workspaceId: string; workspaceName: string; hasActiveCollection?: boolean }>()
const auth = useAuthStore()
const router = useRouter()
const themeStore = useThemeStore()
const emit = defineEmits<{
  'open-collections': []
  'open-runner': []
  'open-history': []
  'open-variables': []
  'open-openapi': []
  'open-collection-settings': []
}>()

const envStore = useEnvironmentsStore()
const wsStore = useWorkspaceStore()
const showEnvManager = ref(false)
const showImport = ref(false)
const showSettings = ref(false)
const showAccount = ref(false)
const settingsTab = ref<'general' | 'team' | 'shares' | 'api-sync' | 'documentation'>('general')

const stageLabels: Record<string, string> = {
  local: 'Local', dev: 'Dev', uat: 'UAT', staging: 'Staging', prod: 'Prod', custom: 'Custom',
}

const envGroups = computed(() => {
  const groups: Record<string, typeof envStore.environments> = {}
  for (const env of envStore.environments) {
    const stage = env.stage || 'custom'
    if (!groups[stage]) groups[stage] = []
    groups[stage].push(env)
  }
  return Object.entries(groups).map(([stage, envs]) => ({
    stage,
    label: stageLabels[stage] || stage,
    envs,
  }))
})

const memberCount = computed(() => wsStore.members.length)

onMounted(async () => {
  wsStore.fetchMembers(props.workspaceId)
  await envStore.fetch(props.workspaceId)
})

function openSettings(tab: 'general' | 'team' | 'shares' | 'api-sync' | 'documentation') {
  settingsTab.value = tab
  showSettings.value = true
}

async function signOut() {
  await auth.logout()
  await router.push('/login')
}
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}
.hover\:text-default:hover {
  color: var(--color-text);
}
</style>
