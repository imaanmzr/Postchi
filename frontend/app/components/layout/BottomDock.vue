<template>
  <div
    class="flex shrink-0 min-h-0 min-w-0 overflow-hidden"
    :class="position === 'right' ? 'flex-col h-full border-l' : 'flex-col w-full border-t'"
    :style="rootStyle"
    style="border-color: var(--color-border); background: var(--color-surface-1)"
  >
    <!-- Compact strip when right + collapsed -->
    <div
      v-if="position === 'right' && collapsed"
      class="flex flex-col items-center gap-2 py-2 w-9 h-full shrink-0"
      style="background: var(--color-bg)"
    >
      <button
        type="button"
        class="ui-btn ui-btn-ghost text-xs px-1"
        title="Expand panel"
        @click="toggleCollapse"
      >◀</button>
      <button
        v-for="tab in dockTabs"
        :key="tab.id"
        type="button"
        class="ui-btn ui-btn-ghost text-[10px] w-7 h-7 p-0"
        :class="activeTab === tab.id ? 'dock-tab-active' : ''"
        :title="tab.label"
        @click="setTab(tab.id)"
      >
        {{ tab.short }}
      </button>
      <div class="flex-1" />
      <button
        type="button"
        class="ui-btn ui-btn-ghost text-[10px] px-1"
        title="Move panel to bottom"
        @click="emit('toggle-position')"
      >↓</button>
    </div>

    <template v-else>
      <div
        class="flex flex-col flex-1 min-h-0 min-w-0 w-full overflow-hidden"
        :style="bodyStyle"
      >
      <div
        class="flex items-center gap-1 px-2 h-9 shrink-0 border-b"
        style="border-color: var(--color-border); background: var(--color-bg)"
      >
        <button
          v-for="tab in dockTabs"
          :key="tab.id"
          type="button"
          class="px-3 py-1 text-xs font-medium tracking-tight rounded-md transition whitespace-nowrap"
          :class="activeTab === tab.id ? 'dock-tab-active' : 'text-muted hover:text-default'"
          @click="setTab(tab.id)"
        >
          {{ tab.label }}
          <span v-if="tab.badge" class="ui-badge ml-1">{{ tab.badge }}</span>
        </button>
        <div class="flex-1 min-w-2" />
        <button
          type="button"
          class="ui-btn ui-btn-ghost text-xs px-2 shrink-0"
          :title="position === 'bottom' ? 'Move panel to right' : 'Move panel to bottom'"
          @click="emit('toggle-position')"
        >
          {{ position === 'bottom' ? 'Right' : 'Bottom' }}
        </button>
        <button
          type="button"
          class="ui-btn ui-btn-ghost text-xs px-2 shrink-0"
          :title="collapsed ? 'Expand panel' : 'Collapse panel'"
          @click="toggleCollapse"
        >
          {{ collapseIcon }}
        </button>
      </div>

      <div
        class="dock-body relative overflow-hidden flex-1 min-h-0 min-w-0 w-full"
        :class="{ 'dock-body-resizing': isResizing }"
      >
        <template v-if="!collapsed">
          <div
            v-if="position === 'right'"
            class="absolute top-0 bottom-0 left-0 w-1.5 z-10 cursor-col-resize hover:bg-[var(--color-accent-muted)]"
            @mousedown="startResize"
          />
          <div
            v-else
            class="absolute top-0 left-0 right-0 h-1.5 z-10 cursor-row-resize hover:bg-[var(--color-accent-muted)]"
            @mousedown="startResize"
          />

          <div class="h-full min-h-0 min-w-0 w-full overflow-hidden">
            <Transition name="dock-tab-fade" mode="out-in">
              <div v-if="activeTab === 'response'" key="response" class="h-full min-h-0 min-w-0 w-full">
                <RequestLoadingIndicator
                  v-if="executing"
                  :elapsed-ms="executeElapsedMs"
                />
                <ResponseViewer
                  v-else-if="response"
                  class="h-full w-full"
                  :response="response"
                  :workspace-id="workspaceId"
                  :share-kind="shareKind"
                  :share-source-id="shareSourceId"
                  :share-title="shareTitle"
                />
                <div v-else class="h-full flex items-center justify-center text-sm text-muted">
                  Send a request to see the response
                </div>
              </div>
              <div v-else-if="activeTab === 'history'" key="history" class="h-full min-h-0 min-w-0 overflow-y-auto p-3">
                <p v-if="!history.length" class="text-xs text-muted">No requests executed yet.</p>
                <div
                  v-for="h in history"
                  :key="h.id"
                  class="w-full text-left text-xs p-2 mb-1 rounded-md font-mono transition group"
                  style="background: var(--color-surface-2)"
                >
                  <button
                    type="button"
                    class="w-full text-left hover:opacity-90"
                    @click="emit('select-history', h)"
                  >
                    <div class="text-[10px] text-muted mb-0.5 flex justify-between gap-2">
                      <span>{{ executorLabel(h) }}</span>
                    </div>
                    <div class="flex items-center gap-2 min-w-0">
                      <MethodBadge :method="historyMethod(h)" class="shrink-0" />
                      <span class="truncate flex-1">{{ historyUrl(h) }}</span>
                      <span :class="statusClass(h.status_code)">{{ h.status_code }}</span>
                      <span class="text-muted shrink-0">{{ h.duration_ms }}ms</span>
                    </div>
                  </button>
                  <div class="mt-1 flex justify-end opacity-0 group-hover:opacity-100 transition">
                    <ShareButton
                      :workspace-id="workspaceId"
                      kind="history"
                      :source-id="h.id"
                      :default-title="historyShareTitle(h)"
                      label="Share"
                    />
                  </div>
                </div>
              </div>
              <div v-else key="runner" class="h-full min-h-0 min-w-0 overflow-y-auto p-3">
                <CollectionRunner :collection-id="collectionId" :workspace-id="workspaceId" />
              </div>
            </Transition>
          </div>
        </template>
      </div>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import type { HistoryEntry } from '~/stores/history'

