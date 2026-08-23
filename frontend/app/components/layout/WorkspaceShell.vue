<template>
  <div class="flex flex-col h-screen overflow-hidden app-bg">
    <WorkspaceToolbar
      :workspace-id="workspaceId"
      :workspace-name="workspaceName"
      :has-active-collection="!!contextCollectionId"
      @open-collections="collectionsOpen = true"
      @open-runner="openDockTab('runner')"
      @open-history="openHistorySidebar"
      @open-variables="openCollectionSettings('vars')"
      @open-openapi="openOpenApi"
      @open-collection-settings="openCollectionSettings('overview')"
    />

    <div class="flex flex-1 overflow-hidden min-h-0">
      <ResizablePane :initial-width="sidebarWidth" storage-key="postchi:workspace-sidebar-width" side="right">
        <aside
          class="h-full flex flex-col border-r overflow-hidden ui-panel"
          style="border-color: var(--color-border)"
        >
          <HistoryPanel
            v-if="sidebarMode === 'history'"
            :workspace-id="workspaceId"
            :entries="history"
            :selected-id="histStore.selectedId"
            @close="sidebarMode = 'collections'"
            @select="onHistorySelect"
          />
          <CollectionTree
            v-else
            :workspace-id="workspaceId"
            :highlight-request-id="highlightRequestId"
            :highlight-collection-id="highlightCollectionId"
            @open-request="onOpenRequest"
            @open-collection="onOpenCollection"
          />
        </aside>
      </ResizablePane>

      <main class="flex-1 flex flex-col overflow-hidden min-w-0 min-h-0">
        <RequestTabBar />
        <div
          class="flex-1 overflow-hidden min-h-0"
          :class="dockPosition === 'right' ? 'flex flex-row' : 'flex flex-col'"
        >
          <div class="flex-1 min-h-0 min-w-0 overflow-hidden">
            <template v-if="activeTab">
              <RequestBuilder
                v-if="activeTab.type === 'request' && activeRequest"
                :request="activeRequest"
                :workspace-id="workspaceId"
                :executing="executing"
                @save="saveRequest"
                @execute="executeRequest"
                @dirty="markDirty"
              />
              <CollectionSettings
                v-else-if="activeTab.type === 'collection' && activeCollection"
                :collection="activeCollection"
                :initial-tab="activeTab.collectionTab"
                @saved="onCollectionSaved"
              />
              <OpenApiConnect
                v-else-if="activeTab.type === 'openapi'"
                :workspace-id="workspaceId"
                :collection-id="contextCollectionId"
              />
            </template>
            <div v-else class="h-full flex flex-col items-center justify-center gap-2 text-muted">
              <p class="text-sm">Select a request from the sidebar</p>
              <Button variant="primary" @click="collectionsOpen = true">Search requests</Button>
            </div>
          </div>

          <BottomDock
            :workspace-id="workspaceId"
            :collection-id="activeCollectionId"
            :history="history"
            :response="dockResponse"
            :active-tab="dockTab"
            :position="dockPosition"
            :size="dockSize"
            :max-size="dockMaxSize"
            :collapsed="dockCollapsed"
            :share-kind="dockShareKind"
            :share-source-id="dockShareSourceId"
            :share-title="dockShareTitle"
            :executing="executing"
            :execute-elapsed-ms="executeElapsedMs"
            @update:active-tab="dockTab = $event"
            @update:collapsed="dockCollapsed = $event"
            @toggle-position="toggleDockPosition"
            @resize="onDockResize"
            @select-history="onHistorySelect"
          />
        </div>
      </main>
    </div>

    <CollectionBrowser
      :open="collectionsOpen"
      :workspace-id="workspaceId"
      @close="collectionsOpen = false"
      @open-request="onOpenRequest"
      @open-collection="onOpenCollection"
    />
  </div>
</template>

<script setup lang="ts">
import type { HistoryEntry } from '~/stores/history'
import { historyEntryToResponse } from '~/stores/history'
import { buildClientErrorResponse, normalizeExecutionResult } from '~/utils/executionResponse'

type DockPosition = 'bottom' | 'right'
type SidebarMode = 'collections' | 'history'

const props = defineProps<{ workspaceId: string; workspaceName: string; loading?: boolean }>()

const route = useRoute()
const router = useRouter()
const colStore = useCollectionsStore()
const envStore = useEnvironmentsStore()
const histStore = useHistoryStore()
const tabsStore = useTabsStore()
const execStore = useExecutionStore()

useWebSocket(computed(() => props.workspaceId))

const sidebarWidth = 300
const DOCK_SIZE_KEY = 'postchi:dock-size'
const DOCK_COLLAPSED_KEY = 'postchi:dock-collapsed'
const DOCK_POSITION_KEY = 'postchi:dock-position'
const DEFAULT_DOCK_SIZE = 320

