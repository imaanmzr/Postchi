<template>
  <div class="flex flex-col h-full overflow-hidden">
    <div class="flex flex-1 min-h-0">
      <ResizablePane :initial-width="280" side="right">
        <aside
          class="h-full border-r flex flex-col min-h-0 overflow-hidden"
          style="border-color: var(--color-border); background: var(--color-surface-1)"
        >
          <div class="p-3 border-b shrink-0 space-y-2" style="border-color: var(--color-border)">
            <Button
              variant="primary"
              class="w-full text-xs inline-flex items-center justify-center gap-1.5"
              @click="createDiagram"
            >
              <PenLine :size="14" class="shrink-0" />
              New user story
            </Button>
            <Input v-model="sidebarSearch" placeholder="Search stories…" class="text-xs" />
          </div>
          <div v-if="diagramsStore.loading" class="p-4 text-xs text-muted">Loading…</div>
          <div v-else-if="!filteredSummaries.length && !sidebarSearch" class="p-4 text-center">
            <p class="text-sm font-medium mb-1">Draw your first user story</p>
            <p class="text-xs text-muted mb-3">Sketch flows here, then link them from your docs.</p>
            <Button variant="primary" class="text-xs inline-flex items-center gap-1.5" @click="createDiagram">
              <PenLine :size="14" class="shrink-0" />
              Create story
            </Button>
          </div>
          <p v-else-if="!filteredSummaries.length" class="p-4 text-xs text-muted">No matching stories.</p>
          <ul v-else class="flex-1 overflow-y-auto p-2 space-y-1">
            <li v-for="item in filteredSummaries" :key="item.slug">
              <button
                type="button"
                class="w-full text-left px-3 py-2 rounded text-sm transition"
                :style="item.slug === activeSlug
                  ? { background: 'var(--color-surface-2)', fontWeight: 600 }
                  : { color: 'var(--color-text-muted)' }"
                @click="selectDiagram(item.slug)"
              >
                <div class="truncate">{{ item.title }}</div>
                <div class="text-[10px] font-mono opacity-60 truncate">{{ item.slug }}</div>
              </button>
            </li>
          </ul>
        </aside>
      </ResizablePane>

      <main class="flex-1 min-w-0 flex flex-col overflow-hidden">
        <div
          v-if="activeSlug"
          class="flex items-center gap-2 px-4 py-2 border-b shrink-0 text-xs"
          style="border-color: var(--color-border)"
        >
          <Input v-model="titleDraft" class="max-w-xs text-sm" placeholder="Story title" @blur="saveTitle" />
          <Button
            class="text-xs shrink-0 inline-flex items-center gap-1.5"
            title="Copy link for docs"
            @click="copyDocLink"
          >
            <Link2 :size="14" class="shrink-0" />
            Copy doc link
          </Button>
          <div class="flex-1" />
          <span
            class="text-[11px] uppercase tracking-wide font-medium px-2 py-1 rounded"
            :style="saveStatusBadgeStyle"
          >
            {{ saveStatusLabel }}
          </span>
          <Button
            variant="primary"
            class="text-xs shrink-0 inline-flex items-center gap-1.5 min-w-[5.5rem] justify-center"
            :disabled="autosave.status.value === 'saved' || autosave.status.value === 'saving'"
            @click="saveNow"
          >
            <Loader2 v-if="autosave.status.value === 'saving'" :size="14" class="shrink-0 animate-spin" />
            <Save v-else :size="14" class="shrink-0" />
            Save now
          </Button>
          <button
            type="button"
            class="delete-btn ui-btn text-xs shrink-0 inline-flex items-center gap-1.5 border transition"
            style="
              border-color: color-mix(in srgb, var(--method-delete) 35%, var(--color-border));
              color: var(--method-delete);
              background: color-mix(in srgb, var(--method-delete) 8%, transparent);
            "
            @click="deleteActive"
          >
            <Trash2 :size="14" class="shrink-0" />
            Delete
          </button>
        </div>

        <div v-if="!activeSlug" class="flex-1 flex items-center justify-center text-muted text-sm px-6 text-center">
          Select a story from the sidebar or create a new one to start drawing.
        </div>
        <div v-else class="flex-1 min-h-0 flex min-w-0">
          <div class="flex-1 min-h-0 relative min-w-0">
            <div
              v-if="loadingDiagram"
              class="absolute inset-0 z-10 flex items-center justify-center text-sm text-muted"
              style="background: color-mix(in srgb, var(--color-bg) 85%, transparent)"
            >
              Loading story…
            </div>
            <ExcalidrawEditor
              v-if="editorMounted"
              ref="editorRef"
              :diagram-slug="activeSlug"
              :session-key="editorSessionKey"
              :initial-data="loadedScene"
              @change="onDiagramChange"
            />
          </div>
          <ResizablePane :initial-width="260" side="left">
            <LinkedRequestsPanel
              class="border-l h-full"
              style="border-color: var(--color-border)"
              :requests="linkedRequests"
              @link="showRequestPicker = true"
              @unlink="unlinkRequest"
            />
          </ResizablePane>
        </div>
      </main>
    </div>

    <CrossWorkspaceRequestPicker
      :open="showRequestPicker"
      :exclude-ids="linkedRequestIds"
      @select="linkRequest"
      @close="showRequestPicker = false"
    />
  </div>
