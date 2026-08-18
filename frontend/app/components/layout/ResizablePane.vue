<template>
  <div
    ref="container"
    class="relative flex-shrink-0"
    :class="isVertical ? 'w-full min-h-0' : 'h-full'"
    :style="paneStyle"
  >
    <slot />
    <div
      v-if="resizable"
      class="absolute z-10 group flex items-center justify-center"
      :class="handleClass"
      @mousedown="startResize"
    >
      <div
        v-if="isVertical"
        class="h-0.5 w-12 rounded-full transition group-hover:bg-accent"
        style="background: var(--color-border)"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
const props = withDefaults(defineProps<{
  direction?: 'horizontal' | 'vertical'
  initialWidth?: number
  minWidth?: number
  maxWidth?: number
  initialHeight?: number
  minHeight?: number
  maxHeight?: number
  side?: 'left' | 'right'
  resizable?: boolean
}>(), {
  direction: 'horizontal',
  initialWidth: 256,
  minWidth: 180,
  maxWidth: 480,
  initialHeight: 320,
  minHeight: 120,
  maxHeight: 800,
  side: 'right',
  resizable: true,
})

const emit = defineEmits<{ resize: [size: number] }>()

const isVertical = computed(() => props.direction === 'vertical')
const size = ref(isVertical.value ? props.initialHeight : props.initialWidth)

watch(() => isVertical.value ? props.initialHeight : props.initialWidth, (v) => {
  if (v != null) size.value = v
})

const paneStyle = computed(() =>
  isVertical.value ? { height: `${size.value}px` } : { width: `${size.value}px` },
)

const handleClass = computed(() =>
  isVertical.value
    ? 'top-0 left-0 right-0 h-3 -translate-y-1/2 cursor-row-resize hover:bg-[var(--color-accent-muted)]'
    : props.side === 'right'
      ? 'top-0 bottom-0 right-0 w-1 cursor-col-resize hover:bg-[var(--color-accent-muted)]'
      : 'top-0 bottom-0 left-0 w-1 cursor-col-resize hover:bg-[var(--color-accent-muted)]',
)

function clamp(next: number) {
  if (isVertical.value) {
    return Math.min(props.maxHeight, Math.max(props.minHeight, next))
  }
  return Math.min(props.maxWidth, Math.max(props.minWidth, next))
}

function startResize(e: MouseEvent) {
  e.preventDefault()
  const start = isVertical.value ? e.clientY : e.clientX
  const startSize = size.value
  const horizontalDir = props.side === 'right' ? 1 : -1
  document.body.style.userSelect = 'none'
  document.body.style.cursor = isVertical.value ? 'row-resize' : 'col-resize'

  function onMove(ev: MouseEvent) {
    const delta = isVertical.value ? start - ev.clientY : (ev.clientX - start) * horizontalDir
    size.value = clamp(startSize + delta)
    emit('resize', size.value)
  }

  function onUp() {
    document.body.style.userSelect = ''
    document.body.style.cursor = ''
    window.removeEventListener('mousemove', onMove)
    window.removeEventListener('mouseup', onUp)
  }

  window.addEventListener('mousemove', onMove)
  window.addEventListener('mouseup', onUp)
}
</script>