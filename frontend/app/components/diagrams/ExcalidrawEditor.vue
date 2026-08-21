<template>
  <ClientOnly>
    <component
      :is="ExcalidrawInVue"
      v-if="ExcalidrawInVue && ready"
      :key="editorKey"
      :initialData="initialScene"
      :onChange="handleChange"
    />
    <template #fallback>
      <div class="h-full flex items-center justify-center text-sm text-muted">Loading editor…</div>
    </template>
  </ClientOnly>
</template>

<script setup lang="ts">
import type { Component } from 'vue'
import { toExcalidrawInitialData, toStoredExcalidrawScene, type ExcalidrawScene } from '~/utils/diagramContent'

const props = defineProps<{
  diagramSlug: string
  sessionKey: number
  initialData?: Record<string, unknown> | null
}>()

const emit = defineEmits<{
  change: [data: ExcalidrawScene]
  ready: []
}>()

let excalidrawLoader: Promise<Component> | null = null

function loadExcalidrawComponent(): Promise<Component> {
  if (!excalidrawLoader) {
    excalidrawLoader = (async () => {
      const [{ applyPureReactInVue }, { Excalidraw }] = await Promise.all([
        import('veaury'),
        import('@excalidraw/excalidraw'),
      ])
      await import('@excalidraw/excalidraw/index.css')
      return applyPureReactInVue(Excalidraw)
    })()
  }
  return excalidrawLoader
}

const ExcalidrawInVue = shallowRef<Component | null>(null)
const ready = ref(false)
const latestScene = shallowRef<ExcalidrawScene | null>(null)
let ignoreChangesUntil = 0

const editorKey = computed(() => `${props.diagramSlug}:${props.sessionKey}`)
const initialScene = computed(() => toExcalidrawInitialData(props.initialData))

watch(editorKey, () => {
  ignoreChangesUntil = Date.now() + 800
  latestScene.value = null
}, { immediate: true })

onMounted(async () => {
  ExcalidrawInVue.value = await loadExcalidrawComponent()
  ready.value = true
  emit('ready')
})

function handleChange(
  elements: unknown[],
  appState: Record<string, unknown>,
  files: Record<string, unknown>,
) {
  if (Date.now() < ignoreChangesUntil) return
  const scene = toStoredExcalidrawScene(elements, appState, files)
  latestScene.value = scene
  emit('change', scene)
}

function getScene(): ExcalidrawScene | null {
  return latestScene.value
}

defineExpose({ getScene })
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}
</style>
