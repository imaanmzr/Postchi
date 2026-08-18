<template>
  <Teleport to="body">
    <div
      v-if="open"
      class="fixed inset-0 z-[100] flex items-start justify-center pt-[15vh] px-4"
      @click.self="emit('close')"
    >
      <div
        class="w-full max-w-lg rounded-lg border shadow-2xl overflow-hidden"
        style="background: var(--color-surface-1); border-color: var(--color-border)"
      >
        <input
          ref="inputEl"
          v-model="query"
          type="text"
          placeholder="Jump to doc…"
          class="ui-input w-full rounded-none border-0 border-b text-sm px-4 py-3"
          style="border-color: var(--color-border)"
          @keydown.escape="emit('close')"
          @keydown.enter.prevent="selectCurrent"
          @keydown.down.prevent="move(1)"
          @keydown.up.prevent="move(-1)"
        />
        <ul class="max-h-72 overflow-auto ui-scroll-y py-1">
          <li
            v-for="(item, idx) in results"
            :key="item.slug"
            class="px-4 py-2 text-sm cursor-pointer transition"
            :class="idx === selected ? 'bg-surface-2' : 'hover:bg-surface-2'"
            @click="select(item.slug)"
          >
            <div class="font-medium truncate">{{ item.title }}</div>
            <div class="text-xs text-muted truncate">{{ item.source_path || item.slug }}</div>
          </li>
          <li v-if="!results.length" class="px-4 py-3 text-xs text-muted">No matches</li>
        </ul>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import Fuse from 'fuse.js'
import type { DocSummary } from '~/utils/docsTree'

const props = defineProps<{
  open: boolean
  summaries: DocSummary[]
}>()

const emit = defineEmits<{
  close: []
  select: [slug: string]
}>()

const query = ref('')
const selected = ref(0)
const inputEl = ref<HTMLInputElement>()

const fuse = computed(() => new Fuse(props.summaries, {
  keys: [
    { name: 'title', weight: 0.5 },
    { name: 'source_path', weight: 0.35 },
    { name: 'slug', weight: 0.15 },
  ],
  threshold: 0.35,
}))

const results = computed(() => {
  const q = query.value.trim()
  if (!q) return props.summaries.slice(0, 20)
  return fuse.value.search(q).slice(0, 20).map(r => r.item)
})

watch(() => props.open, (isOpen) => {
  if (isOpen) {
    query.value = ''
    selected.value = 0
    nextTick(() => inputEl.value?.focus())
  }
})

watch(query, () => { selected.value = 0 })

function move(delta: number) {
  selected.value = Math.max(0, Math.min(selected.value + delta, results.value.length - 1))
}

function selectCurrent() {
  const item = results.value[selected.value]
  if (item) select(item.slug)
}

function select(slug: string) {
  emit('select', slug)
  emit('close')
}
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}
</style>
