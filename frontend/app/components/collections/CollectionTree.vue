<template>
  <div class="flex flex-col h-full text-sm collection-tree">
    <div class="p-2 border-b flex flex-wrap gap-1 items-center" style="border-color: var(--border)">
      <template v-if="selectionMode">
        <label class="flex items-center gap-1.5 text-xs cursor-pointer shrink-0">
          <input
            ref="selectAllCheckbox"
            type="checkbox"
            class="tree-select-checkbox"
            :checked="allSelected"
            @change="toggleSelectAll"
          />
          <span>All</span>
        </label>
        <Button
          class="text-xs !text-[var(--method-delete)]"
          :disabled="!selectedCount || deleting"
          @click="askBulkDelete"
        >
          Delete{{ selectedCount ? ` (${selectedCount})` : '' }}
        </Button>
        <Button class="text-xs ml-auto" @click="exitSelectionMode">Cancel</Button>
      </template>
      <template v-else>
        <Button class="text-xs" :disabled="creating" @click="createFolder">+ Folder</Button>
        <Button class="text-xs" :disabled="creating" @click="createRequest">+ Request</Button>
        <Button class="text-xs ml-auto" :disabled="!hasTreeItems" @click="enterSelectionMode">Select</Button>
      </template>
    </div>
    <div ref="treeScroll" class="flex-1 ui-scroll-y p-1" @click="onTreeBackgroundClick">
      <button
        type="button"
        class="tree-workspace-root"
        :class="{ 'tree-workspace-root--selected': creationTarget === 'workspace' }"
        @click.stop="selectWorkspaceRoot"
      >
        <span class="tree-workspace-label">Workspace</span>
        <span class="tree-workspace-hint">Top level</span>
      </button>
      <p v-if="!tree.length" class="p-3 text-sm text-center tree-muted">
        No collections yet. Select Workspace above, then create a folder or import one.
      </p>
      <div
        v-else
        class="tree-drop-container tree-drop-container--root"
        data-collection-id=""
      >
        <VueDraggable
          v-model="rootNodes"
          tag="div"
          class="tree-draggable-list"
          :group="{ name: 'postchi-tree', pull: true, put: true }"
          :animation="dragLocked ? 0 : 120"
          :disabled="selectionMode"
          handle=".tree-drag-handle"
          ghost-class="tree-drag-ghost"
          chosen-class="tree-drag-chosen"
          drag-class="tree-drag-dragging"
          :swap-threshold="0.5"
          :move="onRootMove"
          @start="onDragStart"
          @move="onRootListMove"
          @end="onDragComplete"
        >
          <div
            v-for="node in rootNodes"
            :key="node.id"
            class="tree-draggable-item"
            :data-item-id="node.id"
            data-item-type="folder"
            @click.stop
          >
            <CollectionTreeNode
              :node="node"
              :requests="requestMap[node.id] || []"
              :request-map="requestMap"
              :variants-map="variantsMap"
              :depth="0"
              :expanded-ids="expandedIds"
              :active-request-id="highlightRequestId"
              :creation-target="creationTarget"
              :selection-mode="selectionMode"
              :selected-keys="selectedKeys"
              @toggle="toggleExpand"
              @select-folder="selectFolderTarget"
              @select-request="onSelectRequest"
              @toggle-select="toggleSelected"
              @context="openContext"
              @drag-complete="onDragComplete"
            />
          </div>
        </VueDraggable>
        <button
          type="button"
          class="tree-root-spacer"
          aria-label="Select workspace root"
          @click.stop="selectWorkspaceRoot"
        />
      </div>
    </div>

    <div
      v-if="contextMenu"
      class="fixed z-50 py-1 rounded shadow-lg text-sm min-w-[140px]"
      :style="{ left: contextMenu.x + 'px', top: contextMenu.y + 'px', background: 'var(--color-surface-2)', border: '1px solid var(--border)' }"
    >
      <button v-for="item in contextItems" :key="item.id" class="block w-full text-left px-3 py-1.5 hover:bg-[var(--color-grid)]" @click="item.action()">
        {{ item.label }}
      </button>
    </div>

    <ConfirmDialog
      v-model:open="confirmOpen"
      title="Delete"
      :message="confirmMessage"
      confirm-label="Delete"
      destructive
      @confirm="confirmAction"
    />

    <CodeSnippetDialog
      :open="!!snippetRequestId"
      :request-id="snippetRequestId || ''"
      @close="snippetRequestId = null"
    />
  </div>
