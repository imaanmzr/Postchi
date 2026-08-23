<template>
  <div class="space-y-4">
    <p v-if="errorMessage" class="text-sm" style="color: var(--method-delete)">{{ errorMessage }}</p>
    <p v-if="successMessage" class="text-sm" style="color: var(--method-get)">{{ successMessage }}</p>

    <div class="rounded border p-3 space-y-2" style="border-color: var(--border)">
      <h4 class="font-medium text-sm">Add OpenAPI spec</h4>
      <p class="text-xs" style="color: var(--text-secondary)">
        Endpoints are imported into a collection automatically when you add a spec.
      </p>
      <Input v-model="newSpec.name" placeholder="Spec name" />
      <Input v-model="newSpec.spec_url" placeholder="https://example.com/openapi.json" />
      <Select v-model="newSpec.collection_id">
        <option value="">Create new collection</option>
        <option v-for="c in collections" :key="c.id" :value="c.id">{{ c.name }}</option>
      </Select>
      <Button variant="primary" :disabled="creating" @click="createSpec">{{ creating ? 'Adding…' : 'Add spec' }}</Button>
      <IndeterminateProgressBar
        v-if="creating"
        label="Fetching OpenAPI spec and importing endpoints…"
      />
    </div>

    <div v-for="spec in apiSpecsStore.specs" :key="spec.id" class="rounded border p-3" style="border-color: var(--border)">
      <div class="flex items-center justify-between mb-2">
        <span class="font-medium">{{ spec.name }}</span>
        <div class="flex gap-2">
          <Button variant="primary" :disabled="syncingId === spec.id" @click="syncNow(spec.id)">
            {{ syncingId === spec.id ? 'Syncing…' : 'Sync now' }}
          </Button>
          <Button @click="previewSync(spec.id)">Preview</Button>
          <Button @click="deleteSpec(spec.id)">Delete</Button>
        </div>
      </div>
      <p class="text-xs mb-2" style="color: var(--text-secondary)">
        <span v-if="spec.source_type === 'upload' || spec.source_type === 'push'">Stored spec ({{ spec.source_type }})</span>
        <span v-else>{{ spec.spec_url }}</span>
        <span v-if="spec.last_synced_at"> · Last synced {{ formatSynced(spec.last_synced_at) }}</span>
        <span v-else> · Not synced yet</span>
      </p>
      <IndeterminateProgressBar
        v-if="syncingId === spec.id"
        class="mb-2"
        :label="syncingLabel(spec.id)"
      />
      <p
        v-if="specSuccessById[spec.id]"
        class="text-xs mb-2"
        style="color: var(--method-get)"
      >
        {{ specSuccessById[spec.id] }}
      </p>
      <div v-if="spec.source_type === 'upload' || spec.source_type === 'push'" class="mb-2">
        <label class="text-xs block mb-1">Re-upload spec file</label>
        <input type="file" accept=".json,.yaml,.yml" class="text-xs" @change="reuploadFile(spec.id, $event)" />
      </div>
      <div class="overflow-x-auto">
        <table class="text-xs w-full">
          <thead>
            <tr style="color: var(--text-secondary)">
              <th class="text-left p-1">Environment</th>
              <th class="text-left p-1">Base URL</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="env in envStore.environments" :key="env.id">
              <td class="p-1">{{ env.name }} ({{ env.stage || 'custom' }})</td>
              <td class="p-1">
                <Input
                  :model-value="urlGrid[spec.id]?.[env.id] || ''"
                  class="text-xs"
                  @update:model-value="setUrl(spec.id, env.id, $event)"
                />
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <Button class="mt-2 text-xs" @click="saveUrls(spec.id)">Save URLs</Button>
    </div>

    <Teleport to="body">
      <div v-if="diffModal" class="fixed inset-0 z-[60] flex items-center justify-center">
        <div class="absolute inset-0 ui-overlay" @click="diffModal = null" />
        <div class="relative z-10 w-full max-w-lg rounded-lg p-5 max-h-[80vh] overflow-y-auto" style="background: var(--surface); border: 1px solid var(--border)">
          <h3 class="font-semibold mb-3">Sync preview</h3>
          <div v-if="diffModal.added.length" class="mb-3">
            <h4 class="text-sm font-medium" style="color: var(--method-get)">Added ({{ diffModal.added.length }})</h4>
            <div v-for="item in diffModal.added" :key="item.operation_id" class="text-xs font-mono">{{ item.method }} {{ item.path || item.name }}</div>
          </div>
          <div v-if="diffModal.updated.length" class="mb-3">
            <h4 class="text-sm font-medium" style="color: var(--method-patch)">Updated ({{ diffModal.updated.length }})</h4>
            <div v-for="item in diffModal.updated" :key="item.operation_id" class="text-xs font-mono">{{ item.method }} {{ item.path || item.name }}</div>
          </div>
          <div v-if="diffModal.removed.length" class="mb-3">
            <h4 class="text-sm font-medium" style="color: var(--method-delete)">Removed ({{ diffModal.removed.length }})</h4>
            <div v-for="item in diffModal.removed" :key="item.operation_id" class="text-xs font-mono">{{ item.name }}</div>
          </div>
          <p v-if="!diffModal.added.length && !diffModal.updated.length && !diffModal.removed.length" class="text-sm" style="color: var(--text-secondary)">No changes detected.</p>
          <div class="flex gap-2 justify-end mt-4">
            <Button @click="diffModal = null">Close</Button>
            <Button variant="primary" @click="applySync">Apply changes</Button>
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import type { SyncDiff } from '~/stores/apiSpecs'
import { normalizeSyncDiff } from '~/stores/apiSpecs'

const props = defineProps<{ workspaceId: string }>()

