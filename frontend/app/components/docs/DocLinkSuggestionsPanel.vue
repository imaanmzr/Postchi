<template>
  <Teleport to="body">
    <Transition name="modal-fade">
      <div
        v-if="open"
        class="fixed inset-0 z-50 flex items-center justify-center p-4"
        @keydown.escape.prevent="onEscape"
      >
        <div class="absolute inset-0 ui-overlay backdrop-blur-sm" @click="$emit('close')" />
        <div
          class="relative z-10 w-full max-w-6xl h-[min(88vh,52rem)] rounded-lg shadow-2xl overflow-hidden flex flex-col"
          style="background: var(--color-surface-1); border: 1px solid var(--color-border)"
          role="dialog"
          aria-labelledby="link-suggestions-title"
          aria-modal="true"
        >
          <header
            class="flex items-center gap-3 px-4 py-3 border-b shrink-0"
            style="border-color: var(--color-border)"
          >
            <div class="min-w-0 flex-1">
              <h2 id="link-suggestions-title" class="font-semibold text-sm">Link suggestions</h2>
              <p class="text-[11px] text-muted mt-0.5">
                Review the document and the full API route before accepting a link.
              </p>
            </div>
            <div class="flex items-center gap-2 shrink-0">
              <Button
                class="text-xs"
                :disabled="docsStore.analyzingLinks"
                @click="analyze"
              >
                {{ docsStore.analyzingLinks ? 'Analyzing…' : 'Re-analyze' }}
              </Button>
              <Button
                v-if="highCount > 0 && highCount < pendingCount"
                class="text-xs"
                :disabled="acceptingAll"
                @click="confirmAccept('high')"
              >
                Accept all high ({{ highCount }})
              </Button>
              <Button
                variant="primary"
                class="text-xs"
                :disabled="!pendingCount || acceptingAll"
                @click="confirmAccept('all')"
              >
                {{ acceptingAll ? 'Accepting…' : `Accept all (${pendingCount})` }}
              </Button>
              <button
                type="button"
                class="ui-btn ui-btn-ghost text-lg leading-none px-2"
                aria-label="Close suggestions"
                @click="$emit('close')"
              >
                ×
              </button>
            </div>
          </header>

          <div
            v-if="analyzeMessage"
            class="px-4 py-2 text-xs text-muted border-b shrink-0"
            style="border-color: var(--color-border)"
          >
            {{ analyzeMessage }}
          </div>

          <div v-if="loading" class="flex-1 flex items-center justify-center text-sm text-muted">
            Loading suggestions…
          </div>

          <div
            v-else-if="!docsStore.suggestions.length"
            class="flex-1 flex items-center justify-center text-sm text-muted px-6 text-center"
          >
            No pending suggestions. Run re-analyze to find doc ↔ API matches.
          </div>

          <div v-else class="flex-1 min-h-0 flex flex-col md:flex-row">
            <aside
              class="w-full md:w-[22rem] lg:w-[26rem] shrink-0 border-b md:border-b-0 md:border-r flex flex-col min-h-0 max-h-[38%] md:max-h-none"
              style="border-color: var(--color-border)"
            >
              <div class="px-3 py-2 border-b shrink-0" style="border-color: var(--color-border)">
                <input
                  v-model="filterQuery"
                  type="search"
                  placeholder="Filter by doc, request, or route…"
                  class="ui-input w-full text-xs"
                />
              </div>
              <ul class="flex-1 overflow-auto ui-scroll-y">
                <li v-for="item in filteredSuggestions" :key="item.id">
                  <button
                    type="button"
                    class="w-full text-left px-3 py-2.5 border-b transition"
                    style="border-color: var(--color-border)"
                    :class="item.id === selectedId ? 'bg-surface-2' : 'hover:bg-surface-2'"
                    :data-suggestion-id="item.id"
                    @click="selectedId = item.id"
                  >
                    <div class="text-[11px] text-muted font-mono truncate" :title="docPath(item)">
                      {{ docPath(item) }}
                    </div>
                    <div class="flex items-center gap-2 mt-1 min-w-0">
                      <MethodBadge :method="item.method" class="shrink-0 scale-90" />
                      <span class="text-xs font-medium truncate">{{ item.request_name }}</span>
                    </div>
                    <div class="flex items-center gap-1.5 mt-1.5">
                      <span
                        class="text-[10px] px-1.5 py-0.5 rounded"
                        :style="confidenceStyle(item.confidence)"
                      >{{ item.confidence }}</span>
                      <span class="text-[10px] text-muted truncate">{{ reasonLabel(item.reason) }}</span>
                    </div>
                  </button>
                </li>
                <li v-if="!filteredSuggestions.length" class="px-3 py-6 text-xs text-muted text-center">
                  No suggestions match this filter.
                </li>
              </ul>
            </aside>

            <section class="flex-1 min-w-0 min-h-0 flex flex-col" style="background: var(--color-bg)">
              <template v-if="selected">
                <div
                  class="px-5 py-4 border-b shrink-0 space-y-3"
                  style="border-color: var(--color-border); background: var(--color-surface-1)"
                >
                  <div class="flex items-start gap-3">
                    <div class="min-w-0 flex-1">
                      <h3 class="text-sm font-semibold leading-snug">{{ selected.doc_title }}</h3>
                      <p class="text-[11px] text-muted font-mono mt-1 break-all">{{ docPath(selected) }}</p>
                    </div>
                    <div class="flex gap-1.5 shrink-0">
                      <Button class="text-xs px-2.5 py-1" variant="ghost" @click="reject(selected.id)">
                        Reject
                      </Button>
                      <Button class="text-xs px-2.5 py-1" variant="primary" @click="accept(selected.id)">
                        Accept
                      </Button>
                    </div>
                  </div>

                  <div
                    class="rounded-md px-3 py-2.5 space-y-2"
                    style="background: var(--color-surface-2); border: 1px solid var(--color-border)"
                  >
                    <div class="flex items-center gap-2 min-w-0">
                      <MethodBadge :method="selected.method" class="shrink-0" />
                      <span class="text-sm font-medium truncate">{{ selected.request_name }}</span>
                      <span class="text-[10px] text-muted truncate ml-auto">{{ selected.collection_name }}</span>
                    </div>
                    <p class="text-xs font-mono leading-relaxed break-all" style="color: var(--color-text)">
                      {{ selected.url }}
                    </p>
                    <div class="flex items-center gap-1.5 flex-wrap">
                      <span
                        class="text-[10px] px-1.5 py-0.5 rounded"
                        :style="confidenceStyle(selected.confidence)"
                      >{{ selected.confidence }}</span>
                      <span class="text-[10px] text-muted">{{ reasonLabel(selected.reason) }}</span>
                    </div>
                  </div>
                </div>

                <div class="flex-1 min-h-0 ui-scroll-y p-5">
                  <p class="text-[10px] uppercase tracking-wide text-muted mb-3">Document preview</p>
                  <div v-if="previewLoading" class="text-sm text-muted">Loading document…</div>
                  <MarkdownViewer
                    v-else-if="previewContent"
                    :content="previewContent"
                    :doc-slugs="docsStore.docSlugs"
                    :doc-titles="docsStore.docTitles"
                  />
                  <p v-else class="text-sm text-muted">Could not load this document.</p>
                </div>
              </template>
              <div v-else class="flex-1 flex items-center justify-center text-sm text-muted">
                Select a suggestion to preview the document and API route.
              </div>
            </section>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>

  <ConfirmDialog
    v-model:open="confirmOpen"
    :title="acceptAllMode === 'high' ? 'Accept high-confidence suggestions?' : 'Accept all suggestions?'"
    :message="acceptAllConfirmMessage"
    confirm-label="Accept"
    @confirm="runAcceptAll"
  />
