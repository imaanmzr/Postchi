<template>
  <Teleport to="body">
    <Transition name="modal-fade">
      <div v-if="open" class="fixed inset-0 z-50 flex items-start justify-center pt-[10vh] p-4">
        <div class="absolute inset-0 ui-overlay backdrop-blur-sm" @click="$emit('close')" />
        <div
          class="relative z-10 w-full max-w-xl rounded-lg shadow-md overflow-hidden flex flex-col max-h-[70vh]"
          style="background: var(--color-surface-1); border: 1px solid var(--color-border)"
        >
          <div class="p-4 border-b shrink-0" style="border-color: var(--color-border)">
            <div class="flex items-center justify-between mb-3">
              <h2 class="text-base font-semibold tracking-tight">Collections</h2>
              <button
                class="ui-btn ui-btn-ghost text-lg leading-none px-2"
                type="button"
                @click="$emit('close')"
              >×</button>
            </div>
            <input
              v-model="query"
              type="search"
              placeholder="Search requests and folders…"
              class="ui-input w-full font-mono text-sm"
              autofocus
            />
            <div class="flex gap-2 mt-3">
              <Button class="text-xs" @click="createFolder">+ Folder</Button>
              <Button class="text-xs" :disabled="creating" @click="createRequest">+ Request</Button>
            </div>
          </div>

          <div class="flex-1 ui-scroll-y p-2 min-h-0">
            <p v-if="!filteredEntries.length" class="p-4 text-sm text-center text-muted">
              {{ query ? 'No matches.' : 'No collections yet. Create a folder or import one.' }}
            </p>

            <button
              v-for="entry in filteredEntries"
              :key="entry.key"
              type="button"
              class="browse-entry w-full flex items-center gap-2 px-3 py-2 rounded-md text-left text-sm transition"
              @click="onSelect(entry)"
            >
              <Folder
                v-if="entry.type === 'collection'"
                class="browse-folder-icon shrink-0"
                :size="14"
                :stroke-width="2"
                aria-hidden="true"
              />
              <MethodBadge v-else :method="entry.method || 'GET'" />
              <span class="truncate font-medium">{{ entry.label }}</span>
              <span v-if="entry.path" class="truncate text-xs text-muted ml-auto">{{ entry.path }}</span>
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>

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
</template>

<script setup lang="ts">
import { Folder } from 'lucide-vue-next'
import type { Collection, RequestItem } from '~/stores/collections'
import { createFolderAtTarget, createRequestAtTarget } from '~/utils/createRequest'

const props = defineProps<{ open: boolean; workspaceId: string }>()
const emit = defineEmits<{
  close: []
  'open-request': [req: RequestItem]
  'open-collection': [col: Collection]
}>()

const colStore = useCollectionsStore()
const route = useRoute()
const query = ref('')
const creating = ref(false)
const confirmOpen = ref(false)
const confirmMessage = ref('')
const snippetRequestId = ref<string | null>(null)
let confirmAction = () => {}

watch(() => props.open, async (isOpen) => {
  if (isOpen) {
    await colStore.fetchCollections(props.workspaceId)
    await colStore.fetchAllRequests(props.workspaceId)
  }
})

interface BrowseEntry {
  key: string
  type: 'collection' | 'request'
  label: string
  path?: string
  method?: string
  collection?: Collection
  request?: RequestItem
}

const entries = computed((): BrowseEntry[] => {
  const out: BrowseEntry[] = []
  const collections = colStore.collections
  const pathMap = new Map<string, string>()

  function collectionPath(id: string, visited = new Set<string>()): string {
    if (pathMap.has(id)) return pathMap.get(id)!
    if (visited.has(id)) return ''
    visited.add(id)
    const col = collections.find(c => c.id === id)
    if (!col) return ''
    const parent = col.parent_id ? collectionPath(col.parent_id, visited) + '/' : ''
    const path = parent + col.name
    pathMap.set(id, path)
    return path
  }

  for (const col of collections) {
    out.push({
      key: `col-${col.id}`,
      type: 'collection',
      label: col.name,
      path: col.parent_id ? collectionPath(col.parent_id) : undefined,
      collection: col,
    })
  }

  for (const req of colStore.requests) {
    if (req.template_id) continue
    const col = collections.find(c => c.id === req.collection_id)
    out.push({
      key: `req-${req.id}`,
      type: 'request',
      label: req.name,
      path: col ? collectionPath(col.id) : undefined,
      method: req.method,
      request: req,
    })
  }

  return out.sort((a, b) => {
    const pa = a.path || a.label
    const pb = b.path || b.label
    return pa.localeCompare(pb)
  })
})

const filteredEntries = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return entries.value
  return entries.value.filter(e =>
    e.label.toLowerCase().includes(q)
    || (e.path && e.path.toLowerCase().includes(q))
    || (e.method && e.method.toLowerCase().includes(q)),
  )
})

function onSelect(entry: BrowseEntry) {
  if (entry.type === 'collection' && entry.collection) {
    emit('open-collection', entry.collection)
    emit('close')
  } else if (entry.type === 'request' && entry.request) {
    emit('open-request', entry.request)
    emit('close')
  }
}

async function createFolder() {
  await createFolderAtTarget(colStore, props.workspaceId, 'workspace')
  await colStore.fetchCollections(props.workspaceId)
}

async function createRequest() {
  if (creating.value) return
  creating.value = true
  try {
    const saved = await createRequestAtTarget(colStore, props.workspaceId, 'workspace')
    emit('open-request', saved)
    emit('close')
  } catch (err) {
    console.error('Failed to create request', err)
  } finally {
    creating.value = false
  }
}

function askDelete(type: string, item: { name: string; id: string }) {
  confirmMessage.value = `Delete ${type} "${item.name}"?`
  confirmOpen.value = true
  confirmAction = async () => {
    if (type === 'collection') await colStore.deleteCollection(item.id)
    else await colStore.deleteRequest(item.id)
    await colStore.fetchCollections(route.params.id as string)
    confirmOpen.value = false
  }
}
</script>

<style scoped>
.modal-fade-enter-active,
.modal-fade-leave-active {
  transition: opacity var(--duration-normal) var(--ease-out);
}
.modal-fade-enter-from,
.modal-fade-leave-to {
  opacity: 0;
}
.text-muted {
  color: var(--color-text-muted);
}
.browse-entry {
  color: var(--color-text);
}
.browse-entry:hover {
  background: var(--color-surface-2);
}
.browse-folder-icon {
  color: var(--color-accent);
}
</style>