</template>

<script setup lang="ts">
import type { SortableEvent } from 'sortablejs'
import { VueDraggable } from 'vue-draggable-plus'
import type { Collection, RequestItem, TreeNode } from '~/stores/collections'
import {
  executeDragPersistence,
  isInvalidFolderDrop,
  parseDraggedItem,
  planDragPersistence,
} from '~/utils/collectionTreeOps'
import {
  createFolderAtTarget,
  createRequestAtTarget,
  type CreationTarget,
} from '~/utils/createRequest'
import {
  allTreeSelectionKeys,
  collectionSubtreeIds,
  topLevelSelectedCollectionIds,
  type TreeSelectionKey,
} from '~/utils/treeSelection'

const props = defineProps<{
  workspaceId: string
  highlightRequestId?: string | null
  highlightCollectionId?: string | null
}>()
const emit = defineEmits<{
  'open-request': [req: RequestItem]
  'open-collection': [col: Collection]
}>()

const colStore = useCollectionsStore()
const tabsStore = useTabsStore()
const route = useRoute()
const { state: dragState, isLocked, startDrag, beginPersist, endDrag, detectHoverFromMove } = useTreeDragState()
const tree = computed(() => colStore.tree)
const requestMap = computed(() => colStore.requestsByCollection)
const variantsMap = computed(() => colStore.variantsByTemplate)
const rootNodes = ref<TreeNode[]>([])
const creating = ref(false)
const dragLocked = computed(() => isLocked())
const creationTarget = ref<CreationTarget>('workspace')
const selectionMode = ref(false)
const selectedKeys = ref<Set<TreeSelectionKey>>(new Set())
const deleting = ref(false)
const selectAllCheckbox = ref<HTMLInputElement | null>(null)

const hasTreeItems = computed(() => colStore.collections.length > 0 || colStore.requests.length > 0)
const allItemKeys = computed(() => allTreeSelectionKeys(colStore.collections, colStore.requests))
const selectedCount = computed(() => selectedKeys.value.size)
const allSelected = computed(() => allItemKeys.value.length > 0 && selectedCount.value === allItemKeys.value.length)
const someSelected = computed(() => selectedCount.value > 0)

watch([allSelected, someSelected], () => {
  if (selectAllCheckbox.value) {
    selectAllCheckbox.value.indeterminate = someSelected.value && !allSelected.value
  }
})

watch(tree, (nodes) => {
  if (!isLocked()) {
    rootNodes.value = nodes.map(node => ({ ...node, children: [...node.children] }))
  }
}, { immediate: true })

const expandedStorageKey = computed(() => `postchi:tree-expanded:${props.workspaceId}`)

function loadExpandedIds(): Record<string, boolean> {
  if (!import.meta.client) return {}
  try {
    const raw = localStorage.getItem(expandedStorageKey.value)
    if (raw) return JSON.parse(raw) as Record<string, boolean>
  } catch { /* ignore */ }
  return {}
}

const expandedIds = ref<Record<string, boolean>>(loadExpandedIds())
const treeScroll = ref<HTMLElement | null>(null)
const contextMenu = ref<{ x: number; y: number; target: { type: 'collection' | 'request'; item: any } } | null>(null)
const confirmOpen = ref(false)
const confirmMessage = ref('')
const snippetRequestId = ref<string | null>(null)
let confirmAction = () => {}

onMounted(() => {
  document.addEventListener('click', () => { contextMenu.value = null })
  if (Object.keys(expandedIds.value).length === 0 && colStore.requests.length > 0) {
    expandFoldersWithRequests()
  }
})

watch(expandedIds, (value) => {
  if (!import.meta.client) return
  localStorage.setItem(expandedStorageKey.value, JSON.stringify(value))
}, { deep: true })

watch(
  () => [props.highlightRequestId, colStore.requests.length] as const,
  ([requestId]) => {
    if (requestId) revealRequest(requestId)
    else if (props.highlightCollectionId) revealCollection(props.highlightCollectionId)
  },
  { immediate: true },
)

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

function scrollToRequest(requestId: string) {
  nextTick(() => {
    const el = treeScroll.value?.querySelector(`[data-request-id="${requestId}"]`)
    el?.scrollIntoView({ block: 'nearest', behavior: 'smooth' })
  })
}

function revealRequest(requestId: string) {
  const req = colStore.requests.find(r => r.id === requestId)
  if (!req) return
  expandAncestors(req.collection_id)
  scrollToRequest(requestId)
}

