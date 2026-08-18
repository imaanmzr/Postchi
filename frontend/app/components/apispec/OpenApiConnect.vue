<template>
  <div class="h-full overflow-y-auto p-6">
    <div class="max-w-2xl mx-auto space-y-6">
      <div>
        <h2 class="text-lg font-semibold tracking-tight">Connect to OpenAPI Spec</h2>
        <p class="text-sm mt-2 text-muted">
          Keep your collection synchronized with an OpenAPI specification. Changes in the spec will be detected automatically.
        </p>
      </div>

      <div class="rounded-lg border p-5 space-y-4" style="border-color: var(--color-border); background: var(--color-surface-1)">
        <label class="text-sm font-medium block">OpenAPI Specification</label>

        <div class="flex gap-1 p-0.5 rounded-md w-fit" style="background: var(--color-surface-2)">
          <button
            type="button"
            class="px-3 py-1 text-xs rounded transition"
            :class="sourceMode === 'url' ? 'source-active' : 'text-muted'"
            @click="sourceMode = 'url'"
          >
            URL
          </button>
          <button
            type="button"
            class="px-3 py-1 text-xs rounded transition"
            :class="sourceMode === 'file' ? 'source-active' : 'text-muted'"
            @click="sourceMode = 'file'"
          >
            File
          </button>
        </div>

        <div v-if="sourceMode === 'url'" class="flex gap-2">
          <Input
            v-model="specUrl"
            placeholder="https://api.example.com/openapi.json"
            class="flex-1 font-mono text-xs"
          />
          <Button variant="primary" :disabled="connecting || !specUrl.trim()" @click="connectUrl">
            {{ connecting ? 'Connecting…' : 'Connect' }}
          </Button>
        </div>

        <div v-else class="space-y-3">
          <input
            ref="fileInput"
            type="file"
            accept=".json,.yaml,.yml"
            class="text-xs ui-input w-full"
            @change="onFileSelect"
          />
          <Button variant="primary" :disabled="connecting || !selectedFile" @click="connectFile">
            {{ connecting ? 'Importing…' : 'Connect' }}
          </Button>
          <p class="text-xs text-muted">
            File import creates a collection from the spec. Ongoing sync requires connecting via URL.
          </p>
        </div>

        <p class="text-xs text-muted">
          Supports OpenAPI 3.x specifications in JSON or YAML format
        </p>

        <p v-if="errorMessage" class="text-sm" style="color: var(--method-delete)">{{ errorMessage }}</p>
        <p v-if="successMessage" class="text-sm" style="color: var(--method-get)">{{ successMessage }}</p>
      </div>

      <ul class="space-y-2 text-sm text-muted">
        <li v-for="feature in features" :key="feature" class="flex items-start gap-2">
          <span style="color: var(--method-get)">✓</span>
          <span>{{ feature }}</span>
        </li>
      </ul>

      <div v-if="linkedSpecs.length" class="space-y-3 pt-2">
        <h3 class="text-sm font-medium">Connected specs</h3>
        <div
          v-for="spec in linkedSpecs"
          :key="spec.id"
          class="rounded-lg border p-4 flex items-center justify-between gap-3"
          style="border-color: var(--color-border); background: var(--color-surface-1)"
        >
          <div class="min-w-0">
            <div class="font-medium text-sm truncate">{{ spec.name }}</div>
            <div class="text-xs text-muted truncate font-mono mt-0.5">{{ spec.spec_url }}</div>
            <div v-if="spec.last_synced_at" class="text-[10px] text-muted mt-1">
              Last synced {{ formatSynced(spec.last_synced_at) }}
            </div>
          </div>
          <div class="flex gap-2 shrink-0">
            <Button variant="primary" :disabled="syncingId === spec.id" @click="syncSpec(spec.id)">
              {{ syncingId === spec.id ? 'Syncing…' : 'Sync' }}
            </Button>
            <Button @click="deleteSpec(spec.id)">Remove</Button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { apiUrl } from '~/utils/apiBase'
import { normalizeSyncDiff } from '~/stores/apiSpecs'

const props = defineProps<{
  workspaceId: string
  collectionId?: string | null
}>()

