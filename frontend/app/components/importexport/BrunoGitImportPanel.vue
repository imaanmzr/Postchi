<template>
  <section class="rounded border p-3 space-y-4" style="border-color: var(--border)">
    <div>
      <h4 class="font-medium text-sm">Sync collections from Git</h4>
      <p class="text-xs text-muted mt-1">
        Connect a GitHub or GitLab repository containing Bruno, Postman, OpenCollection, or OpenAPI files.
        Changes in the repo appear after you click <strong>Sync now</strong>.
        Browser links like <code>/-/tree/main/collections</code> work too.
      </p>
    </div>

    <div class="space-y-2">
      <Input v-model="form.name" placeholder="Collection name" />
      <Input
        v-model="form.repo_url"
        placeholder="https://github.com/org/repo or GitLab /-/tree/ URL"
      />
      <p v-if="provider" class="text-xs text-muted">Detected: {{ provider }}</p>
      <Input v-model="form.branch" placeholder="Branch (default: main)" />
      <Input v-model="form.path_prefix" placeholder="Path prefix (optional, e.g. bruno)" />
      <Input
        v-model="form.access_token"
        type="password"
        autocomplete="off"
        placeholder="Personal access token (required for private repos and GitLab)"
      />
      <p class="text-xs text-muted">
        Bruno folders need a root <code>collection.bru</code> file. Postman, OpenCollection, and OpenAPI files are auto-detected.
        GitHub private repos need contents read access. GitLab needs <code>read_api</code> and <code>read_repository</code>.
      </p>
      <ImportParentPicker
        v-model="importParent"
        :workspace-id="workspaceId"
        :collections="colStore.collections"
      />
      <Button variant="primary" :disabled="creating || !canCreate" @click="createSource">
        {{ creating ? 'Connecting and syncing…' : 'Add git source' }}
      </Button>
    </div>

    <IndeterminateProgressBar
      v-if="creating"
      :label="progress || 'Connecting to the repository and running the first sync…'"
    />
    <p v-if="error" class="text-xs text-red-400">{{ error }}</p>
    <p v-if="success" class="text-xs" style="color: var(--method-get)">{{ success }}</p>

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
          <div v-if="src.config?.branch || src.config?.path_prefix" class="text-xs text-muted">
            Branch <code>{{ src.config?.branch || 'main' }}</code>
            <span v-if="src.config?.path_prefix"> · folder <code>{{ src.config.path_prefix }}</code></span>
          </div>
          <div v-if="src.last_synced_at" class="text-xs text-muted">
            Last synced {{ formatSyncedAt(src.last_synced_at) }}
          </div>
          <IndeterminateProgressBar
            v-if="syncing === src.id"
            class="mt-2"
            label="Syncing collections from repository…"
          />
          <p
            v-if="syncSuccessBySource[src.id]"
            class="text-xs mt-2"
            style="color: var(--method-get)"
          >
            {{ syncSuccessBySource[src.id] }}
          </p>
        </div>
        <div class="flex gap-2 shrink-0 flex-wrap justify-end">
          <Button class="text-xs" @click="toggleEdit(src)">
            {{ editingId === src.id ? 'Close' : 'Edit' }}
          </Button>
          <Button class="text-xs" :disabled="syncing === src.id" @click="syncSource(src.id)">
            {{ syncing === src.id ? 'Syncing…' : 'Sync now' }}
          </Button>
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

    <ConfirmDialog
      v-model:open="deleteOpen"
      title="Delete Bruno git source"
      :message="deleteMessage"
      confirm-label="Delete"
      destructive
      @confirm="confirmDeleteSource"
    />
  </section>
</template>

<script setup lang="ts">
import {
  applyGitLabBrowseUrlHints,
  detectedGitProvider,
  gitRepoConfigPayload,
} from '~/utils/gitRepoForm'
import {
  importParentPayload,
  isImportParentValid,
  type ImportParentChoice,
} from '~/utils/importParent'

interface BrunoSource {
  id: string
  name: string
  has_access_token?: boolean
  last_synced_at?: string
  config?: {
    provider?: string
    repo_url?: string
    branch?: string
    path_prefix?: string
  }
}

interface BrunoSyncResult {
  added_collections?: number
  updated_collections?: number
  added_requests?: number
  updated_requests?: number
  removed_requests?: number
  removed_collections?: number
}

const props = defineProps<{ workspaceId: string }>()

const api = useApi()
const colStore = useCollectionsStore()
const toast = useToast()

const sources = ref<BrunoSource[]>([])
const creating = ref(false)
const updating = ref(false)
const syncing = ref('')
const deleting = ref('')
const deleteOpen = ref(false)
const deleteTarget = ref<BrunoSource | null>(null)
const editingId = ref('')
const progress = ref('')
const error = ref('')
const success = ref('')
const syncSuccessBySource = ref<Record<string, string>>({})

const form = ref({
  name: 'Imported Bruno',
  repo_url: '',
  branch: 'main',
  path_prefix: '',
  access_token: '',
})
const importParent = ref<ImportParentChoice>({ mode: 'root' })

const editForm = ref({
  name: '',
  repo_url: '',
  branch: 'main',
  path_prefix: '',
  access_token: '',
})

const provider = computed(() => detectedGitProvider(form.value.repo_url))
const canCreate = computed(() => !!form.value.name.trim()
  && !!form.value.repo_url.trim()
  && isImportParentValid(importParent.value))

