<template>
  <WorkspaceShell
    v-if="workspace"
    :workspace-id="workspaceId"
    :workspace-name="workspace.name"
    :loading="loading"
  />
  <div v-else class="h-screen flex items-center justify-center" style="color: var(--text-secondary)">
    Loading workspace…
  </div>
</template>

<script setup lang="ts">
const route = useRoute()
const wsStore = useWorkspaceStore()
const colStore = useCollectionsStore()
const envStore = useEnvironmentsStore()
const histStore = useHistoryStore()
const tabsStore = useTabsStore()
const execStore = useExecutionStore()

const workspaceId = computed(() => route.params.id as string)
const workspace = computed(() => wsStore.current)
const loading = ref(true)

onMounted(async () => {
  try {
    await wsStore.fetchWorkspace(workspaceId.value)
    loading.value = false
    await Promise.all([
      colStore.fetchCollections(workspaceId.value),
      colStore.fetchAllRequests(workspaceId.value),
      envStore.fetch(workspaceId.value),
      histStore.fetch(workspaceId.value),
    ])
    envStore.loadActive()
    await envStore.hydrateActive()
  } catch (err) {
    loading.value = false
    console.error('Failed to load workspace data', err)
  }
})

onUnmounted(() => {
  tabsStore.clear()
  execStore.clear()
  colStore.collections = []
  colStore.requests = []
})
</script>