const collectionsOpen = ref(false)
const sidebarMode = ref<SidebarMode>('collections')
const dockTab = ref<'response' | 'history' | 'runner'>('response')
const dockPosition = ref<DockPosition>('bottom')
const dockSize = ref(DEFAULT_DOCK_SIZE)
const dockMaxSize = ref(800)
const dockCollapsed = ref(false)
const historyResponse = ref<any | null>(null)
const selectedHistoryId = ref<string | null>(null)
const lastHistoryIdByRequest = ref<Record<string, string>>({})
const executing = ref(false)
const executeElapsedMs = ref(0)
let executeTimer: ReturnType<typeof setInterval> | null = null

const response = computed(() => {
  const tab = tabsStore.activeTab
  if (tab?.type === 'request' && tab.entityId) {
    return execStore.get(tab.entityId)
  }
  return null
})

const dockResponse = computed(() => historyResponse.value ?? response.value)

function updateDockMaxSize() {
  if (!import.meta.client) return
  dockMaxSize.value = dockPosition.value === 'bottom'
    ? Math.floor(window.innerHeight * 0.75)
    : Math.floor(window.innerWidth * 0.55)
}

if (import.meta.client) {
  const storedSize = Number(localStorage.getItem(DOCK_SIZE_KEY))
  if (storedSize >= 120) dockSize.value = storedSize
  const collapsed = localStorage.getItem(DOCK_COLLAPSED_KEY)
  if (collapsed !== null) dockCollapsed.value = collapsed === 'true'
  const storedPosition = localStorage.getItem(DOCK_POSITION_KEY)
  if (storedPosition === 'bottom' || storedPosition === 'right') {
    dockPosition.value = storedPosition
  }
  updateDockMaxSize()
  window.addEventListener('resize', updateDockMaxSize)
}

onUnmounted(() => {
  if (import.meta.client) {
    window.removeEventListener('resize', updateDockMaxSize)
  }
  stopExecuteTimer()
})

function startExecuteTimer() {
  const start = performance.now()
  executeElapsedMs.value = 0
  stopExecuteTimer()
  executeTimer = setInterval(() => {
    executeElapsedMs.value = Math.round(performance.now() - start)
  }, 50)
}

function stopExecuteTimer() {
  if (executeTimer) {
    clearInterval(executeTimer)
    executeTimer = null
  }
}

function onDockResize(size: number) {
  dockSize.value = size
  if (import.meta.client) {
    localStorage.setItem(DOCK_SIZE_KEY, String(size))
  }
}

function toggleDockPosition() {
  dockPosition.value = dockPosition.value === 'bottom' ? 'right' : 'bottom'
  if (import.meta.client) {
    localStorage.setItem(DOCK_POSITION_KEY, dockPosition.value)
  }
  updateDockMaxSize()
}

function openDockTab(tab: 'response' | 'history' | 'runner') {
  dockTab.value = tab
  dockCollapsed.value = false
  if (import.meta.client) {
    localStorage.setItem(DOCK_COLLAPSED_KEY, 'false')
  }
}

watch(dockCollapsed, (v) => {
  if (import.meta.client) {
    localStorage.setItem(DOCK_COLLAPSED_KEY, String(v))
  }
})

watch(response, (r) => {
  if (r) {
    historyResponse.value = null
    selectedHistoryId.value = null
    dockTab.value = 'response'
    dockCollapsed.value = false
  }
})

const dockShareKind = computed((): 'request' | 'history' | undefined => {
  if (selectedHistoryId.value) return 'history'
  const tab = tabsStore.activeTab
  if (tab?.type === 'request' && tab.entityId) {
    if (lastHistoryIdByRequest.value[tab.entityId]) return 'history'
    if (tab.entityId) return 'request'
  }
  return undefined
})

const dockShareSourceId = computed(() => {
  if (selectedHistoryId.value) return selectedHistoryId.value
  const tab = tabsStore.activeTab
  if (tab?.type === 'request' && tab.entityId) {
    return lastHistoryIdByRequest.value[tab.entityId] || tab.entityId
  }
  return undefined
})

const dockShareTitle = computed(() => {
  if (selectedHistoryId.value) {
    const entry = histStore.entries.find(e => e.id === selectedHistoryId.value)
    return entry?.snapshot?.name || entry?.snapshot?.url || 'Shared execution'
  }
  const req = colStore.activeRequest
  return req?.name || req?.url || 'Shared request'
})

const activeTab = computed(() => tabsStore.activeTab)
const activeRequest = computed(() => colStore.activeRequest)
const activeCollection = computed(() => colStore.activeCollection)
const activeCollectionId = computed(() => colStore.activeCollectionId)
const history = computed(() => histStore.entries)

