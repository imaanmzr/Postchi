<template>
  <div class="min-h-screen p-4" style="background: var(--color-bg)">
    <div
      class="mx-auto rounded-lg border p-6"
      :class="share?.kind === 'catalog' ? 'max-w-6xl' : 'max-w-3xl'"
      style="background: var(--surface); border-color: var(--border)"
    >
      <template v-if="loading">
        <p style="color: var(--text-secondary)">Loading share…</p>
      </template>
      <template v-else-if="gone">
        <h1 class="text-xl font-semibold mb-2">Share unavailable</h1>
        <p style="color: var(--text-secondary)">This share is no longer available. It may have expired or been revoked.</p>
      </template>
      <template v-else-if="share">
        <template v-if="share.kind === 'catalog'">
          <h1 class="text-xl font-semibold mb-1">{{ share.title || 'API Catalog' }}</h1>
          <p class="text-sm mb-4" style="color: var(--text-secondary)">Read-only API documentation snapshot</p>
          <CatalogBrowser
            read-only
            :workspace-id="share.workspace_id"
            :snapshot-endpoints="catalogEndpoints"
            :snapshot-collections="catalogCollections"
            :initial-endpoint-id="catalogLandingRequestId"
          />
        </template>
        <template v-else>
        <h1 class="text-xl font-semibold mb-1">{{ share.title || 'Shared request' }}</h1>
        <p class="text-sm mb-4" style="color: var(--text-secondary)">{{ share.kind === 'history' ? 'Request + response snapshot' : 'Request snapshot' }}</p>

        <div v-if="requestData?.description" class="mb-4">
          <h3 class="text-sm font-medium mb-1">Documentation</h3>
          <MarkdownViewer :content="requestData.description" />
        </div>

        <div v-if="requestData" class="space-y-4 mb-6">
          <div class="flex gap-2 items-center font-mono text-sm">
            <MethodBadge :method="requestData.method" />
            <span class="truncate">{{ requestData.url }}</span>
          </div>
          <div v-if="requestData.headers?.length">
            <h3 class="text-sm font-medium mb-1">Headers</h3>
            <pre class="text-xs p-2 rounded font-mono overflow-auto" style="background: var(--color-surface-2)">{{ formatKV(requestData.headers) }}</pre>
          </div>
          <div v-if="requestData.params?.length">
            <h3 class="text-sm font-medium mb-1">Params</h3>
            <pre class="text-xs p-2 rounded font-mono overflow-auto" style="background: var(--color-surface-2)">{{ formatKV(requestData.params) }}</pre>
          </div>
          <div v-if="requestData.body?.raw">
            <h3 class="text-sm font-medium mb-1">Body</h3>
            <pre class="text-xs p-2 rounded font-mono overflow-auto max-h-48" style="background: var(--color-surface-2)">{{ formattedRequestBody }}</pre>
          </div>
        </div>

        <div v-if="responseData && share.kind === 'history'" class="mb-6 border rounded overflow-hidden" style="border-color: var(--border)">
          <div class="px-3 py-2 border-b text-sm font-medium flex items-center justify-between" style="border-color: var(--border)">
            <span>Response</span>
            <span v-if="executedByLabel" class="text-xs text-muted font-normal">Run by {{ executedByLabel }}</span>
          </div>
          <ResponseViewer :response="sharedResponseView" class="h-96" />
        </div>

        <div v-if="auth.isAuthenticated" class="border-t pt-4" style="border-color: var(--border)">
          <h3 class="font-medium mb-3">Import into my workspace</h3>
          <VariableMappingStep
            v-if="importStep === 'vars'"
            :workspace-id="importWorkspaceId"
            :placeholder-names="placeholderNames"
            @done="onVarsDone"
            @cancel="importStep = 'pick'"
          />
          <template v-else>
            <div class="space-y-3 mb-4">
              <div>
                <label class="text-sm block mb-1">Workspace</label>
                <Select v-model="importWorkspaceId">
                  <option v-for="ws in workspaces" :key="ws.id" :value="ws.id">{{ ws.name }}</option>
                </Select>
              </div>
              <div>
                <label class="text-sm block mb-1">Collection</label>
                <Select v-model="importCollectionId">
                  <option v-for="c in importCollections" :key="c.id" :value="c.id">{{ c.name }}</option>
                </Select>
              </div>
            </div>
            <p v-if="importError" class="text-sm mb-2" style="color: var(--method-delete)">{{ importError }}</p>
            <Button variant="primary" :disabled="importing || !importCollectionId" @click="startImport">
              {{ importing ? 'Importing…' : 'Import copy' }}
            </Button>
          </template>
        </div>
        <p v-else class="text-sm" style="color: var(--text-secondary)">
          <NuxtLink to="/login" style="color: var(--accent)">Sign in</NuxtLink> to import this request into your workspace.
        </p>
        </template>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { CatalogCollection, CatalogEndpoint } from '~/stores/catalog'
