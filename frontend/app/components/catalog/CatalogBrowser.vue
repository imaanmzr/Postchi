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
          :tree="tree"
          :endpoints="filteredEndpoints"
          :collections="collections"
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
            <div v-if="!readOnly" class="ml-auto flex items-center gap-2">
              <Button
                class="text-xs shrink-0 inline-flex items-center gap-1.5"
                title="Copy a live link to this request's documentation"
                @click="copyLiveLink"
              >
                <Link :size="14" aria-hidden="true" />
                {{ liveLinkCopied ? 'Copied' : 'Copy live link' }}
              </Button>
              <NuxtLink
                :to="requestEditorUrl"
                class="ui-btn ui-btn-ghost text-xs shrink-0 inline-flex items-center gap-1.5"
                title="Open the full request editor to send, edit params, headers, and scripts"
              >
                <SquarePen :size="14" aria-hidden="true" />
                Open in request editor
              </NuxtLink>
            </div>
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
import type { CatalogCollection, CatalogEndpoint } from '~/stores/catalog'
import type { RequestItem, TreeNode } from '~/stores/collections'
import { copyToClipboard } from '~/utils/copyToClipboard'
import { buildCatalogRequestUrl, buildWorkspaceRequestUrl } from '~/utils/docLinks'
import { Link, SquarePen } from 'lucide-vue-next'

const props = defineProps<{
  workspaceId: string
  readOnly?: boolean
  initialEndpointId?: string | null
  snapshotEndpoints?: CatalogEndpoint[]
  snapshotCollections?: CatalogCollection[]
}>()

const emit = defineEmits<{
  'endpoint-selected': [id: string]
}>()

const catalogStore = useCatalogStore()
const colStore = useCollectionsStore()
const selectedEndpoint = ref<CatalogEndpoint | null>(null)
const liveLinkCopied = ref(false)

const endpoints = computed(() => {
  if (props.snapshotEndpoints) return props.snapshotEndpoints
  return catalogStore.data?.endpoints || []
})

const collections = computed(() => {
  if (props.snapshotCollections) return props.snapshotCollections
  return catalogStore.data?.collections || []
})

const tree = computed((): TreeNode[] => {
  if (!props.snapshotCollections) return colStore.tree
  return props.snapshotCollections.map((collection, index) => ({
    ...collection,
    workspace_id: props.workspaceId,
    sort_order: index,
    children: [],
  }))
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
  emit('endpoint-selected', ep.id)
}

watch(
  () => [props.initialEndpointId, endpoints.value] as const,
  ([id, availableEndpoints]) => {
    if (!id || selectedEndpoint.value?.id === id) return
    const endpoint = availableEndpoints.find(ep => ep.id === id)
    if (endpoint) selectedEndpoint.value = endpoint
  },
  { immediate: true },
)

async function copyLiveLink() {
  if (!selectedEndpoint.value || !import.meta.client) return
  const path = buildCatalogRequestUrl(props.workspaceId, selectedEndpoint.value.id)
  const copied = await copyToClipboard(new URL(path, window.location.origin).toString())
  if (!copied) return
  liveLinkCopied.value = true
  setTimeout(() => { liveLinkCopied.value = false }, 1500)
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
