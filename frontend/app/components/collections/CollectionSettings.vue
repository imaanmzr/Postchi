<template>
  <div class="flex flex-col h-full overflow-hidden">
    <div class="px-4 py-3 border-b" style="border-color: var(--border)">
      <h2 class="font-semibold">{{ local.name }}</h2>
      <p v-if="local.description && activeTab !== 'overview'" class="text-sm mt-1" style="color: var(--text-secondary)">{{ local.description }}</p>
    </div>
    <div class="flex-1 ui-scroll-y p-4">
      <SubTabBar v-model="activeTab" :tabs="tabs" />
      <div class="mt-4">
        <div v-if="activeTab === 'overview'" class="grid grid-cols-1 lg:grid-cols-2 gap-6">
          <div class="space-y-4">
            <div class="rounded-lg border p-4 space-y-3" style="border-color: var(--color-border); background: var(--color-surface-1)">
              <div class="flex justify-between text-sm">
                <span class="text-muted">Location</span>
                <span class="font-mono text-xs truncate max-w-[200px]" :title="locationLabel">{{ locationLabel }}</span>
              </div>
              <div class="flex justify-between text-sm">
                <span class="text-muted">Version</span>
                <span>1.0.0</span>
              </div>
              <div class="flex justify-between text-sm">
                <span class="text-muted">Environments</span>
                <span>{{ envCount }}</span>
              </div>
              <div class="flex justify-between text-sm">
                <span class="text-muted">Requests</span>
                <span>{{ requestCount }}</span>
              </div>
            </div>

            <div>
              <label class="text-sm block mb-2">Name</label>
              <Input v-model="local.name" class="mb-3" />
              <label class="text-sm block mb-2">Description</label>
              <textarea v-model="local.description" class="ui-input w-full h-24" placeholder="Collection documentation (Markdown supported)" />
              <div class="mt-4 flex flex-wrap gap-2 justify-end">
                <Button @click="viewFullDocs">View full docs</Button>
                <Button @click="exportMarkdown">Export markdown</Button>
                <Button variant="primary" @click="saveOverview">Save</Button>
              </div>
            </div>
          </div>

          <div class="rounded-lg border p-4 min-h-[200px]" style="border-color: var(--color-border); background: var(--color-surface-1)">
            <h3 class="text-sm font-medium mb-3">Documentation</h3>
            <MarkdownViewer v-if="local.description" :content="local.description" />
            <p v-else class="text-sm text-muted">
              Add a description to document this collection. Markdown formatting is supported.
            </p>
          </div>
        </div>
        <div v-else-if="activeTab === 'docs'" class="space-y-4">
          <div v-if="docsLoading" class="text-sm text-muted">Loading documentation…</div>
          <MarkdownViewer v-else-if="fullDocsMarkdown" :content="fullDocsMarkdown" />
          <p v-else class="text-sm text-muted">No documentation generated yet.</p>
          <div class="flex gap-2">
            <Button @click="loadFullDocs">Refresh</Button>
            <Button @click="exportMarkdown">Download markdown</Button>
            <ShareButton
              v-if="local.id && wsStore.current?.id"
              :workspace-id="wsStore.current.id"
              kind="catalog"
              :source-id="local.id"
              :default-title="local.name + ' API Catalog'"
              button-class="text-xs"
            >
              Share catalog
            </ShareButton>
          </div>
        </div>
        <div v-else-if="activeTab === 'headers'">
          <KeyValueTable v-model="headerRows" />
          <div class="mt-4 flex justify-end"><Button variant="primary" @click="saveHeaders">Save</Button></div>
        </div>
        <div v-else-if="activeTab === 'vars'">
          <VarsTableEditor v-model="vars" @save="saveVars" />
        </div>
        <div v-else-if="activeTab === 'auth'">
          <AuthEditor
            v-model="local.auth"
            :inherit-label="authInheritLabel"
          />
          <div class="mt-4 flex justify-end"><Button variant="primary" @click="saveAuth">Save</Button></div>
        </div>
        <div v-else-if="activeTab === 'script'">
          <label class="text-xs" style="color: var(--text-secondary)">Pre-request</label>
          <ScriptEditor v-model="local.pre_request_script" class="mb-4" />
          <label class="text-xs" style="color: var(--text-secondary)">Tests</label>
          <ScriptEditor v-model="local.test_script" />
          <div class="mt-4 flex justify-end"><Button variant="primary" @click="saveScripts">Save</Button></div>
        </div>
        <div v-else-if="activeTab === 'tests'">
          <ScriptEditor v-model="local.test_script" />
          <div class="mt-4 flex justify-end"><Button variant="primary" @click="saveScripts">Save</Button></div>
        </div>
        <div v-else class="text-sm py-8 text-center" style="color: var(--text-secondary)">
          {{ stubMessage }}
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Collection } from '~/stores/collections'
import type { VariablesSpec } from '~/components/shared/VarsTableEditor'
import type { KVRow } from '~/components/shared/KeyValueTable'
import { inheritSourceLabel, resolveCollectionInheritedAuth } from '~/utils/authInheritance'

