<template>
  <div class="flex flex-col h-full overflow-hidden">
    <header
      class="flex items-center gap-3 px-4 h-12 shrink-0 border-b"
      style="background: var(--color-surface-1); border-color: var(--color-border)"
    >
      <NuxtLink
        :to="`/workspaces/${workspaceId}`"
        class="inline-flex items-center gap-1.5 text-xs text-muted hover:text-default transition"
      >
        <ArrowLeft :size="14" />
        <span>Workspace</span>
      </NuxtLink>
      <div class="h-4 w-px" style="background: var(--color-border)" />
      <h1 class="font-semibold text-sm">Documentation</h1>
      <div class="flex-1" />
      <button
        type="button"
        class="text-xs px-2.5 py-1 rounded border transition hover:bg-surface-2 inline-flex items-center gap-1.5"
        style="border-color: var(--color-border); color: var(--color-text-muted)"
        @click="suggestionsOpen = true"
      >
        Suggestions
        <span
          v-if="docsStore.pendingSuggestionCount"
          class="text-[10px] px-1.5 py-0.5 rounded-full font-medium"
          style="background: var(--color-accent); color: var(--color-bg)"
        >{{ docsStore.pendingSuggestionCount }}</span>
      </button>
      <div
        class="inline-flex rounded-md border overflow-hidden text-xs transition-opacity duration-150"
        style="border-color: var(--color-border)"
      >
        <button
          v-for="mode in viewModes"
          :key="mode.id"
          type="button"
          class="px-3 py-1.5 transition"
          :class="{ 'font-medium': viewMode === mode.id }"
          :style="viewMode === mode.id
            ? { background: 'var(--color-surface-2)', color: 'var(--color-text)' }
            : { color: 'var(--color-text-muted)' }"
          @click="setViewMode(mode.id)"
        >
          {{ mode.label }}
        </button>
      </div>
    </header>

    <div v-if="docsStore.error" class="px-4 py-2 text-xs text-red-400 border-b shrink-0" style="border-color: var(--color-border)">
      {{ docsStore.error }}
    </div>

    <div class="flex flex-1 min-h-0">
      <ResizablePane
        v-if="viewMode !== 'graph'"
        :initial-width="docsSidebarWidth"
        storage-key="postchi:docs-sidebar-width"
        side="right"
      >
        <aside
          class="h-full border-r flex flex-col min-h-0 overflow-hidden"
          style="border-color: var(--color-border); background: var(--color-surface-1)"
        >
          <DocsTree
            ref="treeRef"
            v-model:search="treeSearch"
            :workspace-id="workspaceId"
            :summaries="docsStore.summaries"
            :active-slug="activeSlug"
            :loading="docsStore.loading"
            @select="selectDoc"
            @create-local="onCreateLocal"
          />
        </aside>
      </ResizablePane>

      <main class="flex-1 min-w-0 flex flex-col overflow-hidden">
        <template v-if="viewMode === 'graph'">
          <DocsGraph
            :nodes="docsStore.graph?.nodes || []"
            :edges="docsStore.graph?.edges || []"
            :active-slug="activeSlug"
            :summaries="docsStore.summaries"
            @select="onGraphSelect"
          />
        </template>

        <template v-else>
          <div
            v-if="activeSummary"
            class="flex items-center gap-1 px-4 py-1.5 border-b shrink-0 text-xs overflow-x-auto"
            style="border-color: var(--color-border); color: var(--color-text-muted)"
          >
            <button
              v-for="(crumb, idx) in breadcrumbs"
              :key="crumb.path"
              type="button"
              class="inline-flex items-center gap-1 shrink-0 hover:text-default transition"
              @click="onBreadcrumb(crumb.path)"
            >
              <span v-if="idx > 0" class="opacity-50">/</span>
              <span>{{ crumb.label }}</span>
            </button>
          </div>

          <div v-if="docsStore.loadingDoc" class="flex-1 p-6 space-y-3">
            <div class="docs-skeleton h-8 w-1/3 rounded" />
            <div class="docs-skeleton h-64 w-full rounded" />
          </div>

          <template v-else-if="activeDoc || activeSummary">
            <div class="flex items-center gap-3 px-4 py-2 border-b shrink-0" style="border-color: var(--color-border)">
              <input
                v-model="editTitle"
                class="ui-input text-sm font-medium flex-1"
                :readonly="viewMode === 'preview'"
                @input="onEditInput"
              />
              <div class="flex items-center gap-1.5 text-xs shrink-0" :class="saveStatusClass">
                <span class="w-1.5 h-1.5 rounded-full" :class="saveDotClass" />
                {{ saveStatusLabel }}
              </div>
            </div>

            <div class="flex flex-1 min-h-0">
              <div
                class="flex-1 min-h-0"
                :class="viewMode === 'split' ? 'grid grid-cols-2 min-h-0' : 'flex flex-col'"
              >
                <div
                  v-show="viewMode === 'edit' || viewMode === 'split'"
                  class="min-h-0 h-full p-4 overflow-hidden flex flex-col"
                  :class="{ 'border-r': viewMode === 'split' }"
                  style="border-color: var(--color-border)"
                >
                  <MarkdownEditor
                    ref="editorRef"
                    v-model="editContent"
                    :doc-completions="docCompletions"
                    class="flex-1 min-h-0"
                    @toggle-preview="togglePreview"
                    @force-save="forceSave"
                    @view-ready="onEditorViewReady"
                  />
                </div>
                <div
                  v-show="viewMode === 'preview' || viewMode === 'split'"
                  ref="previewEl"
                  class="min-h-0 h-full ui-scroll-y p-6"
                  style="background: var(--color-bg)"
                >
                  <MarkdownViewer
                    :content="previewContent"
                    :doc-slugs="docsStore.docSlugs"
                    :doc-titles="docsStore.docTitles"
                    @navigate="selectDoc"
                  />
                </div>
              </div>
              <DocsBacklinks
                :active-slug="activeSlug"
                :summaries="docsStore.summaries"
                :graph="docsStore.graph"
                @select="selectDoc"
              />
              <DocRequestLinks
                v-if="activeDoc?.id"
                :workspace-id="workspaceId"
                :doc-id="activeDoc.id"
              />
            </div>
          </template>

          <div v-else class="flex-1 flex items-center justify-center text-sm text-muted">
            Select a document from the sidebar.
          </div>
        </template>
      </main>
    </div>

    <DocsCommandPalette
      :open="paletteOpen"
      :summaries="docsStore.summaries"
      @close="paletteOpen = false"
      @select="selectDoc"
    />

    <DocLinkSuggestionsPanel
      :open="suggestionsOpen"
      :workspace-id="workspaceId"
      @close="suggestionsOpen = false"
    />
  </div>
