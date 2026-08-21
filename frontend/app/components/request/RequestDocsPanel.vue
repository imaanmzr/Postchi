<template>
  <div class="grid grid-cols-1 lg:grid-cols-2 gap-6">
    <div v-if="editable" class="space-y-4">
      <div>
        <label class="text-sm flex items-center gap-1.5 mb-2 font-medium">
          <FileText :size="14" class="opacity-70 shrink-0" aria-hidden="true" />
          Documentation notes
        </label>
        <textarea
          v-model="description"
          class="ui-input w-full h-48 font-mono text-sm"
          placeholder="Describe this endpoint for your team (Markdown supported)"
        />
        <p class="text-xs text-muted mt-1">Manual edits mark docs as team-owned and won't be overwritten by OpenAPI sync.</p>
      </div>
      <div class="flex justify-end">
        <Button
          class="inline-flex items-center gap-1.5"
          :variant="isDescriptionDirty ? 'primary' : 'ghost'"
          :disabled="!isDescriptionDirty || saving"
          @click="saveDocs"
        >
          <Check v-if="!isDescriptionDirty" :size="14" aria-hidden="true" />
          <Save v-else :size="14" aria-hidden="true" />
          {{ isDescriptionDirty ? 'Save documentation' : 'Saved' }}
        </Button>
      </div>
    </div>

    <div class="space-y-4" :class="{ 'lg:col-span-2': !editable }">
      <div v-if="bundleLoading" class="text-xs text-muted">Loading documentation…</div>

      <div v-if="sourceSpecId" class="text-xs px-2 py-1 rounded inline-block" style="background: var(--color-surface-2); color: var(--color-text-muted)">
        Synced from OpenAPI
      </div>
      <div v-if="apiDoc?.summary" class="text-sm">
        <span class="font-medium">Summary:</span> {{ apiDoc.summary }}
      </div>
      <div v-if="apiDoc?.tags?.length" class="flex flex-wrap gap-1">
        <span
          v-for="tag in apiDoc.tags"
          :key="tag"
          class="text-xs px-2 py-0.5 rounded"
          style="background: var(--color-surface-2)"
        >{{ tag }}</span>
      </div>
      <div v-if="apiDoc?.deprecated" class="text-xs text-amber-600">Deprecated</div>

      <div v-if="apiDoc?.parameters?.length">
        <h4 class="text-sm font-medium mb-2">Parameters</h4>
        <table class="w-full text-xs border-collapse">
          <thead>
            <tr style="border-bottom: 1px solid var(--color-border)">
              <th class="text-left py-1 pr-2">Name</th>
              <th class="text-left py-1 pr-2">In</th>
              <th class="text-left py-1 pr-2">Required</th>
              <th class="text-left py-1">Description</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="p in apiDoc.parameters" :key="p.name + p.in" style="border-bottom: 1px solid var(--color-border)">
              <td class="py-1 pr-2 font-mono">{{ p.name }}</td>
              <td class="py-1 pr-2">{{ p.in }}</td>
              <td class="py-1 pr-2">{{ p.required ? 'Yes' : 'No' }}</td>
              <td class="py-1">{{ p.description || '-' }}</td>
            </tr>
          </tbody>
        </table>
      </div>

      <div v-if="apiDoc?.requestBody">
        <h4 class="text-sm font-medium mb-2">Request body</h4>
        <p v-if="apiDoc.requestBody.description" class="text-xs text-muted mb-2">{{ apiDoc.requestBody.description }}</p>
        <ApiDocContentBlock :content="apiDoc.requestBody.content" />
      </div>

      <div v-if="responseCodes.length">
        <h4 class="text-sm font-medium mb-2">Responses</h4>
        <div class="space-y-2">
          <details v-for="code in responseCodes" :key="code" class="rounded border" style="border-color: var(--color-border)">
            <summary class="px-3 py-2 cursor-pointer text-sm font-mono flex items-center gap-2">
              <span :class="statusClass(code)">{{ code }}</span>
              <span class="font-normal text-muted">{{ apiDoc.responses[code]?.description || '' }}</span>
            </summary>
            <div class="px-3 pb-3">
              <ApiDocContentBlock :content="apiDoc.responses[code]?.content" />
            </div>
          </details>
        </div>
      </div>

      <div class="border-t pt-4" style="border-color: var(--color-border)">
        <div class="flex items-center gap-2 mb-3">
          <h4 class="text-sm font-medium">Team notes</h4>
          <Button
            v-if="editable && workspaceId"
            class="text-xs"
            @click="openDocPicker"
          >
            Link doc page
          </Button>
        </div>

        <p v-if="linkError" class="text-xs text-red-400 mb-2">{{ linkError }}</p>

        <div v-if="linkedDocs.length" class="space-y-2">
          <div
            v-for="doc in linkedDocs"
            :key="doc.id"
            class="rounded border px-3 py-2.5 transition hover:bg-surface-2"
            style="border-color: var(--color-border)"
          >
            <div class="flex items-start gap-2">
              <button
                type="button"
                class="min-w-0 flex-1 text-left group"
                @click="openPreview(doc)"
              >
                <div class="flex items-center gap-2 flex-wrap">
                  <span class="text-sm font-medium group-hover:text-[var(--color-accent)] transition">
                    {{ doc.title }}
                  </span>
                  <span
                    v-if="doc.link_sources.includes('frontmatter')"
                    class="text-[10px] px-1.5 py-0.5 rounded"
                    style="background: var(--color-surface-2); color: var(--color-text-muted)"
                  >auto</span>
                  <span
                    v-if="doc.link_sources.includes('manual')"
                    class="text-[10px] px-1.5 py-0.5 rounded"
                    style="background: var(--color-surface-2); color: var(--color-accent)"
                  >manual</span>
                </div>
                <p v-if="docExcerpt(doc)" class="text-xs text-muted mt-1 line-clamp-2 leading-relaxed">
                  {{ docExcerpt(doc) }}
                </p>
              </button>
              <div class="flex items-center gap-1 shrink-0">
                <Button class="text-xs px-2 py-1" @click="openPreview(doc)">
                  Preview
                </Button>
                <NuxtLink
                  v-if="workspaceId"
                  :to="docWorkspacePath(doc.slug)"
                  class="ui-btn ui-btn-ghost text-xs px-2 py-1"
                >
                  Open doc
                </NuxtLink>
                <Button
                  v-if="editable && doc.link_id"
                  class="text-xs px-2 py-1"
                  :disabled="unlinkingId === doc.link_id"
                  @click="unlinkDoc(doc)"
                >
                  {{ unlinkingId === doc.link_id ? 'Unlinking…' : 'Unlink' }}
                </Button>
              </div>
            </div>
          </div>
        </div>
        <p v-else class="text-xs text-muted">No linked doc pages yet.</p>
      </div>

      <p v-if="!hasApiDoc && !description && !linkedDocs.length" class="text-sm text-muted">
        No API documentation yet. Connect an OpenAPI spec or add notes manually.
      </p>
    </div>

    <EntitySearchPicker
      :open="docPickerOpen"
      :items="pickerDocs"
      :get-key="(d: DocSummary) => d.id"
      :get-title="(d: DocSummary) => d.title"
      :get-subtitle="(d: DocSummary) => d.source_path || d.slug"
      :search-keys="['title', 'source_path', 'slug']"
      placeholder="Search doc pages…"
      @close="docPickerOpen = false"
      @select="onDocSelected"
    />

    <LinkedDocPreviewDialog
      :open="previewDoc !== null"
      :doc="previewDoc"
      :workspace-id="workspaceId"
      @close="previewDoc = null"
    />
  </div>
