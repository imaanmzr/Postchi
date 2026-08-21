<template>
  <div class="h-full w-full min-w-0 min-h-0 flex flex-col overflow-hidden">
    <div
      v-if="parsedJson !== null"
      class="flex items-center justify-between gap-2 px-2 py-1 border-b flex-shrink-0"
      style="border-color: var(--color-border)"
    >
      <div class="flex gap-1">
        <button
          v-for="mode in bodyModes"
          :key="mode"
          type="button"
          class="text-[10px] px-2 py-0.5 rounded-md transition font-medium"
          :class="viewMode === mode ? 'mode-active' : 'text-muted hover:text-default'"
          @click="viewMode = mode"
        >
          {{ mode }}
        </button>
      </div>
      <button
        type="button"
        class="text-[10px] px-2 py-0.5 rounded-md transition font-medium"
        :class="copiedAll ? 'copy-success' : 'copy-idle'"
        @click="copyAll"
      >
        {{ copiedAll ? 'Copied body' : 'Copy body' }}
      </button>
    </div>

    <JsonTreeViewer
      v-if="parsedJson !== null && viewMode === 'Tree'"
      :data="parsedJson"
      class="flex-1 min-h-0 min-w-0 w-full"
    />
    <div
      v-show="parsedJson === null || viewMode === 'Raw'"
      ref="viewerEl"
      class="viewer-host flex-1 min-h-0 min-w-0 w-full overflow-hidden ui-input-editor"
      style="background: var(--color-editor-bg)"
    />
  </div>
</template>

<script setup lang="ts">
import { EditorView, basicSetup } from 'codemirror'
import { EditorState } from '@codemirror/state'
import { json } from '@codemirror/lang-json'
import { xml } from '@codemirror/lang-xml'
import type { Extension } from '@codemirror/state'
import { editorTheme, editorFillLayout, editorLineWrap, editorSyntax } from '~/utils/codemirror/editorTheme'
import { detectResponseBodyLang, formatResponseBody } from '~/utils/formatResponseBody'
import { parseResponseJson } from '~/utils/parseResponseJson'
import { copyToClipboard } from '~/utils/copyToClipboard'

const props = defineProps<{
  body: unknown
  headers?: Record<string, string>
}>()

const bodyModes = ['Tree', 'Raw'] as const
type BodyMode = typeof bodyModes[number]

const viewerEl = ref<HTMLElement>()
const viewMode = ref<BodyMode>('Tree')
const copiedAll = ref(false)
let view: EditorView | null = null
let copiedAllTimer: ReturnType<typeof setTimeout> | null = null
let resizeObserver: ResizeObserver | null = null

const displayBody = computed(() => formatResponseBody(props.body))
const lang = computed(() => detectResponseBodyLang(props.body, props.headers))
const parsedJson = computed(() => parseResponseJson(props.body))

const viewerLayout = EditorView.theme({
  '&': {
    minHeight: '0',
  },
  '.cm-content': {
    padding: '0.75rem 0',
  },
}, { dark: true })

function measureView() {
  if (view) view.requestMeasure()
}

function bindResizeObserver() {
  resizeObserver?.disconnect()
  if (!viewerEl.value) return
  resizeObserver = new ResizeObserver(() => measureView())
  resizeObserver.observe(viewerEl.value)
}

function langExt(language: string): Extension {
  if (language === 'json') return json()
  if (language === 'xml') return xml()
  return []
}

function createView() {
  if (!viewerEl.value) return
  view?.destroy()
  view = new EditorView({
    parent: viewerEl.value,
    state: EditorState.create({
      doc: displayBody.value,
      extensions: [
        basicSetup,
        editorLineWrap,
        EditorState.readOnly.of(true),
        EditorView.editable.of(false),
        langExt(lang.value),
        editorTheme,
        editorSyntax,
        editorFillLayout,
        viewerLayout,
      ],
    }),
  })
  nextTick(measureView)
}

function updateDoc() {
  if (!view) return
  const next = displayBody.value
  if (next !== view.state.doc.toString()) {
    view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: next } })
  }
}

async function copyAll() {
  const ok = await copyToClipboard(displayBody.value)
  if (!ok) return

  copiedAll.value = true
  if (copiedAllTimer) clearTimeout(copiedAllTimer)
  copiedAllTimer = setTimeout(() => {
    copiedAll.value = false
  }, 1500)
}

watch(parsedJson, (value) => {
  viewMode.value = value === null ? 'Raw' : 'Tree'
})

onMounted(() => {
  createView()
  bindResizeObserver()
})
watch(lang, createView)
watch(displayBody, updateDoc)
watch(viewMode, (mode) => {
  if (mode === 'Raw') nextTick(() => {
    createView()
    bindResizeObserver()
  })
})
onUnmounted(() => {
  resizeObserver?.disconnect()
  view?.destroy()
  if (copiedAllTimer) clearTimeout(copiedAllTimer)
})
</script>

<style scoped>
.mode-active {
  background: var(--btn-primary);
  color: var(--color-on-accent);
}
.text-muted {
  color: var(--color-text-muted);
}
.hover\:text-default:hover {
  color: var(--color-text);
}
.copy-idle {
  background: var(--color-surface-2);
  color: var(--color-text-muted);
}
.copy-success {
  background: var(--method-get);
  color: var(--color-bg);
}
.viewer-host :deep(.cm-editor) {
  width: 100%;
  height: 100%;
  min-height: 0;
}
</style>