</template>

<script setup lang="ts">
import type { DocLinkSuggestion } from '~/stores/docs'

const props = defineProps<{
  open: boolean
  workspaceId: string
}>()

const emit = defineEmits<{ close: [] }>()

const docsStore = useDocsStore()
const toast = useToast()
const loading = ref(false)
const acceptingAll = ref(false)
const analyzeMessage = ref('')
const selectedId = ref<string | null>(null)
const filterQuery = ref('')
const previewLoading = ref(false)
const confirmOpen = ref(false)
const acceptAllMode = ref<'high' | 'all'>('all')

const pendingCount = computed(() => docsStore.suggestions.length)

const highCount = computed(() =>
  docsStore.suggestions.filter(s => isHighConfidence(s.confidence)).length,
)

const selected = computed(() =>
  docsStore.suggestions.find(s => s.id === selectedId.value) ?? null,
)

const filteredSuggestions = computed(() => {
  const q = filterQuery.value.trim().toLowerCase()
  if (!q) return docsStore.suggestions
  return docsStore.suggestions.filter((item) => {
    const haystack = [
      item.doc_title,
      item.doc_slug,
      item.request_name,
      item.url,
      item.method,
      item.collection_name,
      item.reason,
      item.confidence,
      docPath(item),
    ].join(' ').toLowerCase()
    return haystack.includes(q)
  })
})