const contextCollectionId = computed(() => resolveContextCollectionId())

const highlightRequestId = computed(() => {
  const tab = tabsStore.activeTab
  if (tab?.type === 'request' && tab.entityId) return tab.entityId
  return null
})

const highlightCollectionId = computed(() => {
  const tab = tabsStore.activeTab
  if (tab?.type === 'request' && !tab.entityId) {
    return colStore.activeRequest?.collection_id ?? null
  }
  return null
})

function resolveContextCollectionId(): string | null {
  const tab = tabsStore.activeTab
  if (tab?.type === 'collection' && tab.entityId) return tab.entityId
  if (tab?.type === 'request') {
    const req = colStore.requests.find(r => r.id === tab.entityId)
    if (req) return req.collection_id
    if (colStore.activeRequest?.collection_id) return colStore.activeRequest.collection_id
  }
  const roots = colStore.tree
  return roots[0]?.id ?? null
}

function resolveContextCollection() {
  const id = contextCollectionId.value
  if (!id) return null
  return colStore.collections.find(c => c.id === id) ?? null
}

watch(activeTab, async (tab) => {
  if (tab?.type === 'request' && tab.entityId) {
    const req = colStore.requests.find(r => r.id === tab.entityId)
    if (req) colStore.setActiveRequest(req)
  }
  if (tab?.type === 'collection' && tab.entityId) {
    await colStore.fetchCollection(tab.entityId)
  }
}, { immediate: true })

function restoreRequestFromRoute() {
  const requestId = route.query.request
  if (typeof requestId !== 'string') return

  const req = colStore.requests.find(r => r.id === requestId)
  if (!req) return

  onOpenRequest(req)

  const nextQuery = { ...route.query }
  delete nextQuery.request
  delete nextQuery.tab
  void router.replace({ query: nextQuery })
}

watch(
  () => [route.query.request, colStore.requests.length] as const,
  () => restoreRequestFromRoute(),
  { immediate: true },
)

function onOpenRequest(req: any) {
  sidebarMode.value = 'collections'
  colStore.setActiveRequest(req)
  tabsStore.openRequest(req)
}

function onOpenCollection(col: any) {
  sidebarMode.value = 'collections'
  colStore.setActiveCollection(col)
  tabsStore.openCollection(col)
}

function openHistorySidebar() {
  sidebarMode.value = 'history'
  void histStore.fetch(props.workspaceId)
}

function openOpenApi() {
  sidebarMode.value = 'collections'
  tabsStore.openOpenApi()
}

function openCollectionSettings(tab = 'overview') {
  const col = resolveContextCollection()
  if (!col) return
  sidebarMode.value = 'collections'
  colStore.setActiveCollection(col)
  tabsStore.openCollection(col, { tab })
}

function onHistorySelect(entry: HistoryEntry) {
  histStore.select(entry.id)
  selectedHistoryId.value = entry.id
  historyResponse.value = historyEntryToResponse(entry)
  openDockTab('response')

  if (entry.request_id) {
    const req = colStore.requests.find(r => r.id === entry.request_id)
    if (req) onOpenRequest(req)
  }
}

async function saveRequest(req: any) {
  const previousKey = tabsStore.activeKey
  const saved = await colStore.saveRequest(req)
  colStore.setActiveRequest(saved)
  tabsStore.syncSavedRequest(previousKey, saved)
}

async function executeRequest(req: any) {
  const api = useApi()
  historyResponse.value = null
  selectedHistoryId.value = null
  executing.value = true
  startExecuteTimer()
  openDockTab('response')

  const startedAt = performance.now()
  try {
    const result = await api.post<{ history_id?: string } & Record<string, unknown>>(`/api/requests/${req.id}/execute`, {
      environment_id: envStore.activeId,
      request: req,
    })
    const elapsed = Math.round(performance.now() - startedAt)
    const normalized = normalizeExecutionResult(result, elapsed)
    execStore.set(req.id, normalized)
    if (normalized.history_id) {
      lastHistoryIdByRequest.value = { ...lastHistoryIdByRequest.value, [req.id]: normalized.history_id }
    }
    await histStore.fetch(props.workspaceId)
  } catch (err) {
    const elapsed = Math.round(performance.now() - startedAt)
    execStore.set(req.id, buildClientErrorResponse(err, elapsed))
  } finally {
    executing.value = false
    stopExecuteTimer()
  }
}

function markDirty() {
  if (tabsStore.activeKey) tabsStore.markDirty(tabsStore.activeKey)
}

function onCollectionSaved() {
  if (tabsStore.activeKey) tabsStore.markClean(tabsStore.activeKey)
}
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}
</style>
