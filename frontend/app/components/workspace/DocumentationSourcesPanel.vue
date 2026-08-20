<template>
  <div class="space-y-6">
    <section>
      <h3 class="font-medium mb-2">Git markdown docs</h3>
      <p class="text-xs text-muted mb-3">
        Sync <code>.md</code> documentation from GitHub or GitLab — browser links like <code>/-/tree/main/docs</code> work too.
        To import Bruno API collections, use <strong>API Sync → Sync Bruno collection from Git</strong>.
      </p>
      <div class="space-y-2 mb-3">
        <Input v-model="gitForm.name" placeholder="Source name" />
        <Input
          v-model="gitForm.repo_url"
          placeholder="https://gitlab.com/group/project/-/tree/main/docs"
        />
        <p v-if="detectedProvider(gitForm.repo_url)" class="text-xs text-muted">
          Detected: {{ detectedProvider(gitForm.repo_url) }}
        </p>
        <Input v-model="gitForm.branch" placeholder="Branch (default: main)" />
        <Input v-model="gitForm.path_prefix" placeholder="Path prefix (optional, e.g. docs — auto-filled from browser links)" />
        <Input
          v-model="gitForm.link_template"
          placeholder="Link template (optional, e.g. docs/{collection_slug}/{request_slug}.md)"
        />
        <p class="text-xs text-muted">
          Template placeholders: <code>{request_slug}</code>, <code>{request_name}</code>, <code>{collection_slug}</code>, <code>{collection_name}</code>, <code>{method}</code>, <code>{operation_id}</code>, <code>{path_prefix}</code>
        </p>
        <select
          v-model="gitForm.collection_id"
          class="w-full text-sm rounded px-2 py-1.5 border"
          style="border-color: var(--color-border); background: var(--color-surface-1)"
        >
          <option value="">API collection (optional — scopes auto-linking)</option>
          <option v-for="col in workspaceCollections" :key="col.id" :value="col.id">
            {{ col.name }}
          </option>
        </select>
        <Input
          v-model="gitForm.access_token"
          type="password"
          autocomplete="off"
          placeholder="Personal access token (required for private repos)"
        />
        <p class="text-xs text-muted">
          GitHub: <code>contents:read</code>. GitLab: <code>read_api</code> + <code>read_repository</code> (Reporter role or higher on the project).
        </p>
        <div class="flex flex-wrap gap-2 items-center">
          <Button variant="primary" :disabled="creating" @click="createGitSource">
            {{ creating ? 'Creating…' : 'Add git source' }}
          </Button>
          <Button
            class="text-xs"
            :disabled="analyzingLinks || backfilling"
            @click="analyzeDocLinks"
          >
            {{ analyzingLinks ? 'Analyzing…' : 'Analyze doc links' }}
          </Button>
          <Button
            class="text-xs"
            :disabled="backfilling || analyzingLinks"
            @click="backfillOperationIds"
          >
            {{ backfilling ? 'Backfilling…' : 'Backfill API operation IDs' }}
          </Button>
        </div>
        <p v-if="linkAnalyzeMessage" class="text-xs text-muted">{{ linkAnalyzeMessage }}</p>
        <p v-if="syncMessage" class="text-xs text-muted">{{ syncMessage }}</p>
        <p v-if="syncError" class="text-xs text-red-400">{{ syncError }}</p>
      </div>

      <div
        v-for="src in sources"
        :key="src.id"
        class="py-3 border-t text-sm space-y-2"
        style="border-color: var(--border)"
      >
        <div class="flex items-start gap-2">
          <div class="flex-1 min-w-0">
            <div class="font-medium flex items-center gap-2">
              <span>{{ src.name }}</span>
              <span
                v-if="src.has_access_token"
                class="text-[10px] uppercase tracking-wide px-1.5 py-0.5 rounded"
                style="background: var(--color-surface-2); color: var(--color-text-muted)"
              >
                Private
              </span>
            </div>
            <div class="text-xs text-muted truncate">
              {{ providerLabel(src) }} · {{ src.config?.repo_url }}
            </div>
            <div v-if="src.config?.branch || src.config?.path_prefix || src.config?.link_template" class="text-xs text-muted">
              Branch <code>{{ src.config?.branch || 'main' }}</code>
              <span v-if="src.config?.path_prefix"> · folder <code>{{ src.config.path_prefix }}</code></span>
              <span v-if="src.config?.link_template"> · template <code>{{ src.config.link_template }}</code></span>
            </div>
          </div>
          <div class="flex gap-2 shrink-0">
            <Button class="text-xs" @click="toggleEdit(src)">
              {{ editingId === src.id ? 'Close' : 'Edit' }}
            </Button>
            <Button class="text-xs" :disabled="syncing === src.id" @click="syncSource(src.id)">
              {{ syncing === src.id ? 'Syncing…' : 'Sync now' }}
            </Button>
            <label class="text-[10px] text-muted inline-flex items-center gap-1 cursor-pointer">
              <input v-model="analyzeAfterSync" type="checkbox" class="rounded">
              Analyze after sync
            </label>
            <Button
              class="text-xs text-red-400 hover:text-red-300"
              :disabled="deleting === src.id"
              @click="askDeleteSource(src)"
            >
              {{ deleting === src.id ? 'Deleting…' : 'Delete' }}
            </Button>
          </div>
        </div>

        <div v-if="editingId === src.id" class="space-y-2 pt-1">
          <Input v-model="editForm.name" placeholder="Source name" />
          <Input v-model="editForm.repo_url" placeholder="Repository URL" />
          <p v-if="detectedProvider(editForm.repo_url)" class="text-xs text-muted">
            Detected: {{ detectedProvider(editForm.repo_url) }}
          </p>
          <Input v-model="editForm.branch" placeholder="Branch" />
          <Input v-model="editForm.path_prefix" placeholder="Path prefix" />
          <Input
            v-model="editForm.link_template"
            placeholder="Link template (e.g. docs/{collection_slug}/{request_slug}.md)"
          />
          <select
            v-model="editForm.collection_id"
            class="w-full text-sm rounded px-2 py-1.5 border"
            style="border-color: var(--color-border); background: var(--color-surface-1)"
          >
            <option value="">API collection (optional)</option>
            <option v-for="col in workspaceCollections" :key="col.id" :value="col.id">
              {{ col.name }}
            </option>
          </select>
          <Input
            v-model="editForm.access_token"
            type="password"
            autocomplete="off"
            :placeholder="src.has_access_token ? 'Replace access token (leave blank to keep)' : 'Access token'"
          />
          <Button variant="primary" class="text-xs" :disabled="updating" @click="updateSource(src.id)">
            {{ updating ? 'Saving…' : 'Save changes' }}
          </Button>
        </div>
      </div>
    </section>

    <ConfirmDialog
      v-model:open="deleteOpen"
      title="Delete git source"
      :message="deleteMessage"
      confirm-label="Delete"
      destructive
      @confirm="confirmDeleteSource"
    />

    <section class="border-t pt-4" style="border-color: var(--border)">
      <h3 class="font-medium mb-2">CI automation tokens</h3>
      <p class="text-xs text-muted mb-3">Generate tokens for pipelines to push OpenAPI specs via POST /api/workspaces/:id/api-specs/push</p>
      <div class="flex gap-2 mb-3">
        <Input v-model="tokenName" placeholder="Token name" class="flex-1" />
        <Button variant="primary" :disabled="creatingToken" @click="createToken">{{ creatingToken ? 'Creating…' : 'Generate' }}</Button>
      </div>
      <p v-if="newToken" class="text-xs p-2 rounded font-mono mb-3 break-all" style="background: var(--color-surface-2)">
        Copy now — won't be shown again: {{ newToken }}
      </p>
      <div v-for="t in tokens" :key="t.id" class="flex items-center gap-2 py-2 border-t text-sm" style="border-color: var(--border)">
        <div class="flex-1">
          <div class="font-medium">{{ t.name }}</div>
          <div class="text-xs text-muted">{{ t.token_prefix }}… · {{ t.scopes?.join(', ') }}</div>
        </div>
        <Button class="text-xs" @click="revokeToken(t.id)">Revoke</Button>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import {
  applyGitLabBrowseUrlHints,
  detectedGitProvider,
  gitRepoConfigPayload,
} from '~/utils/gitRepoForm'

