<template>
  <div class="flex flex-col h-full overflow-hidden">
    <CatalogFilters
      :filters="catalogStore.filters"
      :tags="allTags"
      @update="onFilterUpdate"
    />

    <div class="flex flex-1 min-h-0">
      <div class="w-80 shrink-0 border-r ui-scroll-y" style="border-color: var(--color-border)">
        <div
          v-for="col in catalogStore.data?.collections || []"
          :key="col.id"
          class="px-3 py-2 border-b text-sm cursor-pointer hover:bg-surface-2"
          style="border-color: var(--color-border)"
          @click="selectedCollectionId = col.id"
        >
          <div class="font-medium">{{ col.name }}</div>
          <div class="text-xs text-muted">
            {{ col.documented_count }}/{{ col.request_count }} documented
          </div>
        </div>
        <div class="p-2">
          <button
            class="text-xs w-full text-left px-2 py-1 rounded hover:bg-surface-2"
            :class="{ 'font-medium': !selectedCollectionId }"
            @click="selectedCollectionId = null"
          >
            All collections
          </button>
        </div>
        <div
          v-for="ep in filteredEndpoints"
          :key="ep.id"
          class="px-3 py-2 border-b text-sm cursor-pointer hover:bg-surface-2 flex items-center gap-2"
          style="border-color: var(--color-border)"
          :class="{ 'bg-surface-2': selectedEndpoint?.id === ep.id }"
          @click="selectEndpoint(ep)"
        >
          <MethodBadge :method="ep.method" />
          <span class="truncate flex-1">{{ ep.name }}</span>
          <span v-if="!ep.docs_complete" class="w-2 h-2 rounded-full bg-amber-500 shrink-0" title="Undocumented" />
        </div>
      </div>

      <div class="flex-1 ui-scroll-y p-4">
        <template v-if="selectedEndpoint">
          <div class="flex items-center gap-2 mb-4">
            <MethodBadge :method="selectedEndpoint.method" />
            <h2 class="font-semibold">{{ selectedEndpoint.name }}</h2>
            <span class="font-mono text-sm text-muted truncate">{{ selectedEndpoint.url }}</span>
            <Button v-if="!readOnly && onOpenInBuilder" class="text-xs ml-auto" @click="onOpenInBuilder(selectedEndpoint)">
              Open in builder
            </Button>
          </div>
          <RequestDocsPanel
            :request="endpointAsRequest(selectedEndpoint)"
            :workspace-id="workspaceId"
            :editable="!readOnly"
            @save="onSaveEndpoint"
          />
        </template>
        <p v-else class="text-sm text-muted">Select an endpoint to view documentation.</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { CatalogEndpoint } from '~/stores/catalog'
import type { RequestItem } from '~/stores/collections'

const props = defineProps<{
  workspaceId: string
  readOnly?: boolean
  snapshotEndpoints?: CatalogEndpoint[]
  onOpenInBuilder?: (ep: CatalogEndpoint) => void
}>()

const catalogStore = useCatalogStore()
const colStore = useCollectionsStore()
const selectedCollectionId = ref<string | null>(null)
const selectedEndpoint = ref<CatalogEndpoint | null>(null)

const endpoints = computed(() => {
  if (props.snapshotEndpoints) return props.snapshotEndpoints
  return catalogStore.data?.endpoints || []
})

const filteredEndpoints = computed(() => {
  let list = endpoints.value
  if (selectedCollectionId.value) {
    list = list.filter(ep => ep.collection_id === selectedCollectionId.value)
  }
  return list
})

const allTags = computed(() => {
  const tags = new Set<string>()
  for (const ep of endpoints.value) {
    for (const t of ep.tags || []) tags.add(t)
  }
  return [...tags].sort()
})

onMounted(async () => {
  if (!props.snapshotEndpoints && props.workspaceId) {
    await catalogStore.fetchWorkspace(props.workspaceId)
  }
})

function onFilterUpdate(filters: Partial<typeof catalogStore.filters>) {
  catalogStore.setFilters(filters)
  if (!props.snapshotEndpoints) {
    catalogStore.fetchWorkspace(props.workspaceId, filters)
  }
}

function selectEndpoint(ep: CatalogEndpoint) {
  selectedEndpoint.value = ep
}

function endpointAsRequest(ep: CatalogEndpoint): RequestItem {
  return {
    id: ep.id,
    collection_id: ep.collection_id,
    name: ep.name,
    method: ep.method,
    url: ep.url,
    description: ep.description,
    api_doc: ep.api_doc,
    source_spec_id: ep.source_spec_id,
    source_operation_id: ep.source_operation_id,
    headers: [],
    params: [],
    body: { mode: 'none', raw: '', raw_lang: 'json' },
    auth: { type: 'none' },
    settings: { timeout_ms: 30000, follow_redirects: true, verify_ssl: true },
    pre_request_script: '',
    test_script: '',
  }
}

async function onSaveEndpoint(req: RequestItem) {
  await colStore.saveRequest(req)
  selectedEndpoint.value = {
    ...selectedEndpoint.value!,
    description: req.description || '',
    docs_complete: true,
  }
}
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}
.hover\:bg-surface-2:hover,
.bg-surface-2 {
  background: var(--color-surface-2);
}
</style>