const deleteMessage = computed(() =>
  deleteTarget.value
    ? `Remove "${deleteTarget.value.name}" and delete its synced collection from this workspace?`
    : '',
)

watch(() => form.value.repo_url, () => applyGitLabBrowseUrlHints(form.value))
watch(() => editForm.value.repo_url, () => applyGitLabBrowseUrlHints(editForm.value))

onMounted(async () => {
  await colStore.fetchCollections(props.workspaceId)
  loadSources()
})

async function loadSources() {
  try {
    sources.value = await api.get<BrunoSource[]>(`/api/workspaces/${props.workspaceId}/bruno-sources`)
  } catch {
    sources.value = []
  }
}

function detectedProvider(url: string) {
  return detectedGitProvider(url)
}

function providerLabel(src: BrunoSource) {
  const p = src.config?.provider
  if (p === 'github') return 'GitHub'
  if (p === 'gitlab') return 'GitLab'
  return p || 'Git'
}

function formatSyncedAt(value: string) {
  try {
    return new Intl.DateTimeFormat(undefined, { dateStyle: 'medium', timeStyle: 'short' }).format(new Date(value))
  } catch {
    return value
  }
}

function formatSyncResult(result: BrunoSyncResult) {
  const parts: string[] = []
  if (result.added_requests) parts.push(`${result.added_requests} request(s) added`)
  if (result.updated_requests) parts.push(`${result.updated_requests} updated`)
  if (result.removed_requests) parts.push(`${result.removed_requests} removed`)
  if (result.added_collections) parts.push(`${result.added_collections} folder(s) added`)
  if (result.removed_collections) parts.push(`${result.removed_collections} folder(s) removed`)
  return parts.length ? parts.join(', ') : 'Already up to date'
}

async function refreshCollections() {
  await colStore.fetchCollections(props.workspaceId)
  await colStore.fetchAllRequests(props.workspaceId)
}

async function refreshAfterSync() {
  try {
    await loadSources()
    await refreshCollections()
  } catch {
    // Sync already succeeded; keep the success message even if UI refresh fails.
  }
}

function showSyncSuccess(sourceId: string, sourceName: string, result: BrunoSyncResult) {
  const message = `Sync complete for "${sourceName}": ${formatSyncResult(result)}`
  syncSuccessBySource.value = { [sourceId]: message }
  success.value = message
  toast.show(message, 'success', 5000)
}

async function createSource() {
  creating.value = true
  error.value = ''
  success.value = ''
  progress.value = 'Connecting to the repository and running the first sync…'

  const controller = new AbortController()
  const timer = setTimeout(() => controller.abort(), 330_000)

  try {
    const payload = gitRepoConfigPayload(form.value)
    const result = await api.post<{ source: BrunoSource; sync: BrunoSyncResult }>(
      `/api/workspaces/${props.workspaceId}/bruno-sources`,
      {
        name: form.value.name.trim(),
        config: payload,
        access_token: form.value.access_token || undefined,
        ...importParentPayload(importParent.value),
      },
      { signal: controller.signal },
    )
    success.value = `Connected "${result.source.name}". ${formatSyncResult(result.sync)}`
    toast.show(success.value, 'success', 5000)
    form.value.access_token = ''
    progress.value = ''
    await refreshAfterSync()
  } catch (e) {
    progress.value = ''
    if (e instanceof Error && e.name === 'AbortError') {
      error.value = 'Sync timed out - try a narrower path prefix or fewer files'
    } else {
      error.value = e instanceof Error ? e.message : 'Failed to add git source'
    }
  } finally {
    clearTimeout(timer)
    creating.value = false
  }
}

async function syncSource(sourceId: string) {
  const source = sources.value.find(s => s.id === sourceId)
  syncing.value = sourceId
  error.value = ''
  success.value = ''
  syncSuccessBySource.value = {}
  try {
    const result = await api.post<BrunoSyncResult>(`/api/bruno-sources/${sourceId}/sync`, {})
    showSyncSuccess(sourceId, source?.name || 'source', result)
    await refreshAfterSync()
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Sync failed'
  } finally {
    syncing.value = ''
  }
}

function toggleEdit(src: BrunoSource) {
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
    access_token: '',
  }
}

async function updateSource(sourceId: string) {
  updating.value = true
  error.value = ''
  try {
    const payload = gitRepoConfigPayload(editForm.value)
    await api.patch(`/api/workspaces/${props.workspaceId}/bruno-sources/${sourceId}`, {
      name: editForm.value.name.trim(),
      config: payload,
      access_token: editForm.value.access_token || undefined,
    })
    editingId.value = ''
    await loadSources()
    success.value = 'Source updated'
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Update failed'
  } finally {
    updating.value = false
  }
}

function askDeleteSource(src: BrunoSource) {
  deleteTarget.value = src
  deleteOpen.value = true
}

async function confirmDeleteSource() {
  const src = deleteTarget.value
  if (!src) return
  deleting.value = src.id
  error.value = ''
  try {
    await api.delete(`/api/workspaces/${props.workspaceId}/bruno-sources/${src.id}`)
    if (editingId.value === src.id) editingId.value = ''
    await loadSources()
    await refreshCollections()
    success.value = `Removed "${src.name}"`
  } catch (e) {
    error.value = e instanceof Error ? e.message : 'Delete failed'
  } finally {
    deleting.value = ''
    deleteTarget.value = null
  }
}
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}
</style>