function revealCollection(collectionId: string) {
  expandAncestors(collectionId)
}

function expandFoldersWithRequests() {
  const next = { ...expandedIds.value }
  for (const colId of Object.keys(requestMap.value)) {
    const visited = new Set<string>()
    let cur = colStore.collections.find(c => c.id === colId)
    while (cur && !visited.has(cur.id)) {
      visited.add(cur.id)
      next[cur.id] = true
      if (!cur.parent_id) break
      cur = colStore.collections.find(c => c.id === cur!.parent_id)
    }
  }
  expandedIds.value = next
}

function toggleExpand(id: string) {
  expandedIds.value = {
    ...expandedIds.value,
    [id]: !expandedIds.value[id],
  }
}

function selectWorkspaceRoot() {
  creationTarget.value = 'workspace'
  colStore.setActiveCollection(null)
}

function selectFolderTarget(folderId: string) {
  creationTarget.value = folderId
  const col = colStore.collections.find(c => c.id === folderId)
  if (col) colStore.setActiveCollection(col)
}

function onSelectRequest(req: RequestItem) {
  creationTarget.value = req.collection_id
  emit('open-request', req)
}

function onTreeBackgroundClick(event: MouseEvent) {
  const target = event.target as HTMLElement
  if (
    target === treeScroll.value
    || target.classList.contains('tree-drop-container--root')
    || target.classList.contains('tree-draggable-list')
    || target.classList.contains('tree-root-spacer')
  ) {
    selectWorkspaceRoot()
  }
}

const workspaceId = computed(() => props.workspaceId || (route.params.id as string))

async function refreshTree() {
  await colStore.fetchCollections(workspaceId.value)
  await colStore.fetchAllRequests(workspaceId.value)
  rootNodes.value = colStore.tree.map(node => ({ ...node, children: [...node.children] }))
}

function onDragStart(evt: SortableEvent) {
  const item = parseDraggedItem(evt.item as HTMLElement)
  if (item) startDrag(item)
}

function onRootListMove(evt: SortableEvent) {
  detectHoverFromMove(evt)
  return onRootMove(evt)
}

function onRootMove(evt: SortableEvent) {
  const dragged = parseDraggedItem(evt.dragged as HTMLElement)
  if (!dragged || dragged.type !== 'folder') return false
  return !isInvalidFolderDrop(colStore.collections, dragged.id, null)
}

async function onDragComplete(evt: SortableEvent) {
  const hoverFolderId = dragState.hoverFolderId
  beginPersist()

  try {
    const plan = planDragPersistence({
      fromEl: evt.from as HTMLElement,
      toEl: evt.to as HTMLElement,
      itemEl: evt.item as HTMLElement,
      relatedEl: (evt.related as HTMLElement | null) ?? null,
      hoverFolderId,
      oldIndex: evt.oldIndex ?? -1,
      newIndex: evt.newIndex ?? -1,
      collections: colStore.collections,
      requests: colStore.requests,
    })

    if (!plan) return
    if (plan === 'invalid') {
      await refreshTree()
      return
    }

    await executeDragPersistence(colStore, plan)
  } catch (err) {
    console.error('Failed to persist tree drag', err)
    await refreshTree()
  } finally {
    endDrag()
    rootNodes.value = colStore.tree.map(node => ({ ...node, children: [...node.children] }))
  }
}

async function createFolder() {
  await createFolderAtTarget(colStore, workspaceId.value, creationTarget.value)
  if (creationTarget.value === 'workspace') {
    await colStore.fetchCollections(workspaceId.value)
  }
}

async function createRequest() {
  if (creating.value) return
  creating.value = true
  try {
    const saved = await createRequestAtTarget(colStore, workspaceId.value, creationTarget.value)
    expandAncestors(saved.collection_id)
    selectFolderTarget(saved.collection_id)
    emit('open-request', saved)
  } catch (err) {
    console.error('Failed to create request', err)
  } finally {
    creating.value = false
  }
}

function openContext(e: MouseEvent, target: { type: 'collection' | 'request'; item: any }) {
  e.preventDefault()
  contextMenu.value = { x: e.clientX, y: e.clientY, target }
}

