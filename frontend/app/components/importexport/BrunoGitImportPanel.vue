<template>
  <section class="rounded border p-3 space-y-2" style="border-color: var(--border)">
    <h4 class="font-medium text-sm">Import Bruno collection from Git</h4>
    <p class="text-xs text-muted">
      Paste a GitHub or GitLab repository URL containing Bruno <code>.bru</code> files.
      Browser links like <code>/-/tree/main/bruno</code> work too. This creates a new API collection in the workspace.
    </p>

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
      The folder must include a root <code>collection.bru</code> file. GitHub private repos need contents read access.
      GitLab needs <code>read_api</code> and <code>read_repository</code>.
    </p>

    <Button variant="primary" :disabled="importing || !canImport" @click="importFromGit">
      {{ importing ? 'Fetching and importing…' : 'Import from Git' }}
    </Button>

    <p v-if="progress" class="text-xs text-muted">{{ progress }}</p>
    <p v-if="error" class="text-xs text-red-400">{{ error }}</p>
    <p v-if="success" class="text-xs" style="color: var(--method-get)">{{ success }}</p>
  </section>
</template>

<script setup lang="ts">
import {
  applyGitLabBrowseUrlHints,
  detectedGitProvider,
  gitRepoConfigPayload,
} from '~/utils/gitRepoForm'

interface ImportResult {
  collection_id?: string
  collections: number
  requests: number
}

const props = defineProps<{ workspaceId: string }>()

const api = useApi()
const colStore = useCollectionsStore()

const form = ref({
  name: 'Imported Bruno',
  repo_url: '',
  branch: 'main',
  path_prefix: '',
  access_token: '',
})

const importing = ref(false)
const progress = ref('')
const error = ref('')
const success = ref('')

const provider = computed(() => detectedGitProvider(form.value.repo_url))
const canImport = computed(() => !!form.value.name.trim() && !!form.value.repo_url.trim())

watch(() => form.value.repo_url, () => applyGitLabBrowseUrlHints(form.value))

function formatResult(result: ImportResult) {
  const parts = []
  if (result.collections) parts.push(`${result.collections} folder(s)`)
  if (result.requests) parts.push(`${result.requests} request(s)`)
  return parts.length ? `Imported ${parts.join(', ')}` : 'Import produced no data'
}

async function importFromGit() {
  importing.value = true
  error.value = ''
  success.value = ''
  progress.value = 'Connecting to the repository and discovering Bruno files…'

  const controller = new AbortController()
  const timer = setTimeout(() => controller.abort(), 330_000)

  try {
    const payload = gitRepoConfigPayload(form.value)
    const result = await api.post<ImportResult>(
      `/api/workspaces/${props.workspaceId}/imports/bruno/git`,
      {
        name: form.value.name.trim(),
        ...payload,
        access_token: form.value.access_token || undefined,
      },
      { signal: controller.signal },
    )
    success.value = formatResult(result)
    form.value.access_token = ''
    progress.value = ''
    await colStore.fetchCollections(props.workspaceId)
    await colStore.fetchAllRequests(props.workspaceId)
  } catch (e) {
    progress.value = ''
    if (e instanceof Error && e.name === 'AbortError') {
      error.value = 'Import timed out — try a narrower path prefix or fewer files'
    } else {
      error.value = e instanceof Error ? e.message : 'Git import failed'
    }
  } finally {
    clearTimeout(timer)
    importing.value = false
  }
}
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}
</style>