const previewContent = computed(() => {
  if (!selected.value) return ''
  return docsStore.docBySlug(selected.value.doc_slug)?.content_md ?? ''
})

const acceptAllConfirmMessage = computed(() => {
  if (acceptAllMode.value === 'high') {
    return `This will create ${highCount.value} manual doc ↔ request link${highCount.value === 1 ? '' : 's'} for exact and high-confidence matches.`
  }
  return `This will create ${pendingCount.value} manual doc ↔ request link${pendingCount.value === 1 ? '' : 's'} from all pending suggestions.`
})

function isHighConfidence(confidence: string) {
  return confidence === 'high' || confidence === 'exact'
}

function docPath(item: DocLinkSuggestion) {
  const summary = docsStore.summaries.find(d => d.id === item.doc_id || d.slug === item.doc_slug)
  return summary?.source_path || item.doc_slug
}

function confidenceStyle(confidence: string) {
  if (isHighConfidence(confidence)) {
    return { background: 'var(--color-surface-2)', color: 'var(--color-accent)' }
  }
  return { background: 'var(--color-surface-2)', color: 'var(--color-text-muted)' }
}

function reasonLabel(reason: string) {
  const labels: Record<string, string> = {
    content_method_path: 'HTTP method/path in doc body',
    path_alignment: 'Doc path matches request path',
    title_similarity: 'Similar title/name',
    folder_alignment: 'Folder mirrors collection',
    exact_name: 'Exact request name match',
    ambiguous_exact_name: 'Exact name, more than one match',
    path_template: 'Path template match',
    ambiguous_path_template: 'Path template, more than one match',
    frontmatter_request: 'Frontmatter request name',
  }
  return labels[reason] ?? reason.replace(/_/g, ' ')
}

function selectFirstVisible() {
  selectedId.value = filteredSuggestions.value[0]?.id ?? docsStore.suggestions[0]?.id ?? null
}

watch(() => [props.open, props.workspaceId] as const, async ([open, wsId]) => {
  if (!open || !wsId) return
  loading.value = true
  filterQuery.value = ''
  analyzeMessage.value = ''
  try {
    await docsStore.fetchSuggestions(wsId, 'pending')
    selectFirstVisible()
  } finally {
    loading.value = false
  }
}, { immediate: true })

watch(filteredSuggestions, (items) => {
  if (!items.length) {
    selectedId.value = null
    return
  }
  if (!items.some(item => item.id === selectedId.value)) {
    selectedId.value = items[0]!.id
  }
})

