<template>
  <div
    ref="editorEl"
    class="w-full min-w-0 rounded-md overflow-hidden min-h-[120px] ui-input-editor"
    style="border: 1px solid var(--color-border); background: var(--color-editor-bg)"
  />
</template>

<script setup lang="ts">
import { EditorView } from 'codemirror'
import { EditorState } from '@codemirror/state'
import { javascript } from '@codemirror/lang-javascript'
import { json } from '@codemirror/lang-json'
import { xml } from '@codemirror/lang-xml'
import { editorBasicSetup } from '~/utils/codemirror/editorSetup'
import { editorLineWrap, editorSelectionTheme, editorSyntax, editorTheme } from '~/utils/codemirror/editorTheme'
import { formatResponseBody } from '~/utils/formatResponseBody'

const model = defineModel<string>({ required: true })
const props = withDefaults(defineProps<{ lang?: string }>(), { lang: 'javascript' })

const editorEl = ref<HTMLElement>()
let view: EditorView | null = null

function langExt(lang: string) {
  if (lang === 'json') return json()
  if (lang === 'xml' || lang === 'html') return xml()
  return javascript()
}

function formatForLang(value: string, lang: string): string {
  if (lang === 'json') return formatResponseBody(value)
  return value
}

function syncFormattedDoc(value: string) {
  if (!view || props.lang !== 'json') return
  const formatted = formatForLang(value, props.lang)
  if (formatted === view.state.doc.toString()) return
  view.dispatch({ changes: { from: 0, to: view.state.doc.length, insert: formatted } })
  if (formatted !== model.value) model.value = formatted
}

function createView() {
  if (!editorEl.value) return
  view?.destroy()
  const doc = formatForLang(model.value, props.lang)
  if (doc !== model.value && props.lang === 'json') {
    model.value = doc
  }
  view = new EditorView({
    parent: editorEl.value,
    state: EditorState.create({
      doc,
      extensions: [
        editorBasicSetup,
        editorLineWrap,
        langExt(props.lang),
        editorSelectionTheme,
        editorTheme,
        editorSyntax,
        EditorView.updateListener.of((u) => {
          if (u.docChanged) model.value = view!.state.doc.toString()
        }),
        EditorView.domEventHandlers({
          blur() {
            syncFormattedDoc(view!.state.doc.toString())
          },
        }),
      ],
    }),
  })
}

onMounted(createView)

watch(() => props.lang, createView)

watch(() => model.value, (v) => {
  if (!view || v === view.state.doc.toString()) return
  syncFormattedDoc(v)
})

onUnmounted(() => view?.destroy())
</script>
