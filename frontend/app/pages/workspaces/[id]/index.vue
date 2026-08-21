<template>
  <WorkspaceShell
    v-if="workspace && (workspace.type === 'default' || !workspace.type)"
    :workspace-id="workspaceId"
    :workspace-name="workspace.name"
    :loading="loading"
  />
  <TesterWorkspaceShell
    v-else-if="workspace?.type === 'tester'"
    :workspace-id="workspaceId"
    :workspace-name="workspace!.name"
  />
  <div v-else-if="workspace?.type === 'pm'" class="h-screen flex items-center justify-center text-muted text-sm">
    Redirecting…
  </div>
  <div v-else class="h-screen flex items-center justify-center" style="color: var(--text-secondary)">
    Loading workspace…
  </div>
</template>

<script setup lang="ts">
definePageMeta({
  key: route => `workspace-${route.params.id}`,
})

const route = useRoute()
const router = useRouter()
const wsStore = useWorkspaceStore()
const colStore = useCollectionsStore()
const envStore = useEnvironmentsStore()
const histStore = useHistoryStore()
const tabsStore = useTabsStore()
const execStore = useExecutionStore()

const workspaceId = computed(() => route.params.id as string)
const workspace = computed(() => wsStore.current)
const loading = ref(true)

async function loadWorkspace(id: string) {
  loading.value = true
  try {
    const ws = await wsStore.fetchWorkspace(id)
    if (ws.type === 'pm') {
      await router.replace(`/workspaces/${id}/diagrams`)
      return
    }
    if (ws.type === 'tester') {
      loading.value = false
      await Promise.all([
        colStore.fetchCollections(id),
        colStore.fetchAllRequests(id),
      ])
      return
    }
    loading.value = false
    await Promise.all([
      colStore.fetchCollections(id),
      colStore.fetchAllRequests(id),
      envStore.fetch(id),
      histStore.fetch(id),
    ])
    envStore.loadActive()
    await envStore.hydrateActive()
  } catch (err) {
    loading.value = false
    console.error('Failed to load workspace data', err)
  }
}

onMounted(() => {
  void loadWorkspace(workspaceId.value)
})

watch(workspaceId, (id) => {
  void loadWorkspace(id)
})

onUnmounted(() => {
  tabsStore.clear()
  execStore.clear()
  colStore.collections = []
  colStore.requests = []
})
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}
</style>