</template>

<script setup lang="ts">
import { ArrowLeft } from 'lucide-vue-next'
import type { EditorView } from '@codemirror/view'
import { useDocAutosave } from '~/composables/useDocAutosave'
import { useEditorPreviewScrollSync } from '~/composables/useEditorPreviewScrollSync'

type ViewMode = 'edit' | 'preview' | 'split' | 'graph'

const props = defineProps<{
  workspaceId: string
  initialSlug?: string | null
}>()

const docsStore = useDocsStore()
const route = useRoute()
const router = useRouter()

const viewModes: { id: ViewMode, label: string }[] = [
  { id: 'edit', label: 'Edit' },
  { id: 'preview', label: 'Preview' },
  { id: 'split', label: 'Split' },
  { id: 'graph', label: 'Graph' },
]

const docsSidebarWidth = 256

const viewMode = ref<ViewMode>('split')
const treeSearch = ref('')
const editTitle = ref('')
const editContent = ref('')
const savedTitle = ref('')
const savedContent = ref('')
const paletteOpen = ref(false)
const suggestionsOpen = ref(false)
const treeRef = ref<{ revealPath: (path: string) => void } | null>(null)
const editorRef = ref<InstanceType<typeof MarkdownEditor> | null>(null)
const previewEl = ref<HTMLElement | null>(null)
const workspaceReady = ref(false)
const suppressAutosave = ref(false)
/** Slug of the doc currently shown in the editor (cache alone isn't enough). */
const appliedSlug = ref<string | null>(null)
let loadGeneration = 0
let pendingLoadSlug: string | null = null

