<template>
  <Teleport to="body">
    <div class="fixed inset-0 z-50 flex items-center justify-center">
      <div class="absolute inset-0 ui-overlay" @click="$emit('close')" />
      <div class="relative z-10 w-full max-w-lg rounded-lg p-5" style="background: var(--surface); border: 1px solid var(--border)">
        <h2 class="text-lg font-semibold mb-4">Import / Export</h2>

        <div class="space-y-4">
          <div>
            <label class="text-sm block mb-1">Format</label>
            <Select v-model="format">
              <option value="auto">Auto-detect</option>
              <option value="bruno">Bruno (.bru / .zip)</option>
              <option value="opencollection">Bruno OpenCollection (.yml / .yaml)</option>
              <option value="postman">Postman v2.1 (.json)</option>
              <option value="openapi">OpenAPI 3.0/3.1 (.json / .yaml)</option>
            </Select>
            <p v-if="detectedFormat" class="text-xs mt-1" style="color: var(--text-muted)">
              Detected: {{ formatLabel(detectedFormat) }}
            </p>
          </div>

          <div>
            <label class="text-sm block mb-1">Import file</label>
            <input type="file" class="text-sm ui-input w-full" @change="onFile" />
          </div>

          <div v-if="error" class="text-sm" style="color: var(--method-delete)">{{ error }}</div>
          <div v-if="success" class="text-sm" style="color: var(--method-get)">{{ success }}</div>

          <div v-if="importVarsStep">
            <VariableMappingStep
              :workspace-id="workspaceId"
              :placeholder-names="importPlaceholders"
              @done="onVarsMapped"
              @cancel="importVarsStep = false"
            />
          </div>

          <div v-else class="flex gap-2">
            <Button variant="primary" :disabled="!file || importing" @click="doImport">
              {{ importing ? 'Importing…' : 'Import' }}
            </Button>
            <Button :disabled="!exportColId" @click="doExport">Export collection</Button>
          </div>

          <div>
            <label class="text-sm block mb-1">Export collection</label>
            <Select v-model="exportColId">
              <option value="">Select collection</option>
              <option v-for="c in colStore.collections" :key="c.id" :value="c.id">{{ c.name }}</option>
            </Select>
          </div>
        </div>

        <div class="mt-4 flex justify-end">
          <Button @click="$emit('close')">Close</Button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { apiUrl } from '~/utils/apiBase'
import { extractPlaceholders } from '~/utils/placeholders'

interface ImportResult {
  collection_id?: string
  collections: number
  requests: number
  environments: number
}

type ImportFormat = 'auto' | 'bruno' | 'opencollection' | 'postman' | 'openapi'

const props = defineProps<{ workspaceId: string }>()
defineEmits<{ close: [] }>()

const colStore = useCollectionsStore()
const envStore = useEnvironmentsStore()
const config = useRuntimeConfig()
const format = ref<ImportFormat>('auto')
const file = ref<File | null>(null)
const fileText = ref('')
const detectedFormat = ref<ImportFormat | ''>('')
const exportColId = ref('')
const importing = ref(false)
const error = ref('')
const success = ref('')
const importVarsStep = ref(false)
const importPlaceholders = ref<string[]>([])

const formatLabels: Record<Exclude<ImportFormat, 'auto'>, string> = {
  bruno: 'Bruno (.bru / .zip)',
  opencollection: 'Bruno OpenCollection',
  postman: 'Postman v2.1',
  openapi: 'OpenAPI 3.0/3.1',
}

onMounted(async () => {
  await colStore.fetchCollections(props.workspaceId)
  await envStore.fetch(props.workspaceId)
})

function formatLabel(f: ImportFormat | '') {
  if (!f || f === 'auto') return ''
  return formatLabels[f]
}

function detectFormat(name: string, text: string): Exclude<ImportFormat, 'auto'> {
  const lower = name.toLowerCase()
  if (lower.endsWith('.zip') || lower.endsWith('.bru')) return 'bruno'
  const trimmed = text.trim()
  if (trimmed.startsWith('{')) {
    try {
      const json = JSON.parse(trimmed)
      if (json.opencollection) return 'opencollection'
      const schema = json.info?.schema || ''
      if (typeof schema === 'string' && schema.includes('postman.com/json/collection')) return 'postman'
      if (json.openapi || json.swagger) return 'openapi'
    } catch { /* fall through */ }
  }
  if (/^opencollection\s*:/m.test(trimmed)) return 'opencollection'
  if (/^openapi\s*:/m.test(trimmed) || /^swagger\s*:/m.test(trimmed)) return 'openapi'
  if (lower.endsWith('.yml') || lower.endsWith('.yaml')) return 'opencollection'
  return 'postman'
}

