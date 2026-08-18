<template>
  <div
    ref="editorEl"
    class="w-full min-w-0 rounded-md overflow-hidden h-full ui-input-editor"
    style="border: 1px solid var(--color-border); background: var(--color-editor-bg)"
  />
</template>

<script setup lang="ts">
import { EditorView } from '@codemirror/view'
import { EditorState } from '@codemirror/state'
import { createMarkdownExtensions } from '~/utils/codemirror/markdownExtensions'
import { editorFillLayout } from '~/utils/codemirror/editorTheme'

const model = defineModel<string>({ required: true })

const props = defineProps<{
  placeholder?: string
  docCompletions?: { label: string, slug: string }[]
}>()

const emit = defineEmits<{
  'toggle-preview': []
  'force-save': []
  'view-ready': [view: EditorView | null]
}>()

const editorEl = ref<HTMLElement>()
const editorView = shallowRef<EditorView | null>(null)
let currentSlug = ''

function createView(content: string) {
  if (!editorEl.value) return
  editorView.value?.destroy()
  const view = new EditorView({
    parent: editorEl.value,
    state: EditorState.create({
      doc: content,
      extensions: [
        ...createMarkdownExtensions({
          placeholder: props.placeholder,
          docCompletions: props.docCompletions,
          onTogglePreview: () => emit('toggle-preview'),
          onForceSave: () => emit('force-save'),
        }),
        editorFillLayout,
        EditorView.updateListener.of((u) => {
          if (u.docChanged) model.value = view.state.doc.toString()
        }),
      ],
    }),
  })
  editorView.value = view
  emit('view-ready', view)
}

onMounted(() => createView(model.value))

watch(() => model.value, (v) => {
  const view = editorView.value
  if (!view || v === view.state.doc.toString()) return
  view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: v } })
})

function setContent(content: string, slug?: string) {
  if (slug && slug !== currentSlug) {
    currentSlug = slug
    createView(content)
    return
  }
  const view = editorView.value
  if (!view) {
    createView(content)
    return
  }
  if (content !== view.state.doc.toString()) {
    view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: content } })
  }
}

onUnmounted(() => {
  editorView.value?.destroy()
  emit('view-ready', null)
})

defineExpose({
  editorView,
  setContent,
  focus: () => editorView.value?.focus(),
})
</script>