interface DocSource {
  id: string
  name: string
  source_type: string
  collection_id?: string
  has_access_token?: boolean
  config?: {
    provider?: string
    repo_url?: string
    branch?: string
    path_prefix?: string
    link_template?: string
  }
}

const props = defineProps<{ workspaceId: string }>()
const api = useApi()
const colStore = useCollectionsStore()

const workspaceCollections = computed(() =>
  colStore.collections.filter(c => c.workspace_id === props.workspaceId),
)

const sources = ref<DocSource[]>([])
const tokens = ref<any[]>([])
const creating = ref(false)
const updating = ref(false)
const creatingToken = ref(false)
const syncing = ref('')
const deleting = ref('')
const deleteOpen = ref(false)
const deleteTarget = ref<DocSource | null>(null)
const syncError = ref('')
const syncMessage = ref('')
const linkAnalyzeMessage = ref('')
const analyzingLinks = ref(false)
const backfilling = ref(false)
const editingId = ref('')
const newToken = ref('')
const tokenName = ref('CI push token')

const gitForm = ref({
  name: 'Git docs',
  repo_url: '',
  branch: 'main',
  path_prefix: '',
  link_template: '',
  collection_id: '',
  access_token: '',
})

const editForm = ref({
  name: '',
  repo_url: '',
  branch: 'main',
  path_prefix: '',
  link_template: '',
  collection_id: '',
  access_token: '',
})

