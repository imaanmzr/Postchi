<template>
  <div class="flex flex-col h-full min-h-0" @keydown="onKeydown">
    <div class="p-3 border-b shrink-0" style="border-color: var(--color-border)">
      <input
        ref="searchInput"
        v-model="searchQuery"
        type="search"
        placeholder="Search docs…"
        class="ui-input w-full text-xs"
        @keydown.stop
      />
    </div>

    <div
      ref="scrollParent"
      class="flex-1 min-h-0 overflow-auto ui-scroll-y docs-tree-scroll"
      tabindex="0"
      @focus="focused = true"
      @blur="focused = false"
    >
      <div v-if="loading" class="p-3 space-y-2">
        <div v-for="i in 8" :key="i" class="docs-tree-skeleton h-6 rounded" />
      </div>
      <p v-else-if="!visibleRows.length" class="p-4 text-xs text-muted">
        No docs yet. Connect a git documentation source in workspace settings.
      </p>
      <div
        v-else
        :style="{ height: `${totalSize}px`, position: 'relative', width: '100%' }"
      >
        <div
          v-for="item in virtualItems"
          :key="String(item.key)"
          :style="{
            position: 'absolute',
            top: 0,
            left: 0,
            width: '100%',
            height: `${item.size}px`,
            transform: `translateY(${item.start}px)`,
          }"
        >
          <DocsTreeNode
            :row="visibleRows[item.index]!"
            :active="visibleRows[item.index]?.doc?.slug === activeSlug"
            @click="onRowClick"
            @toggle="toggleFolder"
            @contextmenu="openContextMenu"
          />
        </div>
      </div>
    </div>

    <div
      v-if="contextMenu"
      class="docs-tree-context fixed z-50 min-w-[160px] rounded-md border py-1 text-xs shadow-lg"
      :style="{ left: `${contextMenu.x}px`, top: `${contextMenu.y}px`, borderColor: 'var(--color-border)', background: 'var(--color-surface-1)' }"
      @click.stop
    >
      <button
        v-for="item in contextItems"
        :key="item.id"
        type="button"
        class="w-full text-left px-3 py-1.5 hover:bg-surface-2 transition"
        @click="item.action()"
      >
        {{ item.label }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useVirtualizer } from '@tanstack/vue-virtual'
import { useLocalStorage } from '@vueuse/core'
import type { DocSummary } from '~/utils/docsTree'
import {
  ancestorFolderPaths,
  buildDocTree,
  filterTreeForSearch,
  flattenTree,
  fuzzySearchDocs,
  // Aliased: a plain `DocsTreeNode` import would shadow the auto-imported
  // <DocsTreeNode> component in the template and break rendering.
  type DocsTreeNode as DocsTreeNodeData,
  type FlatTreeRow,
} from '~/utils/docsTree'

const props = defineProps<{
  workspaceId: string
  summaries: DocSummary[]
  activeSlug?: string | null
  loading?: boolean
}>()

const emit = defineEmits<{
  select: [slug: string]
  'create-local': [folderPath: string]
  'delete-doc': [doc: DocSummary]
  'reveal-path': [path: string]
}>()

const searchQuery = defineModel<string>('search', { default: '' })
const searchInput = ref<HTMLInputElement>()
const scrollParent = ref<HTMLElement>()
const focused = ref(false)
const selectedIndex = ref(0)

const expandedStorage = useLocalStorage<Record<string, boolean>>(
  () => `postchi:docs-tree-expanded:${props.workspaceId}`,
  {},
)

const fuzzy = computed(() => fuzzySearchDocs(props.summaries, searchQuery.value))

const treeRoots = computed(() => {
  const roots = buildDocTree(props.summaries)
  if (!searchQuery.value.trim()) return roots
  return filterTreeForSearch(roots, fuzzy.value.matchingSlugs, fuzzy.value.expandPaths)
})

const expandedForView = computed(() => {
  if (!searchQuery.value.trim()) return expandedStorage.value
  const merged = { ...expandedStorage.value }
  for (const path of fuzzy.value.expandPaths) {
    merged[path] = true
  }
  return merged
})

const visibleRows = computed((): FlatTreeRow[] => {
  const rows = flattenTree(treeRoots.value, expandedForView.value)
  if (!searchQuery.value.trim()) return rows
  return rows.map((row) => {
    if (row.type === 'file' && row.doc) {
      return { ...row, matchRanges: fuzzy.value.highlights.get(row.doc.slug) }
    }
    return row
  })
})

const virtualizer = useVirtualizer(computed(() => ({
  count: visibleRows.value.length,
  getScrollElement: () => scrollParent.value ?? null,
  estimateSize: () => 28,
  overscan: 12,
})))

const virtualItems = computed(() => virtualizer.value.getVirtualItems())
const totalSize = computed(() => virtualizer.value.getTotalSize())

watch(() => props.summaries.length, (count) => {
  if (count > 0 && Object.keys(expandedStorage.value).length === 0) {
    const roots = buildDocTree(props.summaries)
    const next = { ...expandedStorage.value }
    const expandNode = (node: DocsTreeNodeData, depth: number) => {
      if (node.type === 'folder' && depth < 3) {
        next[node.path] = true
        for (const child of node.children) expandNode(child, depth + 1)
      }
    }
    for (const node of roots) expandNode(node, 0)
    expandedStorage.value = next
  }
}, { immediate: true })