</template>

<script setup lang="ts">
import type { RequestItem } from '~/stores/collections'
import type { DocsBundle, LinkedWorkspaceDoc } from '~/stores/docs'
import type { DocSummary } from '~/utils/docsTree'
import { DOC_ALREADY_LINKED_MESSAGE, isSameDoc } from '~/utils/docLinks'
import { Check, Save } from 'lucide-vue-next'

const props = withDefaults(defineProps<{
  request: RequestItem
  workspaceId?: string
  editable?: boolean
}>(), { editable: true })

const emit = defineEmits<{ save: [req: RequestItem]; 'docs-changed': [] }>()

const api = useApi()
const docsStore = useDocsStore()
const toast = useToast()
const description = ref('')
const baselineDescription = ref('')
const bundle = ref<DocsBundle | null>(null)
const bundleLoading = ref(false)
const docPickerOpen = ref(false)
const previewDoc = ref<LinkedWorkspaceDoc | null>(null)
const unlinkingId = ref<string | null>(null)
const linkError = ref<string | null>(null)
const saving = ref(false)

const isDescriptionDirty = computed(
  () => description.value !== baselineDescription.value,
)

watch(() => props.request.id, () => {
  resetDescriptionFromRequest()
  fetchBundle()
}, { immediate: true })

watch(() => props.request.description, (val) => {
  if (!isDescriptionDirty.value) {
    description.value = val || ''
    baselineDescription.value = val || ''
  }
})

const apiDoc = computed(() => {
  const raw = bundle.value?.api_doc ?? props.request.api_doc
  if (!raw || typeof raw !== 'object') return null
  return raw as Record<string, any>
})

