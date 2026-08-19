<template>
  <div v-if="hasVisibleContent">
    <div
      class="tree-folder-row catalog-tree-folder"
      :style="{ paddingLeft: `${depth * 12 + 4}px` }"
      @click="onFolderRowClick"
    >
      <TreeRowIcons
        show-chevron
        show-folder
        :expanded="expanded"
        @toggle="onFolderRowClick"
      />
      <span class="tree-folder-name flex-1 truncate">{{ node.name }}</span>
      <span v-if="stats" class="text-[10px] text-muted shrink-0 tabular-nums">
        {{ stats.documented_count }}/{{ stats.request_count }}
      </span>
    </div>

    <div v-show="expanded">
      <CatalogTreeNode
        v-for="child in node.children"
        :key="child.id"
        :node="child"
        :depth="depth + 1"
        :endpoints-by-collection="endpointsByCollection"
        :collection-stats="collectionStats"
        :expanded-ids="expandedIds"
        :selected-id="selectedId"
        @toggle="$emit('toggle', $event)"
        @select="$emit('select', $event)"
      />

      <button
        v-for="ep in sortedEndpoints"
        :key="ep.id"
        type="button"
        class="catalog-tree-request w-full flex items-center gap-1.5 text-left text-xs"
        :class="{ 'catalog-tree-request--active': ep.id === selectedId }"
        :style="{ paddingLeft: `${(depth + 1) * 12 + 8}px` }"
        @click="$emit('select', ep)"
      >
        <MethodBadge :method="ep.method" class="shrink-0 scale-90" />
        <span class="truncate flex-1">{{ ep.name }}</span>
        <span
          v-if="!ep.docs_complete"
          class="w-1.5 h-1.5 rounded-full bg-amber-500 shrink-0"
          title="Undocumented"
        />
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { CatalogCollection, CatalogEndpoint } from '~/stores/catalog'
import type { TreeNode } from '~/stores/collections'

const props = defineProps<{
  node: TreeNode
  depth: number
  endpointsByCollection: Map<string, CatalogEndpoint[]>
  collectionStats: Map<string, CatalogCollection>
  expandedIds: Record<string, boolean>
  selectedId?: string | null
}>()

const emit = defineEmits<{
  toggle: [id: string]
  select: [ep: CatalogEndpoint]
}>()

const expanded = computed(() => props.expandedIds[props.node.id] === true)

const endpoints = computed(() => props.endpointsByCollection.get(props.node.id) || [])

const sortedEndpoints = computed(() =>
  [...endpoints.value].sort((a, b) => a.name.localeCompare(b.name)),
)

const stats = computed(() => props.collectionStats.get(props.node.id))

function folderHasVisible(node: TreeNode): boolean {
  if ((props.endpointsByCollection.get(node.id) || []).length > 0) return true
  return node.children.some(child => folderHasVisible(child))
}

const hasVisibleContent = computed(() => folderHasVisible(props.node))

function onFolderRowClick() {
  emit('toggle', props.node.id)
}
</script>

<style scoped>
.catalog-tree-request {
  padding-top: 0.3rem;
  padding-bottom: 0.3rem;
  padding-right: 0.5rem;
  color: var(--color-text-muted);
  transition: background-color var(--duration-fast) var(--ease-out), color var(--duration-fast) var(--ease-out);
}
.catalog-tree-request:hover {
  background: var(--color-surface-2);
  color: var(--color-text);
}
.catalog-tree-request--active {
  background: var(--color-surface-2);
  color: var(--color-text);
  box-shadow: inset 2px 0 0 var(--color-accent);
}
</style>