function encodePathSlug(slug: string): string {
  return encodeURIComponent(slug)
}

function decodeSlug(raw: string | string[] | undefined): string | null {
  if (!raw) return null
  const s = Array.isArray(raw) ? raw.join('/') : raw
  try {
    return decodeURIComponent(s)
  } catch {
    return s
  }
}

const activeSlug = computed(() => {
  const fromRoute = decodeSlug(route.params.slug as string | string[] | undefined)
  return fromRoute || props.initialSlug || null
})

const activeSummary = computed(() =>
  activeSlug.value ? docsStore.summaryBySlug(activeSlug.value) : null,
)

const activeDoc = computed(() =>
  activeSlug.value ? docsStore.docBySlug(activeSlug.value) : null,
)

const breadcrumbs = computed(() => {
  const path = activeSummary.value?.source_path
  if (!path) return []
  const parts = path.split('/').filter(Boolean)
  return parts.map((part, idx) => ({
    label: part,
    path: parts.slice(0, idx + 1).join('/'),
  }))
})

const previewContent = computed(() => {
  if (viewMode.value === 'preview') return savedContent.value
  return editContent.value
})

const docCompletions = computed(() =>
  docsStore.summaries.map(d => ({ label: d.title, slug: d.slug })),
)

const scrollSyncEnabled = computed(() => viewMode.value === 'split')

const editorViewForSync = shallowRef<EditorView | null>(null)

function onEditorViewReady(view: EditorView | null) {
  editorViewForSync.value = view
}

useEditorPreviewScrollSync({
  editorView: editorViewForSync,
  previewEl,
  enabled: scrollSyncEnabled,
})

const autosave = useDocAutosave({
  save: async () => {
    const slug = activeSlug.value
    if (!slug || suppressAutosave.value) return
    // Never persist editor state that doesn't belong to the active doc
    // (e.g. an empty editor after a remount) - that would destroy content.
    if (appliedSlug.value !== slug) return
    await docsStore.updateDoc(props.workspaceId, slug, {
      title: editTitle.value,
      content_md: editContent.value,
    })
    savedTitle.value = editTitle.value
    savedContent.value = editContent.value
    autosave.markSaved()
  },
})

const saveStatusLabel = computed(() => {
  switch (autosave.status.value) {
    case 'saving': return 'Saving…'
    case 'unsaved': return 'Unsaved'
    case 'error': return 'Error'
    default: return 'Saved'
  }
})

const saveStatusClass = computed(() => ({
  'text-muted': autosave.status.value === 'saved',
  'text-yellow-500': autosave.status.value === 'unsaved' || autosave.status.value === 'saving',
  'text-red-400': autosave.status.value === 'error',
}))

const saveDotClass = computed(() => ({
  'bg-green-500': autosave.status.value === 'saved',
  'bg-yellow-500 animate-pulse': autosave.status.value === 'saving' || autosave.status.value === 'unsaved',
  'bg-red-500': autosave.status.value === 'error',
}))

function onEditInput() {
  if (suppressAutosave.value) return
  autosave.markUnsaved()
  void autosave.debouncedSave()
}

function onContentInput() {
  if (suppressAutosave.value) return
  autosave.markUnsaved()
  void autosave.debouncedSave()
}

async function forceSave() {
  await autosave.forceSave()
}

function setViewMode(mode: ViewMode) {
  viewMode.value = mode
  if (mode === 'graph' && !docsStore.graph) {
    void docsStore.fetchGraph(props.workspaceId)
  }
}

function togglePreview() {
  viewMode.value = viewMode.value === 'preview' ? 'edit' : 'preview'
}