const config = useRuntimeConfig()
const apiSpecsStore = useApiSpecsStore()
const colStore = useCollectionsStore()

const sourceMode = ref<'url' | 'file'>('url')
const specUrl = ref('')
const selectedFile = ref<File | null>(null)
const fileInput = ref<HTMLInputElement | null>(null)
const connecting = ref(false)
const syncingId = ref('')
const errorMessage = ref('')
const successMessage = ref('')

const features = [
  'Detect new, modified, and removed endpoints',
  'Track local changes against the spec',
  'Sync collection with a single click',
  'Your tests, assertions, and scripts are preserved during sync',
]

const linkedSpecs = computed(() =>
  apiSpecsStore.specs.filter(s => s.workspace_id === props.workspaceId),
)

onMounted(async () => {
  await apiSpecsStore.list(props.workspaceId)
})

function clearMessages() {
  errorMessage.value = ''
  successMessage.value = ''
}

function formatSynced(value: string) {
  try {
    return new Date(value).toLocaleString()
  } catch {
    return value
  }
}

function specNameFromUrl(url: string) {
  try {
    const host = new URL(url).hostname
    return host || 'OpenAPI Spec'
  } catch {
    return 'OpenAPI Spec'
  }
}

function onFileSelect(e: Event) {
  const input = e.target as HTMLInputElement
  selectedFile.value = input.files?.[0] ?? null
}

async function refreshWorkspace() {
  await colStore.fetchCollections(props.workspaceId)
  await colStore.fetchAllRequests(props.workspaceId)
  await apiSpecsStore.list(props.workspaceId)
}

async function connectUrl() {
  connecting.value = true
  clearMessages()
  try {
    await apiSpecsStore.create(props.workspaceId, {
      name: specNameFromUrl(specUrl.value),
      spec_url: specUrl.value.trim(),
      collection_id: props.collectionId || undefined,
    })
    specUrl.value = ''
    await refreshWorkspace()
    successMessage.value = 'Connected and endpoints imported successfully.'
  } catch (e) {
    errorMessage.value = e instanceof Error ? e.message : 'Failed to connect spec'
  } finally {
    connecting.value = false
  }
}

async function connectFile() {
  if (!selectedFile.value) return
  connecting.value = true
  clearMessages()
  try {
    const content = await selectedFile.value.text()
    const name = selectedFile.value.name.replace(/\.(json|ya?ml)$/i, '') || 'OpenAPI Import'
    const res = await fetch(apiUrl(config.public.apiUrl as string, `/api/import/openapi?workspace_id=${props.workspaceId}`), {
      method: 'POST',
      headers: {
        'Content-Type': 'application/octet-stream',
        Authorization: `Bearer ${useAuthStore().accessToken}`,
      },
      body: content,
    })
    if (!res.ok) {
      const err = await res.json().catch(() => ({}))
      throw new Error(err.error || 'Import failed')
    }
    selectedFile.value = null
    if (fileInput.value) fileInput.value.value = ''
    await refreshWorkspace()
    successMessage.value = `Imported "${name}" into a new collection.`
  } catch (e) {
    errorMessage.value = e instanceof Error ? e.message : 'Failed to import spec file'
  } finally {
    connecting.value = false
  }
}

async function syncSpec(id: string) {
  syncingId.value = id
  clearMessages()
  try {
    const diff = normalizeSyncDiff(await apiSpecsStore.sync(id, true))
    await refreshWorkspace()
    successMessage.value = `Sync complete: ${diff.added.length} added, ${diff.updated.length} updated, ${diff.removed.length} removed.`
  } catch (e) {
    errorMessage.value = e instanceof Error ? e.message : 'Sync failed'
  } finally {
    syncingId.value = ''
  }
}

async function deleteSpec(id: string) {
  clearMessages()
  try {
    await apiSpecsStore.delete(id)
    await refreshWorkspace()
    successMessage.value = 'Spec disconnected.'
  } catch (e) {
    errorMessage.value = e instanceof Error ? e.message : 'Failed to remove spec'
  }
}
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}
.source-active {
  background: var(--color-surface-1);
  color: var(--color-text);
  box-shadow: var(--shadow-sm);
}
</style>
