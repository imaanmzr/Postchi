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
          :placeholder="placeholder"
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
            :key="getKey(item)"
            class="px-4 py-2 text-sm cursor-pointer transition"
            :class="idx === selected ? 'bg-surface-2' : 'hover:bg-surface-2'"
            @click="select(item)"
          >
            <slot name="item" :item="item" :selected="idx === selected">
              <div class="font-medium truncate">{{ getTitle(item) }}</div>
              <div v-if="subtitle(item)" class="text-xs text-muted truncate">{{ subtitle(item) }}</div>
            </slot>
          </li>
          <li v-if="!results.length" class="px-4 py-3 text-xs text-muted">{{ emptyLabel }}</li>
        </ul>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts" generic="T">
import Fuse from 'fuse.js'

const props = withDefaults(defineProps<{
  open: boolean
  items: T[]
  getKey: (item: T) => string
  getTitle: (item: T) => string
  getSubtitle?: (item: T) => string
  searchKeys?: string[]
  placeholder?: string
  emptyLabel?: string
}>(), {
  searchKeys: () => ['title'],
  placeholder: 'Search…',
  emptyLabel: 'No matches',
})

function subtitle(item: T) {
  return props.getSubtitle?.(item) ?? ''
}

const emit = defineEmits<{
  close: []
  select: [item: T]
}>()

const query = ref('')
const selected = ref(0)
const inputEl = ref<HTMLInputElement>()

const fuse = computed(() => new Fuse(props.items, {
  keys: props.searchKeys.map(name => ({ name, weight: 1 })),
  threshold: 0.35,
}))

const results = computed(() => {
  const q = query.value.trim()
  if (!q) return props.items.slice(0, 20)
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
  if (item) select(item)
}

function select(item: T) {
  emit('select', item)
  emit('close')
}
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}
.hover\:bg-surface-2:hover,
.bg-surface-2 {
  background: var(--color-surface-2);
}
</style>