</template>

<script setup lang="ts">
import { Link2, Loader2, PenLine, Save, Trash2 } from 'lucide-vue-next'
import { diagramWikilink, type ExcalidrawScene } from '~/utils/diagramContent'
import { copyToClipboard } from '~/utils/copyToClipboard'
import type { LinkedRequest } from '~/utils/linkableRequests'

const props = defineProps<{
  workspaceId: string
  initialSlug?: string
}>()

const route = useRoute()
const router = useRouter()
const toast = useToast()
const diagramsStore = useDiagramsStore()

const activeSlug = ref('')
const titleDraft = ref('')
const loadedScene = ref<Record<string, unknown> | null>(null)
const loadingDiagram = ref(false)
const editorMounted = ref(false)
const editorSessionKey = ref(0)
const pendingScene = ref<ExcalidrawScene | null>(null)
const sidebarSearch = ref('')
const showRequestPicker = ref(false)
const editorRef = ref<{ getScene: () => ExcalidrawScene | null } | null>(null)
const switching = ref(false)
const sceneCache = new Map<string, Record<string, unknown> | null>()

const autosave = useDocAutosave({
  debounceMs: 2500,
  save: async () => {
    if (!activeSlug.value) return
    const scene = pendingScene.value || editorRef.value?.getScene()
    if (!scene) return
    await diagramsStore.updateDiagram(props.workspaceId, activeSlug.value, { content: scene })
    sceneCache.set(activeSlug.value, scene)
    pendingScene.value = scene
  },
})

const filteredSummaries = computed(() => {
  const q = sidebarSearch.value.trim().toLowerCase()
  if (!q) return diagramsStore.summaries
  return diagramsStore.summaries.filter(d =>
    d.title.toLowerCase().includes(q) || d.slug.toLowerCase().includes(q),
  )
})

const saveStatusLabel = computed(() => {
  switch (autosave.status.value) {
    case 'saved': return 'Saved'
    case 'saving': return 'Saving…'
    case 'unsaved': return 'Unsaved changes'
    case 'error': return autosave.errorMessage.value || 'Save failed'
    default: return ''
  }
})

const saveStatusColor = computed(() => {
  switch (autosave.status.value) {
    case 'saved': return 'var(--method-get)'
    case 'error': return 'var(--method-delete)'
    default: return 'var(--color-text-muted)'
  }
})

const saveStatusBadgeStyle = computed(() => {
  const color = saveStatusColor.value
  return {
    color,
    background: `color-mix(in srgb, ${color} 12%, transparent)`,
  }
})

const linkedRequests = computed(() => {
  if (!activeSlug.value || diagramsStore.current?.slug !== activeSlug.value) return []
  return diagramsStore.current.requests || []
})

