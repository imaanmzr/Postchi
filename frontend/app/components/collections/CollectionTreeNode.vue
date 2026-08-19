<template>
  <div class="tree-node">
    <div
      class="tree-folder-row"
      :class="{
        'tree-folder-row--hover-target': isHoverTarget,
        'tree-folder-row--selected': creationTarget === node.id,
        'tree-row--checked': isChecked('collection', node.id),
      }"
      :style="{ paddingLeft: `${depth * 12 + 4}px` }"
      :data-drop-folder-id="node.id"
      @click="onFolderRowClick"
      @contextmenu="$emit('context', $event, { type: 'collection', item: node })"
    >
      <input
        v-if="selectionMode"
        type="checkbox"
        class="tree-select-checkbox shrink-0"
        :checked="isChecked('collection', node.id)"
        :aria-label="`Select folder ${node.name}`"
        @click.stop
        @change="toggleChecked('collection', node.id)"
      />
      <TreeRowIcons
        show-grip
        show-chevron
        show-folder
        :expanded="expanded"
        @toggle="onFolderClick"
      />
      <span class="tree-folder-name flex-1 truncate">{{ node.name }}</span>
    </div>

    <div
      v-if="!expanded && isHoverTarget"
      class="tree-drop-container tree-drop-container--collapsed-hover"
      :data-collection-id="node.id"
    >
      <div class="tree-drop-placeholder">Drop into {{ node.name }}</div>
    </div>

    <div
      v-show="expanded"
      class="tree-drop-container"
      :class="{ 'tree-drop-container--active': isHoverTarget }"
      :data-collection-id="node.id"
    >
      <VueDraggable
        v-model="childEntries"
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
        :empty-insert-threshold="12"
        :move="onMove"
        @start="onDragStart"
        @move="onListMove"
        @end="onDragEnd"
      >
        <div
          v-for="entry in childEntries"
          :key="entryKey(entry)"
          class="tree-draggable-item"
          :data-item-id="entry.kind === 'folder' ? entry.node.id : entry.request.id"
          :data-item-type="entry.kind"
        >
          <div
            v-if="entry.kind === 'request'"
            class="tree-request-row"
            :class="{ 'tree-row--checked': isChecked('request', entry.request.id) }"
            :style="{ paddingLeft: `${(depth + 1) * 12 + 8}px` }"
          >
            <input
              v-if="selectionMode"
              type="checkbox"
              class="tree-select-checkbox shrink-0"
              :checked="isChecked('request', entry.request.id)"
              :aria-label="`Select request ${entry.request.name}`"
              @click.stop
              @change="toggleChecked('request', entry.request.id)"
            />
            <TreeRowIcons show-grip />
            <button
              class="tree-request-btn"
              :class="{ 'tree-request-active': entry.request.id === activeRequestId }"
              :data-request-id="entry.request.id"
              @click="onRequestClick(entry.request)"
              @contextmenu="$emit('context', $event, { type: 'request', item: entry.request })"
            >
              <MethodBadge :method="entry.request.method" />
              <span class="truncate">{{ entry.request.name }}</span>
              <span v-if="variantCount(entry.request.id)" class="tree-variant-pill">
                {{ variantCount(entry.request.id) }} variants
              </span>
            </button>
          </div>

          <template v-else>
            <CollectionTreeNode
              :node="entry.node"
              :requests="requestMap[entry.node.id] || []"
              :request-map="requestMap"
              :variants-map="variantsMap"
              :depth="depth + 1"
              :expanded-ids="expandedIds"
              :active-request-id="activeRequestId"
              :creation-target="creationTarget"
              :selection-mode="selectionMode"
              :selected-keys="selectedKeys"
              @toggle="$emit('toggle', $event)"
              @select-folder="$emit('select-folder', $event)"
              @select-request="$emit('select-request', $event)"
              @toggle-select="$emit('toggle-select', $event)"
              @context="(e, t) => $emit('context', e, t)"
              @drag-complete="$emit('drag-complete', $event)"
            />
          </template>

          <div
            v-for="child in entry.kind === 'request' ? (variantsMap[entry.request.id] || []) : []"
            :key="child.id"
            class="tree-variant-row"
            :class="{ 'tree-row--checked': isChecked('request', child.id) }"
            :style="{ paddingLeft: `${(depth + 1) * 12 + 24}px` }"
          >
            <input
              v-if="selectionMode"
              type="checkbox"
              class="tree-select-checkbox shrink-0"
              :checked="isChecked('request', child.id)"
              :aria-label="`Select request ${child.name}`"
              @click.stop
              @change="toggleChecked('request', child.id)"
            />
            <button
              class="tree-request-btn tree-request-btn--variant"
              :class="{ 'tree-request-active': child.id === activeRequestId }"
              :data-request-id="child.id"
              @click="onRequestClick(child)"
              @contextmenu="$emit('context', $event, { type: 'request', item: child })"
            >
              <span class="tree-variant-arrow">↳</span>
              <MethodBadge :method="child.method" />
              <span class="truncate">{{ child.name }}</span>
            </button>
          </div>
        </div>
      </VueDraggable>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { SortableEvent } from 'sortablejs'
