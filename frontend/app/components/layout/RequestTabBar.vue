<template>
  <div
    ref="scrollEl"
    class="flex items-center gap-1 px-3 py-1.5 border-b ui-scroll-x shrink-0"
    style="background: var(--color-bg); border-color: var(--color-border)"
  >
    <TransitionGroup name="tab-list" tag="div" class="flex items-center gap-1">
      <div
        v-for="(tab, index) in tabsStore.openTabs"
        :key="tab.key"
        class="group flex items-center gap-1.5 px-2.5 py-1 rounded-md text-xs cursor-pointer transition shrink-0"
        :class="tabsStore.activeKey === tab.key ? 'tab-active' : 'tab-idle'"
        @click="tabsStore.setActive(tab.key)"
        @contextmenu="openContext($event, tab, index)"
      >
        <MethodBadge v-if="tab.type === 'request'" :method="tab.method || 'GET'" />
        <span v-else-if="tab.type === 'openapi'" class="text-[10px] opacity-70">◎</span>
        <span v-else-if="tab.type === 'collection'" class="text-[10px] opacity-70">⚙</span>
        <span class="truncate max-w-[140px] font-medium tracking-tight">{{ tab.label }}</span>
        <span v-if="tab.dirty" class="w-1.5 h-1.5 rounded-full shrink-0" style="background: var(--method-patch)" />
        <button
          type="button"
          class="opacity-0 group-hover:opacity-100 ml-0.5 text-xs text-muted hover:text-default"
          @click.stop="tabsStore.closeTab(tab.key)"
        >×</button>
      </div>
    </TransitionGroup>

    <div
      v-if="contextMenu"
      class="fixed z-50 py-1 text-sm min-w-[168px] ui-context-menu"
      :style="{ left: contextMenu.x + 'px', top: contextMenu.y + 'px' }"
    >
      <button
        v-for="item in contextItems"
        :key="item.id"
        type="button"
        class="block w-full text-left px-3 py-1.5 text-xs disabled:opacity-40 disabled:cursor-not-allowed"
        :disabled="item.disabled"
        @click="runItem(item)"
      >
        {{ item.label }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { TabEntry } from '~/stores/tabs'
import type { RequestItem } from '~/stores/collections'
import { createRequestAtTarget } from '~/utils/createRequest'

const tabsStore = useTabsStore()
const colStore = useCollectionsStore()
const route = useRoute()

const scrollEl = ref<HTMLElement | null>(null)
useHorizontalWheelScroll(scrollEl)

const contextMenu = ref<{ x: number; y: number; tab: TabEntry; index: number } | null>(null)

onMounted(() => {
  document.addEventListener('click', closeContext)
})

onUnmounted(() => {
  document.removeEventListener('click', closeContext)
})

function closeContext() {
  contextMenu.value = null
}

function openContext(e: MouseEvent, tab: TabEntry, index: number) {
  e.preventDefault()
  contextMenu.value = { x: e.clientX, y: e.clientY, tab, index }
}

interface ContextItem {
  id: string
  label: string
  disabled?: boolean
  action: () => void | Promise<void>
}

const contextItems = computed((): ContextItem[] => {
  if (!contextMenu.value) return []
  const { tab, index } = contextMenu.value
  const tabs = tabsStore.openTabs
  const hasLeft = index > 0
  const hasRight = index < tabs.length - 1
  const hasOthers = tabs.length > 1
  const hasSaved = tabs.some(t => !t.dirty)
  const canClone = tab.type === 'request' && !!tab.entityId

  return [
    { id: 'new', label: 'New Request', action: () => createRequest(tab) },
    { id: 'clone', label: 'Clone Request', disabled: !canClone, action: () => cloneRequest(tab) },
    { id: 'revert', label: 'Revert Changes', disabled: !tab.dirty, action: () => revertChanges(tab) },
    { id: 'close', label: 'Close', action: () => tabsStore.closeTab(tab.key) },
    { id: 'close-others', label: 'Close Others', disabled: !hasOthers, action: () => tabsStore.closeOthers(tab.key) },
    { id: 'close-left', label: 'Close to the Left', disabled: !hasLeft, action: () => tabsStore.closeToLeft(tab.key) },
    { id: 'close-right', label: 'Close to the Right', disabled: !hasRight, action: () => tabsStore.closeToRight(tab.key) },
    { id: 'close-saved', label: 'Close Saved', disabled: !hasSaved, action: () => tabsStore.closeSaved() },
    { id: 'close-all', label: 'Close All', disabled: tabs.length === 0, action: () => tabsStore.closeAll() },
  ]
})

function runItem(item: ContextItem) {
  if (item.disabled) return
  closeContext()
  void item.action()
}

function blankRequest(collectionId: string): RequestItem {
  return {
    id: '',
    collection_id: collectionId,
    name: 'New Request',
    method: 'GET',
    url: '',
    headers: [],
    params: [],
    body: { mode: 'none', raw: '', raw_lang: 'json' },
    auth: { type: 'inherit' },
    settings: { timeout_ms: 30000, follow_redirects: true, verify_ssl: true },
    pre_request_script: '',
    test_script: '',
  }
}

function collectionIdForTab(tab: TabEntry): string | null {
  if (tab.type === 'collection') return tab.entityId
  if (tab.type === 'request') {
    const req = colStore.requests.find(r => r.id === tab.entityId)
    if (req) return req.collection_id
    if (colStore.activeRequest?.collection_id) return colStore.activeRequest.collection_id
  }
  return colStore.activeCollectionId
}

function createRequest(tab: TabEntry) {
  const workspaceId = route.params.id as string
  if (!workspaceId) return
  const collectionId = collectionIdForTab(tab)
  const target = collectionId || 'workspace'
  void (async () => {
    try {
      const saved = await createRequestAtTarget(colStore, workspaceId, target)
      colStore.setActiveRequest(saved)
      tabsStore.openRequest(saved)
    } catch (err) {
      console.error('Failed to create request', err)
    }
  })()
}

async function cloneRequest(tab: TabEntry) {
  if (tab.type !== 'request' || !tab.entityId) return
  const dup = await colStore.duplicateRequest(tab.entityId)
  colStore.setActiveRequest(dup)
  tabsStore.openRequest(dup)
}

function revertChanges(tab: TabEntry) {
  if (!tab.dirty) return
  tabsStore.setActive(tab.key)

  if (tab.type === 'request') {
    if (!tab.entityId) {
      const collectionId = collectionIdForTab(tab)
      if (!collectionId) return
      colStore.setActiveRequest(blankRequest(collectionId))
    } else {
      const saved = colStore.requests.find(r => r.id === tab.entityId)
      if (saved) colStore.setActiveRequest({ ...saved })
    }
  } else if (tab.type === 'collection' && tab.entityId) {
    void colStore.fetchCollection(tab.entityId)
  }

  tabsStore.markClean(tab.key)
}
</script>

<style scoped>
.tab-active {
  background: var(--color-surface-2);
  color: var(--color-text);
}
.tab-idle {
  color: var(--color-text-muted);
}
.tab-idle:hover {
  background: var(--color-surface-1);
  color: var(--color-text);
}
.text-muted {
  color: var(--color-text-muted);
}
.hover\:text-default:hover {
  color: var(--color-text);
}
.tab-list-enter-active,
.tab-list-leave-active {
  transition: opacity var(--duration-fast) var(--ease-out), transform var(--duration-fast) var(--ease-out);
}
.tab-list-enter-from,
.tab-list-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
</style>