const contextItems = computed(() => {
  if (!contextMenu.value) return []
  const { type, item } = contextMenu.value.target
  const items = []
  if (type === 'collection') {
    items.push({ id: 'open', label: 'Open settings', action: () => emit('open-collection', item) })
    items.push({ id: 'folder', label: 'New folder', action: () => createSubFolder(item.id) })
    items.push({ id: 'req', label: 'New request', action: () => createSubRequest(item.id) })
    items.push({ id: 'dup', label: 'Duplicate', action: () => duplicateCol(item.id) })
    items.push({ id: 'del', label: 'Delete', action: () => askDelete('collection', item) })
  } else {
    items.push({ id: 'code', label: 'Generate Code', action: () => { snippetRequestId.value = item.id; contextMenu.value = null } })
    items.push({ id: 'dup', label: 'Duplicate', action: () => duplicateReq(item.id) })
    items.push({ id: 'del', label: 'Delete', action: () => askDelete('request', item) })
  }
  return items
})

async function createSubFolder(parentId: string) {
  selectFolderTarget(parentId)
  await createFolderAtTarget(colStore, workspaceId.value, parentId)
}

async function createSubRequest(colId: string) {
  if (creating.value) return
  creating.value = true
  try {
    selectFolderTarget(colId)
    const saved = await createRequestAtTarget(colStore, workspaceId.value, colId)
    expandAncestors(saved.collection_id)
    emit('open-request', saved)
  } catch (err) {
    console.error('Failed to create request', err)
  } finally {
    creating.value = false
  }
}

async function duplicateCol(id: string) {
  await colStore.duplicateCollection(id)
  await colStore.fetchCollections(workspaceId.value)
}

async function duplicateReq(id: string) {
  await colStore.duplicateRequest(id)
}

function askDelete(type: string, item: any) {
  confirmMessage.value = `Delete ${type} "${item.name}"?`
  confirmOpen.value = true
  confirmAction = async () => {
    if (type === 'collection') {
      await colStore.deleteCollection(item.id)
      tabsStore.closeTabsForEntities('collection', [item.id], { skipDirtyConfirm: true })
    } else {
      await colStore.deleteRequest(item.id)
      tabsStore.closeTabsForEntities('request', [item.id], { skipDirtyConfirm: true })
    }
    await colStore.fetchCollections(workspaceId.value)
    confirmOpen.value = false
  }
}

function enterSelectionMode() {
  selectionMode.value = true
  selectedKeys.value = new Set()
}

function exitSelectionMode() {
  selectionMode.value = false
  selectedKeys.value = new Set()
}

function toggleSelected(key: TreeSelectionKey) {
  const next = new Set(selectedKeys.value)
  if (next.has(key)) next.delete(key)
  else next.add(key)
  selectedKeys.value = next
}

function toggleSelectAll() {
  if (allSelected.value) {
    selectedKeys.value = new Set()
    return
  }
  selectedKeys.value = new Set(allItemKeys.value)
}

function askBulkDelete() {
  const count = selectedCount.value
  if (!count) return
  confirmMessage.value = `Delete ${count} selected item${count === 1 ? '' : 's'}? This cannot be undone.`
  confirmOpen.value = true
  confirmAction = async () => {
    deleting.value = true
    try {
      await bulkDeleteSelected()
      exitSelectionMode()
    } finally {
      deleting.value = false
      confirmOpen.value = false
    }
  }
}

async function bulkDeleteSelected() {
  const selectedCollectionIds = new Set<string>()
  const selectedRequestIds = new Set<string>()
  for (const key of selectedKeys.value) {
    const [type, id] = key.split(':') as ['collection' | 'request', string]
    if (type === 'collection') selectedCollectionIds.add(id)
    else selectedRequestIds.add(id)
  }

  const topLevelCollections = topLevelSelectedCollectionIds(selectedCollectionIds, colStore.collections)
  const deletedCollectionSubtree = collectionSubtreeIds(topLevelCollections, colStore.collections)

  for (const id of topLevelCollections) {
    await colStore.deleteCollection(id)
  }

  await colStore.fetchCollections(workspaceId.value)
  await colStore.fetchAllRequests(workspaceId.value)

  for (const id of selectedRequestIds) {
    const req = colStore.requests.find(r => r.id === id)
    if (!req) continue
    if (deletedCollectionSubtree.has(req.collection_id)) continue
    await colStore.deleteRequest(id)
  }

  tabsStore.closeTabsForEntities('collection', [...selectedCollectionIds], { skipDirtyConfirm: true })
  tabsStore.closeTabsForEntities('request', [...selectedRequestIds], { skipDirtyConfirm: true })
}
</script>

<style scoped>
.tree-muted {
  color: var(--color-text-muted);
}
</style>