import type { Share } from '~/stores/shares'
import type { RequestItem } from '~/stores/collections'
import { extractPlaceholdersFromRequest } from '~/utils/placeholders'
import { formatRequestBody } from '~/utils/formatRequestBody'

definePageMeta({ layout: false })

const route = useRoute()
const auth = useAuthStore()
const sharesStore = useSharesStore()
const wsStore = useWorkspaceStore()
const colStore = useCollectionsStore()
const envStore = useEnvironmentsStore()

const token = computed(() => route.params.token as string)
const loading = ref(true)
const gone = ref(false)
const share = ref<Share | null>(null)
const importWorkspaceId = ref('')
const importCollectionId = ref('')
const importError = ref('')
const importing = ref(false)
const importStep = ref<'pick' | 'vars'>('pick')
const pendingEnvId = ref('')

const workspaces = computed(() => wsStore.workspaces)
const importCollections = computed(() =>
  colStore.collections.filter(c => c.workspace_id === importWorkspaceId.value),
)

const catalogEndpoints = computed((): CatalogEndpoint[] => {
  if (!share.value?.snapshot || share.value.kind !== 'catalog') return []
  const endpoints = share.value.snapshot.endpoints as any[]
  if (!Array.isArray(endpoints)) return []
  return endpoints.map(ep => ({
    id: ep.id,
    collection_id: ep.collection_id || share.value!.source_id,
    collection_name: ep.collection_name || (share.value!.snapshot.collection as any)?.name || '',
    name: ep.name,
    method: ep.method,
    url: ep.url,
    description: ep.description || '',
    tags: [],
    response_codes: ep.api_doc?.responses ? Object.keys(ep.api_doc.responses) : [],
    api_doc: ep.api_doc || {},
    docs_complete: !!(ep.description || ep.api_doc?.responses),
  }))
})

const catalogCollections = computed((): CatalogCollection[] => {
  if (!share.value?.snapshot || share.value.kind !== 'catalog') return []
  const rawCollections = share.value.snapshot.collections as any[]
  if (Array.isArray(rawCollections) && rawCollections.length) {
    return rawCollections.map(collection => ({
      id: collection.id,
      name: collection.name,
      description: collection.description || '',
      request_count: catalogEndpoints.value.filter(ep => ep.collection_id === collection.id).length,
      documented_count: catalogEndpoints.value.filter(ep => ep.collection_id === collection.id && ep.docs_complete).length,
    }))
  }

  const collection = share.value.snapshot.collection as Record<string, unknown> | undefined
  if (!collection?.id || !collection.name) return []
  return [{
    id: collection.id as string,
    name: collection.name as string,
    description: (collection.description as string) || '',
    request_count: catalogEndpoints.value.length,
    documented_count: catalogEndpoints.value.filter(ep => ep.docs_complete).length,
  }]
})