watch(() => props.activeSlug, (slug) => {
  if (!slug) return
  const summary = props.summaries.find(d => d.slug === slug)
  if (!summary?.source_path) return
  const paths = ancestorFolderPaths(summary.source_path)
  const next = { ...expandedStorage.value }
  for (const p of paths) next[p] = true
  expandedStorage.value = next
})

function toggleFolder(path: string) {
  const isOpen = expandedStorage.value[path] === true
  expandedStorage.value = {
    ...expandedStorage.value,
    [path]: !isOpen,
  }
}

function onRowClick(row: FlatTreeRow) {
  if (row.doc) {
    selectedIndex.value = visibleRows.value.findIndex(r => r.id === row.id)
    emit('select', row.doc.slug)
  }
}

function expandAll() {
  const next = { ...expandedStorage.value }
  for (const row of visibleRows.value) {
    if (row.type === 'folder') next[row.path] = true
  }
  expandedStorage.value = next
}

function collapseAll() {
  expandedStorage.value = {}
}

const contextMenu = ref<{ x: number, y: number, row: FlatTreeRow } | null>(null)

const contextItems = computed(() => {
  const row = contextMenu.value?.row
  if (!row) return []
  const items: { id: string, label: string, action: () => void }[] = []
  if (row.type === 'file' && row.doc) {
    items.push({
      id: 'open',
      label: 'Open',
      action: () => { emit('select', row.doc!.slug); contextMenu.value = null },
    })
    items.push({
      id: 'copy-path',
      label: 'Copy path',
      action: () => {
        void navigator.clipboard.writeText(row.doc!.source_path)
        contextMenu.value = null
      },
    })
    items.push({
      id: 'copy-wikilink',
      label: 'Copy wikilink',
      action: () => {
        void navigator.clipboard.writeText(`[[${row.doc!.slug}]]`)
        contextMenu.value = null
      },
    })
    if (row.doc.is_local) {
      items.push({
        id: 'delete',
        label: 'Delete…',
        action: () => { emit('delete-doc', row.doc!); contextMenu.value = null },
      })
    }
  }
  if (row.type === 'folder') {
    items.push({
      id: 'new-local',
      label: 'New local doc…',
      action: () => { emit('create-local', row.path); contextMenu.value = null },
    })
  }
  items.push({ id: 'expand', label: 'Expand all', action: () => { expandAll(); contextMenu.value = null } })
  items.push({ id: 'collapse', label: 'Collapse all', action: () => { collapseAll(); contextMenu.value = null } })
  return items
})

function openContextMenu(event: MouseEvent, row: FlatTreeRow) {
  contextMenu.value = { x: event.clientX, y: event.clientY, row }
}

function onKeydown(e: KeyboardEvent) {
  if (!focused.value && document.activeElement !== searchInput.value) return
  const rows = visibleRows.value
  if (!rows.length) return

  if (e.key === 'ArrowDown') {
    e.preventDefault()
    selectedIndex.value = Math.min(selectedIndex.value + 1, rows.length - 1)
    scrollToSelected()
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    selectedIndex.value = Math.max(selectedIndex.value - 1, 0)
    scrollToSelected()
  } else if (e.key === 'Home') {
    e.preventDefault()
    selectedIndex.value = 0
    scrollToSelected()
  } else if (e.key === 'End') {
    e.preventDefault()
    selectedIndex.value = rows.length - 1
    scrollToSelected()
  } else if (e.key === 'Enter') {
    e.preventDefault()
    const row = rows[selectedIndex.value]
    if (!row) return
    if (row.type === 'folder') toggleFolder(row.path)
    else if (row.doc) emit('select', row.doc.slug)
  } else if (e.key === 'ArrowRight') {
    e.preventDefault()
    const row = rows[selectedIndex.value]
    if (row.type === 'folder') expandedStorage.value = { ...expandedStorage.value, [row.path]: true }
  } else if (e.key === 'ArrowLeft') {
    e.preventDefault()
    const row = rows[selectedIndex.value]
    if (row?.type === 'folder') expandedStorage.value = { ...expandedStorage.value, [row.path]: false }
  }
}

function scrollToSelected() {
  virtualizer.value.scrollToIndex(selectedIndex.value, { align: 'auto' })
}

function revealPath(path: string) {
  const parts = path.split('/').filter(Boolean)
  const next = { ...expandedStorage.value }
  for (let i = 0; i < parts.length - 1; i++) {
    next[parts.slice(0, i + 1).join('/')] = true
  }
  expandedStorage.value = next
  const idx = visibleRows.value.findIndex(r => r.path === path || r.doc?.slug === path)
  if (idx >= 0) {
    selectedIndex.value = idx
    nextTick(() => scrollToSelected())
  }
}

onMounted(() => {
  document.addEventListener('click', () => { contextMenu.value = null })
})

defineExpose({ revealPath, focusSearch: () => searchInput.value?.focus() })
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}
.docs-tree-skeleton {
  background: linear-gradient(90deg, var(--color-surface-2) 25%, var(--color-surface-1) 50%, var(--color-surface-2) 75%);
  background-size: 200% 100%;
  animation: docs-shimmer 1.2s infinite;
}
@keyframes docs-shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}
.docs-tree-context button:hover {
  background: var(--color-surface-2);
}
</style>