watch(selectedId, async (id) => {
  if (!id) return
  await nextTick()
  document.querySelector(`[data-suggestion-id="${id}"]`)?.scrollIntoView({ block: 'nearest' })
})

watch(selected, async (item) => {
  if (!item || !props.workspaceId) return
  if (docsStore.docBySlug(item.doc_slug)) {
    previewLoading.value = false
    return
  }
  previewLoading.value = true
  const slug = item.doc_slug
  try {
    await docsStore.fetchDoc(props.workspaceId, slug)
  } finally {
    if (selected.value?.doc_slug === slug) previewLoading.value = false
  }
})

async function analyze() {
  analyzeMessage.value = ''
  try {
    const result = await docsStore.analyzeLinks(props.workspaceId)
    analyzeMessage.value = `Found ${result.pending_total ?? 0} pending suggestion(s).`
    selectFirstVisible()
    await docsStore.fetchGraph(props.workspaceId)
  } catch (e) {
    analyzeMessage.value = e instanceof Error ? e.message : 'Analyze failed'
  }
}

function selectAfterRemoval(removedIndex: number) {
  const next = docsStore.suggestions[removedIndex] ?? docsStore.suggestions[removedIndex - 1]
  selectedId.value = next?.id ?? null
}

async function accept(id: string) {
  const idx = docsStore.suggestions.findIndex(s => s.id === id)
  await docsStore.acceptSuggestion(props.workspaceId, id)
  selectAfterRemoval(idx)
  await docsStore.fetchGraph(props.workspaceId)
  toast.show('Link accepted.')
}

async function reject(id: string) {
  const idx = docsStore.suggestions.findIndex(s => s.id === id)
  await docsStore.rejectSuggestion(props.workspaceId, id)
  selectAfterRemoval(idx)
  toast.show('Suggestion dismissed.')
}

function onEscape() {
  if (confirmOpen.value) return
  emit('close')
}

function moveSelection(delta: number) {
  const items = filteredSuggestions.value
  if (!items.length) return
  const idx = items.findIndex(item => item.id === selectedId.value)
  const nextIdx = Math.min(items.length - 1, Math.max(0, (idx < 0 ? 0 : idx) + delta))
  selectedId.value = items[nextIdx]!.id
}

function onWindowKeydown(e: KeyboardEvent) {
  if (!props.open) return
  if (e.key === 'Escape') {
    onEscape()
    return
  }
  if (confirmOpen.value) return
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    moveSelection(1)
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    moveSelection(-1)
  }
}

watch(() => props.open, (open) => {
  if (open) window.addEventListener('keydown', onWindowKeydown)
  else window.removeEventListener('keydown', onWindowKeydown)
})

onUnmounted(() => {
  window.removeEventListener('keydown', onWindowKeydown)
})

function confirmAccept(mode: 'high' | 'all') {
  acceptAllMode.value = mode
  confirmOpen.value = true
}

async function runAcceptAll() {
  acceptingAll.value = true
  try {
    const result = await docsStore.acceptAllSuggestions(props.workspaceId, acceptAllMode.value)
    selectFirstVisible()
    await docsStore.fetchGraph(props.workspaceId)
    toast.show(`Accepted ${result.accepted ?? 0} link(s).`)
  } finally {
    acceptingAll.value = false
  }
}
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}
.modal-fade-enter-active,
.modal-fade-leave-active {
  transition: opacity var(--duration-normal) var(--ease-out);
}
.modal-fade-enter-active .relative,
.modal-fade-leave-active .relative {
  transition: transform var(--duration-normal) var(--ease-out), opacity var(--duration-normal) var(--ease-out);
}
.modal-fade-enter-from,
.modal-fade-leave-to {
  opacity: 0;
}
.modal-fade-enter-from .relative,
.modal-fade-leave-to .relative {
  transform: scale(0.96);
  opacity: 0;
}
</style>