const props = defineProps<{
  collection: Collection
  initialTab?: string
}>()
const emit = defineEmits<{ saved: [] }>()

const colStore = useCollectionsStore()
const envStore = useEnvironmentsStore()
const wsStore = useWorkspaceStore()
const tabsStore = useTabsStore()

function collectionDraft(c: Collection) {
  const tab = tabsStore.openTabs.find(t => t.key === `collection:${c.id}`)
  return {
    ...c,
    name: c.name || tab?.label || c.name,
    auth: c.auth || { type: 'inherit' },
  }
}

const activeTab = ref(props.initialTab || 'overview')
const local = ref(collectionDraft(props.collection))
const vars = ref<VariablesSpec>(props.collection.variables || { pre_request: [], post_response: [] })
const headerRows = ref<KVRow[]>(
  (props.collection.headers || []).map(h => ({ key: h.key, value: h.value, enabled: h.enabled })),
)

watch(() => props.initialTab, (tab) => {
  if (tab) activeTab.value = tab
})

watch(() => props.collection, (c) => {
  local.value = collectionDraft(c)
  vars.value = c.variables || { pre_request: [], post_response: [] }
  headerRows.value = (c.headers || []).map(h => ({ key: h.key, value: h.value, enabled: h.enabled }))
}, { deep: true })

const varCount = computed(() => (vars.value.pre_request?.length || 0) + (vars.value.post_response?.length || 0))

const locationLabel = computed(() => {
  const ws = wsStore.current?.name || 'Workspace'
  return `${ws} / ${local.value.name}`
})

const envCount = computed(() => envStore.environments.length)

const authInheritLabel = computed(() => {
  if (local.value.auth?.type !== 'inherit') return ''
  const resolved = resolveCollectionInheritedAuth(local.value, colStore.collections)
  if (!resolved.source) return 'No auth configured in parent folders.'
  return inheritSourceLabel(resolved.source)
})

const requestCount = computed(() => {
  const ids = new Set<string>()
  function collect(id: string) {
    ids.add(id)
    for (const c of colStore.collections.filter(c => c.parent_id === id)) {
      collect(c.id)
    }
  }
  collect(props.collection.id)
  return colStore.requests.filter(r => ids.has(r.collection_id)).length
})

const tabs = computed(() => [
  { id: 'overview', label: 'Overview' },
  { id: 'docs', label: 'API Docs' },
  { id: 'headers', label: 'Headers', badge: headerRows.value.length || undefined },
  { id: 'vars', label: 'Vars', badge: varCount.value || undefined },
  { id: 'auth', label: 'Auth' },
  { id: 'script', label: 'Script' },
  { id: 'tests', label: 'Tests' },
  { id: 'presets', label: 'Presets' },
  { id: 'proxy', label: 'Proxy' },
  { id: 'certs', label: 'Client Certificates' },
  { id: 'secrets', label: 'Secrets' },
  { id: 'protobuf', label: 'Protobuf' },
])

const stubMessage = computed(() => {
  if (activeTab.value === 'protobuf') return 'Protobuf support coming soon.'
  return 'Configure via JSON in a future release. Storage is ready.'
})

async function saveOverview() {
  await colStore.updateCollection(local.value.id, { name: local.value.name, description: local.value.description })
  emit('saved')
}

function patchPayload(data: Partial<Collection>): Partial<Collection> {
  return local.value.name ? { name: local.value.name, ...data } : data
}

async function saveHeaders() {
  await colStore.updateCollection(local.value.id, patchPayload({
    headers: headerRows.value.filter(r => r.key).map(r => ({ key: r.key, value: r.value, enabled: r.enabled })),
  }))
  emit('saved')
}

async function saveVars() {
  await colStore.updateCollection(local.value.id, patchPayload({ variables: vars.value }))
  emit('saved')
}

async function saveAuth() {
  await colStore.updateCollection(local.value.id, patchPayload({ auth: local.value.auth }))
  emit('saved')
}

async function saveScripts() {
  await colStore.updateCollection(local.value.id, patchPayload({
    pre_request_script: local.value.pre_request_script,
    test_script: local.value.test_script,
  }))
  emit('saved')
}

const api = useApi()
const fullDocsMarkdown = ref('')
const docsLoading = ref(false)

async function loadFullDocs() {
  docsLoading.value = true
  try {
    fullDocsMarkdown.value = await api.fetchText(`/api/collections/${local.value.id}/docs`)
  } finally {
    docsLoading.value = false
  }
}

async function viewFullDocs() {
  activeTab.value = 'docs'
  await loadFullDocs()
}

async function exportMarkdown() {
  await api.download(`/api/collections/${local.value.id}/docs`, `${local.value.name.replace(/\s+/g, '-')}-api-docs.md`)
}
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}
</style>