function applyDocToEditor(doc: { title: string, content_md: string }, slug: string) {
  appliedSlug.value = slug
  suppressAutosave.value = true
  autosave.cancelPending()
  editTitle.value = doc.title
  editContent.value = doc.content_md
  savedTitle.value = doc.title
  savedContent.value = doc.content_md
  autosave.markSaved()
  nextTick(() => {
    editorRef.value?.setContent(doc.content_md, slug)
    suppressAutosave.value = false
  })
}

async function loadDocContent(slug: string) {
  if (!slug || !workspaceReady.value) return
  if (pendingLoadSlug === slug) return

  pendingLoadSlug = slug
  const generation = ++loadGeneration
  try {
    const doc = await docsStore.fetchDoc(props.workspaceId, slug)
    if (generation !== loadGeneration) return
    if (!doc) return
    applyDocToEditor(doc, slug)
  } finally {
    if (pendingLoadSlug === slug) pendingLoadSlug = null
  }
}

async function openDoc(slug: string) {
  if (!slug || !workspaceReady.value) return

  const isActive = slug === activeSlug.value
  // Only skip if this doc is already displayed in the editor; a store cache
  // hit alone is not enough (e.g. after a page remount the editor is empty).
  if (isActive && appliedSlug.value === slug) return

  if (!isActive) {
    if (!(await confirmLeaveIfDirty())) return
    autosave.cancelPending()
    autosave.markSaved()
    await router.replace(`/workspaces/${props.workspaceId}/docs/${encodePathSlug(slug)}`)
  }

  await loadDocContent(slug)
}

async function confirmLeaveIfDirty(): Promise<boolean> {
  if (autosave.status.value !== 'unsaved') return true
  if (!import.meta.client) return true
  return window.confirm('You have unsaved changes. Discard them?')
}

async function selectDoc(slug: string) {
  await openDoc(slug)
}

function onGraphSelect(node: { id: string, type: string }) {
  if (node.type === 'doc') void openDoc(node.id)
}

function onBreadcrumb(path: string) {
  treeRef.value?.revealPath(path)
}

async function onCreateLocal(folderPath: string) {
  const name = window.prompt('Document name', 'Untitled')
  if (!name) return
  const sourcePath = `${folderPath}/${name.replace(/\//g, '-')}`
  try {
    const doc = await docsStore.createLocalDoc(props.workspaceId, sourcePath, name)
    selectDoc(doc.slug)
  } catch {
    window.alert('Could not create document at that path.')
  }
}

function onKeydown(e: KeyboardEvent) {
  if ((e.metaKey || e.ctrlKey) && e.key === 'p') {
    e.preventDefault()
    paletteOpen.value = true
  }
}

watch(activeSlug, (slug) => {
  if (!slug || !workspaceReady.value) return
  if (appliedSlug.value === slug) return
  void loadDocContent(slug)
})

watch(editContent, () => {
  if (suppressAutosave.value) return
  onContentInput()
})

onMounted(async () => {
  window.addEventListener('keydown', onKeydown)
  try {
    await docsStore.fetchWorkspace(props.workspaceId)
    workspaceReady.value = true
    void docsStore.fetchGraph(props.workspaceId)
    void docsStore.fetchSuggestions(props.workspaceId, 'pending').catch(() => {})
    if (activeSlug.value) {
      await openDoc(activeSlug.value)
    } else if (docsStore.summaries[0]) {
      await openDoc(docsStore.summaries[0].slug)
    }
  } catch (e) {
    // API errors are surfaced via docsStore.error; log anything else.
    console.error('[docs] failed to open initial document', e)
  }
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown)
})

onBeforeRouteLeave(async () => {
  if (autosave.status.value === 'unsaved') {
    await autosave.flushSave()
  }
})
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}
.hover\:text-default:hover {
  color: var(--color-text);
}
.docs-skeleton {
  background: linear-gradient(90deg, var(--color-surface-2) 25%, var(--color-surface-1) 50%, var(--color-surface-2) 75%);
  background-size: 200% 100%;
  animation: docs-shimmer 1.2s infinite;
}
@keyframes docs-shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}
</style>