const apiSpecsStore = useApiSpecsStore()
const colStore = useCollectionsStore()
const envStore = useEnvironmentsStore()
const toast = useToast()

const newSpec = ref({ name: '', spec_url: '', collection_id: '' })
const creating = ref(false)
const syncingId = ref('')
const syncingMode = ref<'sync' | 'reupload'>('sync')
const urlGrid = ref<Record<string, Record<string, string>>>({})
const diffModal = ref<SyncDiff | null>(null)
const pendingSyncId = ref('')
const errorMessage = ref('')
const successMessage = ref('')
const specSuccessById = ref<Record<string, string>>({})

const collections = computed(() => colStore.collections.filter(c => c.workspace_id === props.workspaceId))

onMounted(async () => {
  await apiSpecsStore.list(props.workspaceId)
  await colStore.fetchCollections(props.workspaceId)
  await envStore.fetch(props.workspaceId)
})

function formatSynced(value: string) {
  try {
    return new Date(value).toLocaleString()
  } catch {
    return value
  }
}

function syncingLabel(specId: string) {
  if (syncingId.value !== specId) return ''
  return syncingMode.value === 'reupload'
    ? 'Re-uploading OpenAPI spec and syncing endpoints…'
    : 'Syncing endpoints from OpenAPI spec…'
}

function clearMessages() {
  errorMessage.value = ''
  successMessage.value = ''
}

async function refreshWorkspaceData() {
  try {
    await colStore.fetchCollections(props.workspaceId)
    await colStore.fetchAllRequests(props.workspaceId)
    await apiSpecsStore.list(props.workspaceId)
  } catch {
    // Keep success messaging when only the post-sync refresh fails.
  }
}

function showSpecSuccess(specId: string, message: string) {
  specSuccessById.value = { [specId]: message }
  successMessage.value = message
  toast.show(message, 'success', 5000)
}

function syncSummary(diff: SyncDiff) {
  const normalized = normalizeSyncDiff(diff)
  return `${normalized.added.length} added, ${normalized.updated.length} updated, ${normalized.removed.length} removed`
}

async function runSync(id: string, apply: boolean) {
  clearMessages()
  specSuccessById.value = {}
  syncingMode.value = 'sync'
  syncingId.value = id
  try {
    const diff = await apiSpecsStore.sync(id, apply)
    if (!apply) {
      diffModal.value = normalizeSyncDiff(diff)
      pendingSyncId.value = id
      return diff
    }
    const spec = apiSpecsStore.specs.find(s => s.id === id)
    const message = `Sync complete for "${spec?.name || 'spec'}": ${syncSummary(diff)}`
    showSpecSuccess(id, message)
    await refreshWorkspaceData()
    return diff
  } catch (e) {
    errorMessage.value = e instanceof Error ? e.message : 'Sync failed'
    throw e
  } finally {
    syncingId.value = ''
  }
}

async function createSpec() {
  creating.value = true
  clearMessages()
  try {
    await apiSpecsStore.create(props.workspaceId, {
      name: newSpec.value.name,
      spec_url: newSpec.value.spec_url,
      collection_id: newSpec.value.collection_id || undefined,
    })
    newSpec.value = { name: '', spec_url: '', collection_id: '' }
    const message = 'Spec added and endpoints imported.'
    successMessage.value = message
    toast.show(message, 'success', 5000)
    await refreshWorkspaceData()
  } catch (e) {
    errorMessage.value = e instanceof Error ? e.message : 'Failed to add spec'
  } finally {
    creating.value = false
  }
}

function setUrl(specId: string, envId: string, val: string) {
  if (!urlGrid.value[specId]) urlGrid.value[specId] = {}
  urlGrid.value[specId][envId] = val
}

async function saveUrls(specId: string) {
  clearMessages()
  try {
    const grid = urlGrid.value[specId] || {}
    const urls = Object.entries(grid).map(([environment_id, base_url]) => ({ environment_id, base_url }))
    await apiSpecsStore.setEnvironmentUrls(specId, urls)
    successMessage.value = 'Environment URLs saved.'
  } catch (e) {
    errorMessage.value = e instanceof Error ? e.message : 'Failed to save URLs'
  }
}

async function syncNow(id: string) {
  try {
    await runSync(id, true)
  } catch {
    // errorMessage set in runSync
  }
}

async function previewSync(id: string) {
  try {
    await runSync(id, false)
  } catch {
    // errorMessage set in runSync
  }
}

async function applySync() {
  if (!pendingSyncId.value) return
  try {
    await runSync(pendingSyncId.value, true)
    diffModal.value = null
    pendingSyncId.value = ''
  } catch {
    // keep modal open on failure
  }
}

async function deleteSpec(id: string) {
  clearMessages()
  try {
    await apiSpecsStore.delete(id)
    await refreshWorkspaceData()
    successMessage.value = 'Spec deleted.'
  } catch (e) {
    errorMessage.value = e instanceof Error ? e.message : 'Failed to delete spec'
  }
}

async function reuploadFile(specId: string, event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return
  clearMessages()
  specSuccessById.value = {}
  syncingMode.value = 'reupload'
  syncingId.value = specId
  try {
    const text = await file.text()
    const diff = await apiSpecsStore.reupload(specId, text, true)
    const spec = apiSpecsStore.specs.find(s => s.id === specId)
    const message = `Re-upload complete for "${spec?.name || 'spec'}": ${syncSummary(diff)}`
    showSpecSuccess(specId, message)
    await refreshWorkspaceData()
  } catch (e) {
    errorMessage.value = e instanceof Error ? e.message : 'Re-upload failed'
  } finally {
    syncingId.value = ''
    input.value = ''
  }
}
</script>