function resolvedFormat(): Exclude<ImportFormat, 'auto'> {
  if (format.value !== 'auto') return format.value
  return detectedFormat.value || 'postman'
}

async function onFile(e: Event) {
  const picked = (e.target as HTMLInputElement).files?.[0] || null
  file.value = picked
  error.value = ''
  success.value = ''
  detectedFormat.value = ''
  fileText.value = ''
  if (!picked) return
  if (picked.name.toLowerCase().endsWith('.zip') || picked.name.toLowerCase().endsWith('.bru')) {
    detectedFormat.value = 'bruno'
    return
  }
  fileText.value = await picked.text()
  detectedFormat.value = detectFormat(picked.name, fileText.value)
}

function formatResult(r: ImportResult) {
  const parts = []
  if (r.collections) parts.push(`${r.collections} folder(s)`)
  if (r.requests) parts.push(`${r.requests} request(s)`)
  if (r.environments) parts.push(`${r.environments} environment(s)`)
  return parts.length ? `Imported ${parts.join(', ')}` : 'Import produced no data'
}

async function importText(path: string, body: string, contentType: string): Promise<ImportResult> {
  const res = await fetch(apiUrl(config.public.apiUrl as string, path), {
    method: 'POST',
    headers: {
      'Content-Type': contentType,
      Authorization: `Bearer ${useAuthStore().accessToken}`,
    },
    body,
  })
  if (!res.ok) {
    const err = await res.json().catch(() => ({}))
    throw new Error(err.error || 'Import failed')
  }
  return res.json()
}

async function doImport() {
  if (!file.value) return
  const text = fileText.value || (file.value.name.toLowerCase().endsWith('.zip') ? '' : await file.value.text())
  const placeholders = extractPlaceholders(text)
  if (placeholders.length && envStore.activeId) {
    const { missing } = await envStore.resolveVariables(envStore.activeId, placeholders)
    if (missing.length) {
      importPlaceholders.value = placeholders
      importVarsStep.value = true
      return
    }
  }
  await runImport()
}

async function onVarsMapped() {
  importVarsStep.value = false
  await runImport()
}

async function runImport() {
  if (!file.value) return
  importing.value = true
  error.value = ''
  success.value = ''
  try {
    const api = useApi()
    const kind = resolvedFormat()
    let result: ImportResult
    if (kind === 'bruno') {
      const fd = new FormData()
      fd.append('file', file.value)
      result = await api.upload(`/api/import/bruno?workspace_id=${props.workspaceId}`, fd)
    } else {
      const text = fileText.value || await file.value.text()
      const qs = `?workspace_id=${props.workspaceId}`
      if (kind === 'postman') {
        result = await importText(`/api/import/postman${qs}`, text, 'application/json')
      } else if (kind === 'opencollection') {
        result = await importText(`/api/import/opencollection${qs}`, text, 'application/octet-stream')
      } else {
        result = await importText(`/api/import/openapi${qs}`, text, 'application/octet-stream')
      }
    }
    const total = (result.collections || 0) + (result.requests || 0) + (result.environments || 0)
    if (total === 0) throw new Error('Import produced no collections or requests')
    success.value = formatResult(result)
    await colStore.fetchCollections(props.workspaceId)
    await colStore.fetchAllRequests(props.workspaceId)
  } catch (e: any) {
    error.value = e.message || 'Import failed'
  } finally {
    importing.value = false
  }
}

async function doExport() {
  if (!exportColId.value) return
  const api = useApi()
  const ext = format.value === 'bruno' ? 'zip' : 'json'
  const path = format.value === 'bruno'
    ? `/api/export/bruno?collection_id=${exportColId.value}`
    : `/api/export/postman?collection_id=${exportColId.value}`
  if (format.value === 'bruno') {
    await api.download(path, `collection.${ext}`)
  } else {
    const data = await api.get<Record<string, unknown>>(path)
    const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' })
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = 'collection.json'
    a.click()
    URL.revokeObjectURL(url)
  }
}
</script>
