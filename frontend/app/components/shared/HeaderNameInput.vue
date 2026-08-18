<template>
  <div class="relative min-w-0 flex-1">
    <input
      ref="inputRef"
      :value="modelValue"
      type="text"
      :placeholder="placeholder"
      :class="inputClass"
      @input="onInput"
      @keydown="onKeydown"
      @focus="onFocus"
      @blur="close"
    />
    <ul
      v-if="open && suggestions.length"
      class="absolute z-50 left-0 right-0 mt-1 max-h-56 overflow-auto rounded-md border text-sm shadow-md ui-context-menu"
    >
      <li
        v-for="(name, i) in suggestions"
        :key="name"
        class="px-3 py-1.5 cursor-pointer truncate"
        :class="i === activeIndex ? 'suggestion-active' : ''"
        @mousedown.prevent="select(name)"
      >
        {{ name }}
      </li>
    </ul>
  </div>
</template>

<script setup lang="ts">
import { filterHttpHeaders } from '~/utils/httpHeaders'

withDefaults(defineProps<{
  modelValue: string
  placeholder?: string
  inputClass?: string
}>(), {
  placeholder: 'Header',
  inputClass: 'ui-input w-full',
})

const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const inputRef = ref<HTMLInputElement | null>(null)
const open = ref(false)
const activeIndex = ref(0)
const suggestions = ref<string[]>([])

function refreshSuggestions() {
  suggestions.value = filterHttpHeaders(inputRef.value?.value ?? '')
  activeIndex.value = 0
  open.value = suggestions.value.length > 0
}

function onInput(e: Event) {
  emit('update:modelValue', (e.target as HTMLInputElement).value)
  nextTick(refreshSuggestions)
}

function onFocus() {
  refreshSuggestions()
}

function close() {
  open.value = false
}

function select(name: string) {
  emit('update:modelValue', name)
  open.value = false
  nextTick(() => inputRef.value?.focus())
}

function onKeydown(e: KeyboardEvent) {
  if (!open.value || !suggestions.value.length) return

  if (e.key === 'ArrowDown') {
    e.preventDefault()
    activeIndex.value = (activeIndex.value + 1) % suggestions.value.length
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    activeIndex.value = (activeIndex.value - 1 + suggestions.value.length) % suggestions.value.length
  } else if (e.key === 'Enter' || e.key === 'Tab') {
    e.preventDefault()
    select(suggestions.value[activeIndex.value])
  } else if (e.key === 'Escape') {
    e.preventDefault()
    close()
  }
}
</script>

<style scoped>
.suggestion-active {
  background: var(--color-surface-2);
  color: var(--color-text);
}
</style>