watch(() => gitForm.value.repo_url, () => applyGitLabBrowseUrlHints(gitForm.value))
watch(() => editForm.value.repo_url, () => {
  if (editingId.value) applyGitLabBrowseUrlHints(editForm.value)
})

onMounted(async () => {
  await colStore.fetchCollections(props.workspaceId)
  await load()
})

async function load() {
  sources.value = await api.get(`/api/workspaces/${props.workspaceId}/doc-sources`)
  tokens.value = await api.get(`/api/workspaces/${props.workspaceId}/api-tokens`)
}

function detectedProvider(repoUrl: string): string {
  return detectedGitProvider(repoUrl)
}

function providerLabel(src: DocSource) {
  const p = src.config?.provider || detectedGitProvider(src.config?.repo_url || '').toLowerCase()
  return p === 'gitlab' ? 'GitLab' : 'GitHub'
}

const deleteMessage = computed(() => {
  if (!deleteTarget.value) return ''
  return `Remove "${deleteTarget.value.name}" and all pages synced from this source? Manual doc links to those pages will also be removed.`
})

function askDeleteSource(src: DocSource) {
  deleteTarget.value = src
  deleteOpen.value = true
}

async function confirmDeleteSource() {
  const src = deleteTarget.value
  if (!src) return
  deleteOpen.value = false
  deleting.value = src.id
  syncError.value = ''
  try {
    await api.delete(`/api/workspaces/${props.workspaceId}/doc-sources/${src.id}`)
    if (editingId.value === src.id) editingId.value = ''
    await load()
  } catch (e) {
    syncError.value = e instanceof Error ? e.message : 'Delete failed'
  } finally {
    deleting.value = ''
    deleteTarget.value = null
  }
}

async function createGitSource() {
  creating.value = true
  try {
    await api.post(`/api/workspaces/${props.workspaceId}/doc-sources`, {
      name: gitForm.value.name,
      source_type: 'git',
      collection_id: gitForm.value.collection_id || undefined,
      config: gitRepoConfigPayload(gitForm.value),
      access_token: gitForm.value.access_token || undefined,
    })
    gitForm.value.access_token = ''
    await load()
  } finally {
    creating.value = false
  }
}

