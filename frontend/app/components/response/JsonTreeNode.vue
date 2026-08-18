<template>
  <div>
    <div
      v-if="isPrimitive"
      class="group flex items-start gap-2 py-0.5 pr-2 rounded hover:bg-[var(--color-surface-2)]"
    >
      <span v-if="label != null" class="shrink-0" style="color: var(--accent)">{{ label }}:</span>
      <span class="min-w-0 break-all" style="color: var(--color-syntax-string)">{{ displayValue }}</span>
      <button
        type="button"
        class="shrink-0 ml-auto px-1.5 py-0.5 rounded text-[10px] transition opacity-40 group-hover:opacity-100 focus:opacity-100"
        :style="copied ? { background: 'var(--method-get)', color: 'var(--color-bg)' } : { background: 'var(--color-surface-2)', color: 'var(--text-secondary)' }"
        :title="copied ? 'Copied' : 'Copy value'"
        @click="copyValue"
      >
        {{ copied ? 'Copied' : 'Copy' }}
      </button>
    </div>

    <div v-else>
      <div
        class="group flex items-center gap-1 py-0.5 pr-2 rounded hover:bg-[var(--color-surface-2)]"
        :class="isSticky ? 'sticky z-10' : ''"
        :style="isSticky ? { top: stickyTop, background: 'var(--surface)' } : {}"
      >
        <button
          type="button"
          class="shrink-0 w-4 text-left"
          style="color: var(--text-secondary)"
          :aria-expanded="expanded"
          @click="expanded = !expanded"
        >
          {{ expanded ? '▾' : '▸' }}
        </button>
        <span v-if="label != null" class="shrink-0" style="color: var(--accent)">{{ label }}:</span>
        <span class="min-w-0 truncate" style="color: var(--text-secondary)">{{ preview }}</span>
        <button
          type="button"
          class="shrink-0 ml-auto px-1.5 py-0.5 rounded text-[10px] transition opacity-70 group-hover:opacity-100 focus:opacity-100"
          :style="copied ? { background: 'var(--method-get)', color: 'var(--color-bg)' } : { background: 'var(--color-surface-2)', color: 'var(--text-secondary)' }"
          :title="copied ? 'Copied' : 'Copy value'"
          @click="copyValue"
        >
          {{ copied ? 'Copied' : 'Copy' }}
        </button>
      </div>

      <div
        v-if="expanded"
        class="ml-3 pl-3 border-l"
        style="border-color: var(--border)"
      >
        <JsonTreeNode
          v-for="child in children"
          :key="child.path"
          :label="child.label"
          :value="child.value"
          :path="child.path"
          :depth="depth + 1"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { serializeJsonValue } from '~/utils/parseResponseJson'
import { copyToClipboard } from '~/utils/copyToClipboard'

const props = withDefaults(defineProps<{
  label?: string | number | null
  value: unknown
  path: string
  depth?: number
}>(), {
  depth: 0,
})

const expanded = ref(true)
const copied = ref(false)
let copiedTimer: ReturnType<typeof setTimeout> | null = null

const isSticky = computed(() => props.depth <= 1)
const stickyTop = computed(() => `${props.depth * 22}px`)

const isPrimitive = computed(() => props.value === null || typeof props.value !== 'object')

const displayValue = computed(() => {
  if (props.value === null) return 'null'
  if (typeof props.value === 'string') return JSON.stringify(props.value)
  return String(props.value)
})

const preview = computed(() => {
  if (Array.isArray(props.value)) return `Array(${props.value.length})`
  return `Object(${Object.keys(props.value as object).length})`
})

const children = computed(() => {
  if (isPrimitive.value) return []
  if (Array.isArray(props.value)) {
    return props.value.map((item, index) => ({
      label: index,
      value: item,
      path: `${props.path}[${index}]`,
    }))
  }
  return Object.entries(props.value as Record<string, unknown>).map(([key, val]) => ({
    label: key,
    value: val,
    path: props.path === '$' ? key : `${props.path}.${key}`,
  }))
})

async function copyValue() {
  const ok = await copyToClipboard(serializeJsonValue(props.value))
  if (!ok) return
  copied.value = true
  if (copiedTimer) clearTimeout(copiedTimer)
  copiedTimer = setTimeout(() => {
    copied.value = false
    copiedTimer = null
  }, 1500)
}
</script>
