<template>
  <PMWorkspaceShell :workspace-id="workspaceId" :workspace-name="workspaceName">
    <DiagramsBrowser :workspace-id="workspaceId" :initial-slug="initialSlug" />
  </PMWorkspaceShell>
</template>

<script setup lang="ts">
definePageMeta({
  key: route => `workspace-diagrams-${route.params.id}`,
})

const route = useRoute()
const wsStore = useWorkspaceStore()

const workspaceId = computed(() => route.params.id as string)
const initialSlug = computed(() => {
  const slug = route.params.slug
  return Array.isArray(slug) ? slug.join('/') : slug
})
const workspaceName = computed(() => wsStore.current?.name || 'Workspace')

onMounted(async () => {
  if (!wsStore.current || wsStore.current.id !== workspaceId.value) {
    await wsStore.fetchWorkspace(workspaceId.value)
  }
  if ((wsStore.current?.type ?? 'default') !== 'pm') {
    await navigateTo(`/workspaces/${workspaceId.value}`)
  }
})
</script>