function toggleEdit(src: DocSource) {
  if (editingId.value === src.id) {
    editingId.value = ''
    return
  }
  editingId.value = src.id
  editForm.value = {
    name: src.name,
    repo_url: src.config?.repo_url || '',
    branch: src.config?.branch || 'main',
    path_prefix: src.config?.path_prefix || '',
    link_template: src.config?.link_template || '',
    collection_id: src.collection_id || '',
    access_token: '',
  }
}

async function updateSource(id: string) {
  updating.value = true
  try {
    const body: Record<string, unknown> = {
      name: editForm.value.name,
      config: gitRepoConfigPayload(editForm.value),
      collection_id: editForm.value.collection_id || null,
    }
    if (editForm.value.access_token) {
      body.access_token = editForm.value.access_token
    }
    await api.patch(`/api/workspaces/${props.workspaceId}/doc-sources/${id}`, body)
    editingId.value = ''
    await load()
  } finally {
    updating.value = false
  }
}

async function syncSource(id: string) {
  syncing.value = id
  syncError.value = ''
  syncMessage.value = ''
  linkAnalyzeMessage.value = ''
  const controller = new AbortController()
  const timer = setTimeout(() => controller.abort(), 330_000)
  try {
    const result = await api.post<{ synced?: number, total?: number, capped?: number, errors?: number, auto_linked?: number }>(`/api/doc-sources/${id}/sync`, undefined, {
      signal: controller.signal,
    })
    if (result?.synced != null) {
      syncError.value = ''
      const parts = [`Synced ${result.synced} doc(s)`]
      if (result.total != null) parts.push(`of ${result.total} found`)
      if (result.capped) parts.push(`(${result.capped} skipped — over limit)`)
      if (result.errors) parts.push(`(${result.errors} errors)`)
      if (result.auto_linked) parts.push(`${result.auto_linked} auto-linked`)
      syncMessage.value = parts.join(' ')
    }
    if (analyzeAfterSync.value) {
      await analyzeDocLinks()
    }
    await load()
  } catch (e) {
    if (e instanceof Error && e.name === 'AbortError') {
      syncError.value = 'Sync timed out — try a narrower path prefix (e.g. docs) or fewer files'
    } else {
      syncError.value = e instanceof Error ? e.message : 'Sync failed'
    }
  } finally {
    clearTimeout(timer)
    syncing.value = ''
  }
}

const analyzeAfterSync = ref(false)

async function analyzeDocLinks() {
  analyzingLinks.value = true
  linkAnalyzeMessage.value = ''
  try {
    const docsStore = useDocsStore()
    const result = await docsStore.analyzeLinks(props.workspaceId)
    linkAnalyzeMessage.value = `Analysis complete: ${result.auto_linked ?? 0} auto-linked, ${result.pending_total ?? 0} pending suggestion(s).`
  } catch (e) {
    linkAnalyzeMessage.value = e instanceof Error ? e.message : 'Analyze failed'
  } finally {
    analyzingLinks.value = false
  }
}

async function backfillOperationIds() {
  backfilling.value = true
  linkAnalyzeMessage.value = ''
  try {
    const docsStore = useDocsStore()
    const result = await docsStore.backfillOperationIds(props.workspaceId)
    linkAnalyzeMessage.value = `Backfilled ${result.updated ?? 0} request(s); ${result.skipped ?? 0} skipped.`
  } catch (e) {
    linkAnalyzeMessage.value = e instanceof Error ? e.message : 'Backfill failed'
  } finally {
    backfilling.value = false
  }
}

async function createToken() {
  creatingToken.value = true
  try {
    const res = await api.post<{ token: string }>(`/api/workspaces/${props.workspaceId}/api-tokens`, {
      name: tokenName.value,
      scopes: ['spec:push'],
    })
    newToken.value = res.token
    await load()
  } finally {
    creatingToken.value = false
  }
}

async function revokeToken(id: string) {
  await api.delete(`/api/workspaces/${props.workspaceId}/api-tokens/${id}`)
  await load()
}
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}
</style>
