<template>
  <div class="flex flex-col h-full overflow-hidden">
    <CatalogFilters
      :filters="catalogStore.filters"
      :tags="allTags"
      @update="onFilterUpdate"
    />

    <div class="flex flex-1 min-h-0">
      <aside
        class="shrink-0 border-r flex flex-col min-h-0 overflow-hidden"
        style="width: 300px; border-color: var(--color-border); background: var(--color-surface-1)"
      >
        <CatalogTree
          :workspace-id="workspaceId"
          :tree="colStore.tree"
          :endpoints="filteredEndpoints"
          :collections="catalogStore.data?.collections || []"
          :selected-id="selectedEndpoint?.id"
          :loading="catalogStore.loading"
          @select="selectEndpoint"
        />
      </aside>

      <div class="flex-1 ui-scroll-y p-4 min-w-0">
        <template v-if="selectedEndpoint">
          <div class="flex items-center gap-2 mb-4 flex-wrap">
            <MethodBadge :method="selectedEndpoint.method" />
            <h2 class="font-semibold">{{ selectedEndpoint.name }}</h2>
            <span class="font-mono text-sm text-muted truncate min-w-0">{{ selectedEndpoint.url }}</span>
            <NuxtLink
              v-if="!readOnly"
              :to="requestEditorUrl"
              class="ui-btn ui-btn-ghost text-xs ml-auto shrink-0 inline-flex items-center gap-1.5"
              title="Open the full request editor to send, edit params, headers, and scripts"
            >
              <SquarePen :size="14" aria-hidden="true" />
              Open in request editor
            </NuxtLink>
          </div>
          <RequestDocsPanel
            :request="endpointAsRequest(selectedEndpoint)"
            :workspace-id="workspaceId"
            :editable="!readOnly"
            @save="onSaveEndpoint"
            @docs-changed="refreshCatalogSelection"
          />
        </template>
        <p v-else class="text-sm text-muted">Select an endpoint from the tree to view or edit its documentation.</p>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { CatalogEndpoint } from '~/stores/catalog'
import type { RequestItem } from '~/stores/collections'
import { buildWorkspaceRequestUrl } from '~/utils/docLinks'
import { SquarePen } from 'lucide-vue-next'

const props = defineProps<{
  workspaceId: string
  readOnly?: boolean
  snapshotEndpoints?: CatalogEndpoint[]
}>()

const catalogStore = useCatalogStore()
const colStore = useCollectionsStore()
const selectedEndpoint = ref<CatalogEndpoint | null>(null)

const endpoints = computed(() => {
  if (props.snapshotEndpoints) return props.snapshotEndpoints
  return catalogStore.data?.endpoints || []
})

const filteredEndpoints = computed(() => endpoints.value)

const requestEditorUrl = computed(() => {
  if (!selectedEndpoint.value) return '#'
  return buildWorkspaceRequestUrl(props.workspaceId, selectedEndpoint.value.id)
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
    await Promise.all([
      catalogStore.fetchWorkspace(props.workspaceId),
      colStore.fetchCollections(props.workspaceId),
    ])
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

async function refreshCatalogSelection() {
  if (props.snapshotEndpoints || !selectedEndpoint.value) return
  const currentId = selectedEndpoint.value.id
  await catalogStore.fetchWorkspace(props.workspaceId)
  const updated = catalogStore.data?.endpoints.find(e => e.id === currentId)
  if (updated) selectedEndpoint.value = updated
}

async function onSaveEndpoint(req: RequestItem) {
  await colStore.saveRequest(req)
  selectedEndpoint.value = {
    ...selectedEndpoint.value!,
    description: req.description || '',
    docs_complete: true,
  }
  await refreshCatalogSelection()
}
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}
</style>
