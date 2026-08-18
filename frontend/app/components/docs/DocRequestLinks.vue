<template>
  <aside
    class="shrink-0 border-l w-56 flex flex-col min-h-0"
    style="border-color: var(--color-border); background: var(--color-surface-1)"
  >
    <div class="px-3 py-2 border-b shrink-0" style="border-color: var(--color-border)">
      <div class="flex items-center justify-between gap-2">
        <span class="text-xs font-medium">
          Linked requests
          <span v-if="links.length" class="text-muted font-normal">({{ links.length }})</span>
        </span>
        <button
          type="button"
          class="text-[10px] px-2 py-0.5 rounded hover:bg-surface-2 transition"
          style="color: var(--color-accent)"
          @click="requestPickerOpen = true"
        >
          Link
        </button>
      </div>
    </div>

    <div v-if="loading" class="px-3 py-2 text-xs text-muted">Loading…</div>

    <ul v-else class="flex-1 min-h-0 overflow-auto ui-scroll-y py-1">
      <li v-for="link in links" :key="link.id" class="group px-2 py-1">
        <div class="flex items-center gap-1.5 rounded px-1 py-1.5 hover:bg-surface-2">
          <MethodBadge :method="link.method" class="shrink-0 scale-90" />
          <div class="min-w-0 flex-1">
            <div class="text-xs font-medium truncate">{{ link.request_name }}</div>
            <div class="text-[10px] text-muted truncate">{{ link.collection_name }}</div>
          </div>
          <button
            type="button"
            class="text-[10px] text-muted opacity-0 group-hover:opacity-100 hover:text-default shrink-0 px-1"
            title="Remove link"
            @click="unlink(link)"
          >
            ×
          </button>
        </div>
      </li>
      <li v-if="!links.length" class="px-3 py-2 text-xs text-muted">No linked requests.</li>
    </ul>

    <EntitySearchPicker
      :open="requestPickerOpen"
      :items="pickerRequests"
      :get-key="(r: WorkspaceRequest) => r.id"
      :get-title="(r: WorkspaceRequest) => r.name"
      :get-subtitle="(r: WorkspaceRequest) => `${r.method} ${r.url}`"
      :search-keys="['name', 'url', 'method', 'source_operation_id']"
      placeholder="Search requests…"
      @close="requestPickerOpen = false"
      @select="onRequestSelected"
    >
      <template #item="{ item }">
        <div class="flex items-center gap-2">
          <MethodBadge :method="item.method" class="shrink-0 scale-90" />
          <div class="min-w-0">
            <div class="font-medium truncate">{{ item.name }}</div>
            <div class="text-xs text-muted truncate">{{ item.url }}</div>
          </div>
        </div>
      </template>
    </EntitySearchPicker>
  </aside>
</template>

<script setup lang="ts">
import type { DocLinkItem } from '~/stores/docs'
import type { RequestItem } from '~/stores/collections'

type WorkspaceRequest = RequestItem & { collection_name?: string }

const props = defineProps<{
  workspaceId: string
  docId: string
}>()

const docsStore = useDocsStore()
const api = useApi()
const links = ref<DocLinkItem[]>([])
const loading = ref(false)
const requestPickerOpen = ref(false)
const workspaceRequests = ref<WorkspaceRequest[]>([])

const linkedRequestIds = computed(() => new Set(links.value.map(l => l.request_id)))

const pickerRequests = computed(() =>
  workspaceRequests.value.filter(r => !linkedRequestIds.value.has(r.id)),
)

async function fetchLinks() {
  if (!props.docId) return
  loading.value = true
  try {
    links.value = await docsStore.fetchDocLinks(props.workspaceId, props.docId)
  } catch {
    links.value = []
  } finally {
    loading.value = false
  }
}

async function fetchRequests() {
  try {
    workspaceRequests.value = await api.get<WorkspaceRequest[]>(
      `/api/workspaces/${props.workspaceId}/requests`,
    )
  } catch {
    workspaceRequests.value = []
  }
}

watch(() => [props.workspaceId, props.docId] as const, () => {
  fetchLinks()
}, { immediate: true })

watch(requestPickerOpen, (open) => {
  if (open && !workspaceRequests.value.length) fetchRequests()
})

async function onRequestSelected(req: WorkspaceRequest) {
  await docsStore.createDocLink(props.workspaceId, props.docId, { request_id: req.id })
  await fetchLinks()
  await docsStore.fetchGraph(props.workspaceId)
}

async function unlink(link: DocLinkItem) {
  await docsStore.deleteDocLink(props.workspaceId, props.docId, link.id)
  await fetchLinks()
  await docsStore.fetchGraph(props.workspaceId)
}
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}
.hover\:bg-surface-2:hover {
  background: var(--color-surface-2);
}
</style>