import { VueDraggable } from 'vue-draggable-plus'
import type { RequestItem, TreeNode } from '~/stores/collections'
import type { CreationTarget } from '~/utils/createRequest'
import { mergeTreeEntries, type TreeEntry } from '~/utils/treeEntries'
import { isInvalidFolderDrop, parseDraggedItem } from '~/utils/collectionTreeOps'
import { useTreeDragState } from '~/composables/useTreeDragState'

const props = defineProps<{
  node: TreeNode
  requests: RequestItem[]
  requestMap: Record<string, RequestItem[]>
  variantsMap: Record<string, RequestItem[]>
  depth: number
  expandedIds: Record<string, boolean>
  activeRequestId?: string | null
  creationTarget?: CreationTarget
  selectionMode?: boolean
  selectedKeys?: Set<string>
}>()

const emit = defineEmits<{
  toggle: [id: string]
  'select-folder': [id: string]
  'select-request': [req: RequestItem]
  'toggle-select': [key: string]
  context: [e: MouseEvent, target: { type: 'collection' | 'request'; item: any }]
  'drag-complete': [evt: SortableEvent]
}>()

const colStore = useCollectionsStore()
const { state: dragState, isLocked, startDrag, detectHoverFromMove } = useTreeDragState()
const expanded = computed(() => props.expandedIds[props.node.id] === true)
const childEntries = ref<TreeEntry[]>([])
const dragLocked = computed(() => isLocked())

const isHoverTarget = computed(() => dragState.active && dragState.hoverFolderId === props.node.id)

watch(
  () => [props.node.children, props.requests] as const,
  () => {
    if (isLocked()) return
    childEntries.value = mergeTreeEntries(props.node, props.requests)
  },
  { immediate: true },
)

function entryKey(entry: TreeEntry) {
  return entry.kind === 'folder' ? `folder:${entry.node.id}` : `request:${entry.request.id}`
}

function variantCount(id: string) {
  return props.variantsMap[id]?.length || 0
}

function onFolderClick() {
  emit('toggle', props.node.id)
}

function isChecked(type: 'collection' | 'request', id: string) {
  return props.selectedKeys?.has(`${type}:${id}`) ?? false
}

function toggleChecked(type: 'collection' | 'request', id: string) {
  emit('toggle-select', `${type}:${id}`)
}

function onFolderRowClick() {
  if (props.selectionMode) {
    toggleChecked('collection', props.node.id)
    return
  }
  emit('toggle', props.node.id)
  emit('select-folder', props.node.id)
}

function onRequestClick(req: RequestItem) {
  if (props.selectionMode) {
    toggleChecked('request', req.id)
    return
  }
  emit('select-request', req)
}

function onDragStart(evt: SortableEvent) {
  const item = parseDraggedItem(evt.item as HTMLElement)
  if (item) startDrag(item)
}

function onListMove(evt: SortableEvent) {
  detectHoverFromMove(evt)
  return onMove(evt)
}

function onMove(evt: SortableEvent) {
  const dragged = parseDraggedItem(evt.dragged as HTMLElement)
  if (dragged?.type === 'folder' && dragged.id) {
    return !isInvalidFolderDrop(colStore.collections, dragged.id, props.node.id)
  }
  return true
}

function onDragEnd(evt: SortableEvent) {
  emit('drag-complete', evt)
}
</script>