const linkedRequestIds = computed(() => linkedRequests.value.map(r => r.id))

onMounted(async () => {
  await diagramsStore.fetchDiagrams(props.workspaceId)
  const slug = props.initialSlug || (route.params.slug as string | undefined)
  if (slug) {
    await selectDiagram(decodeURIComponent(slug))
  }
})

onBeforeRouteLeave(async () => {
  if (autosave.status.value === 'unsaved') {
    await saveNow()
  }
})

function bumpEditor(scene: Record<string, unknown> | null) {
  loadedScene.value = scene
  editorSessionKey.value += 1
  editorMounted.value = true
}

async function selectDiagram(slug: string) {
  if (slug === activeSlug.value && editorMounted.value) return
  if (switching.value) return
  switching.value = true
  loadingDiagram.value = true
  try {
    if (autosave.status.value === 'unsaved') {
      await saveNow()
    }
    autosave.cancelPending()
    pendingScene.value = null
    autosave.markSaved()
    activeSlug.value = slug

    const diagram = await diagramsStore.fetchDiagram(props.workspaceId, slug)
    titleDraft.value = diagram.title
    const scene = diagram.content || null
    sceneCache.set(slug, scene)
    bumpEditor(scene)
    await router.replace(`/workspaces/${props.workspaceId}/diagrams/${encodeURIComponent(slug)}`)
  } finally {
    loadingDiagram.value = false
    switching.value = false
  }
}

async function createDiagram() {
  const title = window.prompt('Story title', `Story ${diagramsStore.summaries.length + 1}`)
  if (!title?.trim()) return
  const diagram = await diagramsStore.createDiagram(props.workspaceId, title.trim())
  await selectDiagram(diagram.slug)
}

function onDiagramChange(scene: ExcalidrawScene) {
  pendingScene.value = scene
  sceneCache.set(activeSlug.value, scene)
  autosave.markUnsaved()
  autosave.debouncedSave()
}

async function saveNow() {
  autosave.cancelPending()
  const scene = pendingScene.value || editorRef.value?.getScene()
  if (!activeSlug.value || !scene) return
  pendingScene.value = scene
  await autosave.forceSave()
}

async function saveTitle() {
  const title = titleDraft.value.trim()
  if (!activeSlug.value || !title) return
  await diagramsStore.updateDiagram(props.workspaceId, activeSlug.value, { title })
}

async function copyDocLink() {
  if (!activeSlug.value) return
  const link = diagramWikilink(activeSlug.value, titleDraft.value.trim() || activeSlug.value)
  const ok = await copyToClipboard(link)
  toast.show(ok ? 'Doc link copied — paste into any doc' : 'Could not copy link', ok ? 'success' : 'error')
}

async function linkRequest(req: LinkedRequest) {
  if (!activeSlug.value) return
  showRequestPicker.value = false
  await diagramsStore.linkRequest(props.workspaceId, activeSlug.value, req.id)
}

async function unlinkRequest(requestId: string) {
  if (!activeSlug.value) return
  await diagramsStore.unlinkRequest(props.workspaceId, activeSlug.value, requestId)
}

async function deleteActive() {
  if (!activeSlug.value || !confirm('Delete this user story?')) return
  const slug = activeSlug.value
  await diagramsStore.deleteDiagram(props.workspaceId, slug)
  sceneCache.delete(slug)
  activeSlug.value = ''
  loadedScene.value = null
  editorMounted.value = false
  pendingScene.value = null
  if (diagramsStore.summaries.length) {
    await selectDiagram(diagramsStore.summaries[0].slug)
  } else {
    await router.replace(`/workspaces/${props.workspaceId}/diagrams`)
  }
}
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}

.delete-btn:hover:not(:disabled) {
  background: color-mix(in srgb, var(--method-delete) 16%, transparent) !important;
  border-color: color-mix(in srgb, var(--method-delete) 55%, var(--color-border)) !important;
}
</style>
