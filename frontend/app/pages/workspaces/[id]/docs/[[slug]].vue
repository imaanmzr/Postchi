<template>
  <PMWorkspaceShell
    v-if="usePmShell"
    :workspace-id="workspaceId"
    :workspace-name="workspaceName"
  >
    <DocsBrowser :workspace-id="workspaceId" workspace-type="pm" embedded />
  </PMWorkspaceShell>
  <div v-else class="h-screen flex flex-col overflow-hidden">
    <DocsBrowser :workspace-id="workspaceId" />
  </div>
</template>

<script setup lang="ts">
definePageMeta({
  key: route => `workspace-docs-${route.params.id}`,
})

const route = useRoute()
const wsStore = useWorkspaceStore()

const workspaceId = computed(() => route.params.id as string)
const workspaceName = computed(() => wsStore.current?.name || 'Workspace')

const usePmShell = computed(() =>
  wsStore.current?.id === workspaceId.value && wsStore.current.type === 'pm',
)

async function ensureWorkspace(id: string) {
  try {
    if (wsStore.current?.id !== id) {
      await wsStore.fetchWorkspace(id)
    }
    if (wsStore.current?.type === 'tester') {
      await navigateTo(`/workspaces/${id}`)
    }
  } catch (err) {
    console.error('[docs] failed to load workspace', err)
  }
}

onMounted(() => {
  void ensureWorkspace(workspaceId.value)
})

watch(workspaceId, (id) => {
  void ensureWorkspace(id)
})
</script>