const sourceSpecId = computed(() => props.request.source_spec_id)
const hasApiDoc = computed(() => apiDoc.value && Object.keys(apiDoc.value).length > 0)
const linkedDocs = computed(() => bundle.value?.linked_workspace_docs ?? [])

const responseCodes = computed(() => {
  const responses = apiDoc.value?.responses
  if (!responses) return []
  return Object.keys(responses).sort((a, b) => {
    const na = parseInt(a) || 999
    const nb = parseInt(b) || 999
    return na - nb
  })
})

const pickerDocs = computed(() =>
  docsStore.summaries.filter(doc => !isDocLinked(doc)),
)

function linkedDocIdentity(doc: LinkedWorkspaceDoc | DocSummary) {
  const source_path = 'source_path' in doc
    ? doc.source_path
    : docsStore.summaryBySlug(doc.slug)?.source_path ?? null
  return {
    id: doc.id,
    slug: doc.slug,
    source_path,
  }
}

function isDocLinked(doc: DocSummary): boolean {
  const candidate = linkedDocIdentity(doc)
  return linkedDocs.value.some(linked => isSameDoc(candidate, linkedDocIdentity(linked)))
}

function resetDescriptionFromRequest() {
  const text = props.request.description || ''
  description.value = text
  baselineDescription.value = text
}

function docWorkspacePath(slug: string) {
  return `/workspaces/${props.workspaceId}/docs/${encodeURIComponent(slug)}`
}

function openDocPicker() {
  linkError.value = null
  docPickerOpen.value = true
}

function docExcerpt(doc: LinkedWorkspaceDoc): string {
  const summary = docsStore.summaryBySlug(doc.slug)
  const path = summary?.source_path || doc.slug.replace(/-/g, '/')
  const plain = doc.content_md
    .replace(/^---[\s\S]*?---\n?/m, '')
    .replace(/^#+\s+/gm, '')
    .replace(/\[\[([^\]]+)\]\]/g, '$1')
    .replace(/`{1,3}[^`]*`{1,3}/g, '')
    .replace(/[#*_>]/g, '')
    .replace(/\s+/g, ' ')
    .trim()
  if (plain) {
    return plain.length > 120 ? `${plain.slice(0, 120)}…` : plain
  }
  return path
}

function openPreview(doc: LinkedWorkspaceDoc) {
  previewDoc.value = doc
}

async function fetchBundle() {
  if (!props.request.id) return
  if (!props.editable) {
    bundle.value = null
    return
  }
  bundleLoading.value = true
  try {
    bundle.value = await api.get<DocsBundle>(`/api/requests/${props.request.id}/docs-bundle`)
    if (!isDescriptionDirty.value) {
      const text = bundle.value.description || ''
      description.value = text
      baselineDescription.value = text
    }
  } catch {
    bundle.value = null
  } finally {
    bundleLoading.value = false
  }
}

async function ensureDocSummaries() {
  if (!props.workspaceId) return
  if (!docsStore.summaries.length) {
    await docsStore.fetchWorkspace(props.workspaceId)
  }
}

watch(docPickerOpen, (open) => {
  if (open) ensureDocSummaries()
})

async function onDocSelected(doc: DocSummary) {
  if (!props.workspaceId) return
  linkError.value = null
  if (isDocLinked(doc)) {
    linkError.value = DOC_ALREADY_LINKED_MESSAGE
    return
  }
  try {
    await docsStore.createDocLink(props.workspaceId, doc.id, { request_id: props.request.id })
    docPickerOpen.value = false
    await fetchBundle()
    emit('docs-changed')
    toast.show(`Linked "${doc.title}" successfully.`)
  } catch (e: unknown) {
    linkError.value = e instanceof Error ? e.message : 'Failed to link document'
  }
}

async function unlinkDoc(doc: LinkedWorkspaceDoc) {
  if (!props.workspaceId || !doc.link_id || unlinkingId.value) return
  unlinkingId.value = doc.link_id
  try {
    await docsStore.deleteDocLink(props.workspaceId, doc.id, doc.link_id)
    if (previewDoc.value?.id === doc.id) previewDoc.value = null
    await fetchBundle()
    emit('docs-changed')
  } finally {
    unlinkingId.value = null
  }
}

function statusClass(code: string) {
  const n = parseInt(code)
  if (n >= 200 && n < 300) return 'text-green-600'
  if (n >= 400 && n < 500) return 'text-amber-600'
  if (n >= 500) return 'text-red-600'
  return ''
}

async function saveDocs() {
  if (!isDescriptionDirty.value || saving.value) return
  saving.value = true
  try {
    emit('save', {
      ...props.request,
      description: description.value,
      docs_overridden: true,
    })
    baselineDescription.value = description.value
    emit('docs-changed')
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}
.hover\:bg-surface-2:hover {
  background: var(--color-surface-2);
}
</style>