export type DockPosition = 'bottom' | 'right'

const props = withDefaults(defineProps<{
  workspaceId: string
  collectionId: string | null
  history: HistoryEntry[]
  response: any | null
  activeTab: 'response' | 'history' | 'runner'
  position: DockPosition
  size: number
  maxSize: number
  collapsed?: boolean
  shareKind?: 'request' | 'history'
  shareSourceId?: string
  shareTitle?: string
  executing?: boolean
  executeElapsedMs?: number
}>(), {
  collapsed: false,
  executing: false,
  executeElapsedMs: 0,
})

const emit = defineEmits<{
  'update:activeTab': [tab: 'response' | 'history' | 'runner']
  'update:collapsed': [value: boolean]
  'toggle-position': []
  resize: [size: number]
  'select-history': [entry: HistoryEntry]
}>()

const isResizing = ref(false)

const dockTabs = computed(() => [
  { id: 'response' as const, label: 'Response', short: 'R' },
  { id: 'history' as const, label: 'History', short: 'H', badge: props.history.length || undefined },
  { id: 'runner' as const, label: 'Runner', short: '▶' },
])

const collapseIcon = computed(() => {
  if (props.collapsed) {
    return props.position === 'bottom' ? '▲' : '◀'
  }
  return props.position === 'bottom' ? '▼' : '▶'
})

const bodyStyle = computed(() => {
  if (props.collapsed) {
    return props.position === 'bottom'
      ? { width: '100%', height: '0px', flex: '0 0 auto' }
      : { width: '0px', height: '100%', flex: '0 0 auto' }
  }
  return props.position === 'bottom'
    ? { width: '100%', height: `${props.size}px`, flex: '0 0 auto' }
    : { width: '100%', height: '100%', flex: '1 1 auto' }
})

const rootStyle = computed(() => {
  if (props.position !== 'right' || props.collapsed) return undefined
  return { width: `${props.size}px` }
})

function setTab(tab: 'response' | 'history' | 'runner') {
  emit('update:activeTab', tab)
  if (props.collapsed) {
    emit('update:collapsed', false)
  }
}

function toggleCollapse() {
  emit('update:collapsed', !props.collapsed)
}

function statusClass(code: number) {
  if (code >= 200 && code < 300) return 'text-[var(--method-get)]'
  if (code >= 400) return 'text-[var(--method-delete)]'
  return 'text-[var(--method-put)]'
}

function historyMethod(h: HistoryEntry) {
  return h.snapshot?.method || 'GET'
}

function historyUrl(h: HistoryEntry) {
  return h.snapshot?.url || h.snapshot?.name || 'Unknown'
}

function executorLabel(h: HistoryEntry) {
  return h.executed_by_name || h.executed_by_email || 'Teammate'
}

function historyShareTitle(h: HistoryEntry) {
  return h.snapshot?.name || h.snapshot?.url || 'Shared execution'
}

function clamp(next: number) {
  const min = props.position === 'bottom' ? 120 : 280
  return Math.min(props.maxSize, Math.max(min, next))
}

function startResize(e: MouseEvent) {
  e.preventDefault()
  isResizing.value = true
  const start = props.position === 'bottom' ? e.clientY : e.clientX
  const startSize = props.size
  document.body.style.userSelect = 'none'
  document.body.style.cursor = props.position === 'bottom' ? 'row-resize' : 'col-resize'

  function onMove(ev: MouseEvent) {
    const delta = props.position === 'bottom'
      ? start - ev.clientY
      : start - ev.clientX
    emit('resize', clamp(startSize + delta))
  }

  function onUp() {
    isResizing.value = false
    document.body.style.userSelect = ''
    document.body.style.cursor = ''
    window.removeEventListener('mousemove', onMove)
    window.removeEventListener('mouseup', onUp)
  }

  window.addEventListener('mousemove', onMove)
  window.addEventListener('mouseup', onUp)
}
</script>

<style scoped>
.dock-body {
  transition:
    width var(--duration-normal) var(--ease-out),
    height var(--duration-normal) var(--ease-out);
}

.dock-body-resizing {
  transition: none;
}

.dock-tab-active {
  background: var(--color-surface-2);
  color: var(--color-text);
  border: 1px solid var(--color-border);
}
.text-muted {
  color: var(--color-text-muted);
}
.hover\:text-default:hover {
  color: var(--color-text);
}
.dock-tab-fade-enter-active,
.dock-tab-fade-leave-active {
  transition: opacity var(--duration-fast) var(--ease-out);
}
.dock-tab-fade-enter-from,
.dock-tab-fade-leave-to {
  opacity: 0;
}
</style>
