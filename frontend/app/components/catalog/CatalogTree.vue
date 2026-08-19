<template>
  <div class="collection-tree catalog-tree flex flex-col h-full text-sm">
    <div v-if="loading" class="p-3 text-xs text-muted">Loading endpoints…</div>
    <p v-else-if="!visibleRoots.length" class="p-3 text-sm text-center text-muted">
      No endpoints match the current filters.
    </p>
    <div v-else class="flex-1 ui-scroll-y p-1">
      <CatalogTreeNode
        v-for="node in visibleRoots"
        :key="node.id"
        :node="node"
        :depth="0"
        :endpoints-by-collection="endpointsByCollection"
        :collection-stats="collectionStats"
        :expanded-ids="expandedIds"
        :selected-id="selectedId"
        @toggle="toggleExpand"
        @select="$emit('select', $event)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import type { CatalogCollection, CatalogEndpoint } from '~/stores/catalog'
import type { TreeNode } from '~/stores/collections'

const props = defineProps<{
  workspaceId: string
  tree: TreeNode[]
  endpoints: CatalogEndpoint[]
  collections: CatalogCollection[]
  selectedId?: string | null
  loading?: boolean
}>()

defineEmits<{
  select: [ep: CatalogEndpoint]
}>()

const colStore = useCollectionsStore()

const expandedStorageKey = computed(() => `postchi:catalog-tree-expanded:${props.workspaceId}`)

function loadExpandedIds(): Record<string, boolean> {
  if (!import.meta.client) return {}
  try {
    const raw = localStorage.getItem(expandedStorageKey.value)
    if (raw) return JSON.parse(raw) as Record<string, boolean>
  } catch { /* ignore */ }
  return {}
}

const expandedIds = ref<Record<string, boolean>>(loadExpandedIds())

watch(expandedIds, (value) => {
  if (!import.meta.client) return
  localStorage.setItem(expandedStorageKey.value, JSON.stringify(value))
}, { deep: true })

const endpointsByCollection = computed(() => {
  const map = new Map<string, CatalogEndpoint[]>()
  for (const ep of props.endpoints) {
    const list = map.get(ep.collection_id) || []
    list.push(ep)
    map.set(ep.collection_id, list)
  }
  return map
})

const collectionStats = computed(() => {
  const map = new Map<string, CatalogCollection>()
  for (const col of props.collections) {
    map.set(col.id, col)
  }
  return map
})

function folderHasVisible(node: TreeNode): boolean {
  if ((endpointsByCollection.value.get(node.id) || []).length > 0) return true
  return node.children.some(child => folderHasVisible(child))
}

const visibleRoots = computed(() => props.tree.filter(node => folderHasVisible(node)))

function toggleExpand(id: string) {
  expandedIds.value = {
    ...expandedIds.value,
    [id]: !expandedIds.value[id],
  }
}

function expandAncestors(collectionId: string) {
  const next = { ...expandedIds.value }
  const visited = new Set<string>()
  let cur = colStore.collections.find(c => c.id === collectionId)
  while (cur && !visited.has(cur.id)) {
    visited.add(cur.id)
    next[cur.id] = true
    if (!cur.parent_id) break
    cur = colStore.collections.find(c => c.id === cur!.parent_id)
  }
  expandedIds.value = next
}

watch(() => props.selectedId, (id) => {
  if (!id) return
  const ep = props.endpoints.find(e => e.id === id)
  if (ep) expandAncestors(ep.collection_id)
}, { immediate: true })

onMounted(() => {
  if (Object.keys(expandedIds.value).length === 0 && props.endpoints.length > 0) {
    const next = { ...expandedIds.value }
    for (const ep of props.endpoints) {
      let cur = colStore.collections.find(c => c.id === ep.collection_id)
      const visited = new Set<string>()
      while (cur && !visited.has(cur.id)) {
        visited.add(cur.id)
        next[cur.id] = true
        if (!cur.parent_id) break
        cur = colStore.collections.find(c => c.id === cur!.parent_id)
      }
    }
    expandedIds.value = next
  }
})
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}
</style>
