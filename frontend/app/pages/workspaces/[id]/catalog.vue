<template>
  <div class="flex flex-col h-full">
    <div class="px-4 py-3 border-b flex items-center gap-3" style="border-color: var(--color-border)">
      <h1 class="font-semibold text-sm">API Reference</h1>
      <NuxtLink :to="`/workspaces/${workspaceId}`" class="text-xs text-muted hover:underline">Back to workspace</NuxtLink>
    </div>
    <CatalogBrowser :workspace-id="workspaceId" :on-open-in-builder="openInBuilder" />
  </div>
</template>

<script setup lang="ts">
import type { CatalogEndpoint } from '~/stores/catalog'

const route = useRoute()
const workspaceId = computed(() => route.params.id as string)
const tabsStore = useTabsStore()
const colStore = useCollectionsStore()

function openInBuilder(ep: CatalogEndpoint) {
  const req = colStore.requests.find(r => r.id === ep.id)
  if (req) {
    tabsStore.openRequest(req)
    colStore.setActiveRequest(req)
    navigateTo(`/workspaces/${workspaceId.value}`)
  }
}
</script>
