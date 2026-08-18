import { EditorView } from '@codemirror/view'
import type { Ref } from 'vue'

function scrollRatio(el: HTMLElement): number {
  const max = el.scrollHeight - el.clientHeight
  if (max <= 0) return 0
  return el.scrollTop / max
}

function setScrollRatio(el: HTMLElement, ratio: number) {
  const max = el.scrollHeight - el.clientHeight
  el.scrollTop = Math.max(0, Math.min(1, ratio)) * max
}

export function useEditorPreviewScrollSync(options: {
  editorView: Ref<EditorView | null>
  previewEl: Ref<HTMLElement | null>
  enabled: Ref<boolean>
}) {
  let suppressEditorScroll = false
  let suppressPreviewScroll = false
  let rafId: number | null = null
  let pendingEditor = false
  let pendingPreview = false

  function syncEditorToPreview() {
    if (suppressEditorScroll || !options.enabled.value) return
    const view = options.editorView.value
    const preview = options.previewEl.value
    if (!view || !preview) return

    suppressPreviewScroll = true
    try {
      setScrollRatio(preview, scrollRatio(view.scrollDOM))
    } finally {
      suppressPreviewScroll = false
    }
  }

  function syncPreviewToEditor() {
    if (suppressPreviewScroll || !options.enabled.value) return
    const view = options.editorView.value
    const preview = options.previewEl.value
    if (!view || !preview) return

    suppressEditorScroll = true
    try {
      setScrollRatio(view.scrollDOM, scrollRatio(preview))
    } finally {
      suppressEditorScroll = false
    }
  }

  function flushPending() {
    rafId = null
    if (pendingEditor) syncEditorToPreview()
    if (pendingPreview) syncPreviewToEditor()
    pendingEditor = false
    pendingPreview = false
  }

  function scheduleFlush() {
    if (rafId != null) return
    rafId = requestAnimationFrame(flushPending)
  }

  function onEditorScroll() {
    pendingEditor = true
    scheduleFlush()
  }

  function onPreviewScroll() {
    pendingPreview = true
    scheduleFlush()
  }

  function attach() {
    const view = options.editorView.value
    const preview = options.previewEl.value
    if (!view || !preview) return

    view.scrollDOM.addEventListener('scroll', onEditorScroll, { passive: true })
    preview.addEventListener('scroll', onPreviewScroll, { passive: true })
  }

  function detach() {
    const view = options.editorView.value
    const preview = options.previewEl.value
    view?.scrollDOM.removeEventListener('scroll', onEditorScroll)
    preview?.removeEventListener('scroll', onPreviewScroll)
    if (rafId != null) {
      cancelAnimationFrame(rafId)
      rafId = null
    }
    pendingEditor = false
    pendingPreview = false
  }

  watch([options.editorView, options.previewEl, options.enabled], () => {
    detach()
    if (options.enabled.value) attach()
  }, { immediate: true })

  onUnmounted(detach)

  return { attach, detach }
}