const catalogLandingRequestId = computed(() => {
  if (!share.value?.snapshot || share.value.kind !== 'catalog') return ''
  const requestId = share.value.snapshot.landing_request_id
  return typeof requestId === 'string' ? requestId : ''
})

const requestData = computed(() => {
  if (!share.value?.snapshot || share.value.kind === 'catalog') return null
  if (share.value.kind === 'history') {
    return share.value.snapshot.snapshot as RequestItem
  }
  return share.value.snapshot as RequestItem
})

const responseData = computed(() => {
  if (!share.value?.snapshot || share.value.kind !== 'history') return null
  return share.value.snapshot.response as Record<string, unknown>
})

const sharedResponseView = computed(() => {
  if (!share.value?.snapshot || share.value.kind !== 'history') return null
  const snap = share.value.snapshot
  const resp = (snap.response || {}) as Record<string, unknown>
  const body = typeof resp.body === 'string' ? resp.body : ''
  return {
    status_code: (snap.status_code as number) ?? (resp.status_code as number) ?? 0,
    body,
    headers: (resp.headers as Record<string, string>) ?? {},
    timing: (resp.timing as Record<string, number>) ?? { total_ms: snap.duration_ms as number },
    test_results: (snap.test_results as unknown[]) ?? (resp.test_results as unknown[]),
    console: (resp.console as string[]) ?? [],
    body_size: body.length,
  }
})

const executedByLabel = computed(() => {
  if (!share.value?.snapshot) return ''
  const name = share.value.snapshot.executed_by_name as string | undefined
  const email = share.value.snapshot.executed_by_email as string | undefined
  return name || email || ''
})

const placeholderNames = computed(() => {
  if (!requestData.value) return []
  return extractPlaceholdersFromRequest(requestData.value)
})

function formatKV(rows: { key: string; value: string; enabled?: boolean }[]) {
  return rows.filter(r => r.enabled).map(r => `${r.key}: ${r.value}`).join('\n')
}

const formattedRequestBody = computed(() => {
  if (!requestData.value?.body?.raw) return ''
  return formatRequestBody(requestData.value.body).raw
})

onMounted(async () => {
  try {
    share.value = await sharesStore.fetchByToken(token.value)
  } catch (e: any) {
    if (e.message?.includes('410') || e.message?.includes('no longer')) {
      gone.value = true
    }
  } finally {
    loading.value = false
  }
  await auth.ensureSession()
  if (auth.isAuthenticated) {
    await wsStore.fetchWorkspaces()
    if (wsStore.workspaces.length) {
      importWorkspaceId.value = wsStore.workspaces[0].id
      await colStore.fetchCollections(importWorkspaceId.value)
      await envStore.fetch(importWorkspaceId.value)
    }
  }
})

watch(importWorkspaceId, async (id) => {
  if (!id) return
  await colStore.fetchCollections(id)
  await envStore.fetch(id)
  importCollectionId.value = colStore.collections[0]?.id || ''
})

async function startImport() {
  importError.value = ''
  if (placeholderNames.value.length && envStore.activeId) {
    const { missing } = await envStore.resolveVariables(envStore.activeId, placeholderNames.value)
    if (missing.length) {
      importStep.value = 'vars'
      return
    }
  }
  await doImport()
}

async function onVarsDone(envId: string) {
  pendingEnvId.value = envId
  importStep.value = 'pick'
  await doImport(envId)
}

async function doImport(envId?: string) {
  importing.value = true
  importError.value = ''
  try {
    const result = await sharesStore.importShare(token.value, {
      workspace_id: importWorkspaceId.value,
      collection_id: importCollectionId.value,
      target_environment_id: envId || pendingEnvId.value || envStore.activeId || undefined,
    })
    await navigateTo(`/workspaces/${importWorkspaceId.value}?request=${result.request_id}`)
  } catch (e: any) {
    importError.value = e.message
  } finally {
    importing.value = false
  }
}
</script>
