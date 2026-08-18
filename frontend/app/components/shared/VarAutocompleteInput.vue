<template>
  <div
    class="relative min-w-0 var-autocomplete"
    :class="wrapperClass"
  >
    <div
      v-if="showHighlight"
      class="var-autocomplete-shell"
      :class="shellClass"
    >
      <div class="var-autocomplete-stack">
        <div
          ref="mirrorRef"
          class="var-autocomplete-mirror pointer-events-none whitespace-pre"
          :class="fieldClass"
          aria-hidden="true"
        >
          <span
            v-for="(part, i) in highlightParts"
            :key="i"
            :class="part.kind === 'text' ? 'var-autocomplete-text' : 'var-autocomplete-token'"
          >{{ part.text }}</span>
          <span v-if="!modelValue && placeholder" class="var-autocomplete-placeholder">{{ placeholder }}</span>
        </div>
        <input
          ref="inputRef"
          :value="modelValue"
          :type="type"
          :placeholder="''"
          class="var-autocomplete-input"
          :class="fieldClass"
          @input="onInput"
          @keydown="onKeydown"
          @keyup="updateAutocompleteContext"
          @click="updateAutocompleteContext"
          @focus="updateAutocompleteContext"
          @scroll="syncMirrorScroll"
          @blur="close"
        />
      </div>
    </div>
    <input
      v-else
      ref="inputRef"
      :value="modelValue"
      :type="type"
      :placeholder="placeholder"
      :class="inputClass"
      @input="onInput"
      @keydown="onKeydown"
      @keyup="updateAutocompleteContext"
      @click="updateAutocompleteContext"
      @focus="updateAutocompleteContext"
      @blur="close"
    />
    <ul
      v-if="open && filtered.length"
      class="absolute z-50 left-0 right-0 mt-1 max-h-48 overflow-auto rounded border text-xs shadow-lg"
      :style="{ background: 'var(--color-surface-2)', borderColor: 'var(--border)' }"
    >
      <li
        v-for="(item, i) in filtered"
        :key="`${item.source}:${item.name}`"
        class="px-2 py-1.5 cursor-pointer font-mono flex items-center gap-2 min-w-0"
        :style="i === activeIndex ? { background: 'var(--surface-hover)' } : {}"
        @mousedown.prevent="select(item)"
      >
        <span class="truncate">{{ item.name }}</span>
        <span class="shrink-0 uppercase text-[10px]" style="color: var(--text-secondary)">{{ item.source }}</span>
        <span
          v-if="item.value"
          class="truncate ml-auto"
          style="color: var(--text-secondary); max-width: 40%"
        >{{ preview(item.value) }}</span>
      </li>
    </ul>
  </div>
</template>

<script setup lang="ts">
import type { VarSuggestion } from '~/utils/variableSuggestions'
import {
  detectVariableAutocompleteContext,
  splitVariableHighlightParts,
} from '~/utils/variableHighlight'

const props = withDefaults(defineProps<{
  modelValue: string
  collectionId?: string
  placeholder?: string
  type?: string
  inputClass?: string
  wrapperClass?: string
}>(), {
  type: 'text',
  placeholder: '',
  inputClass: 'ui-input w-full',
  wrapperClass: '',
})

const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const { filter } = useAvailableVariables(props.collectionId)

const inputRef = ref<HTMLInputElement | null>(null)
const mirrorRef = ref<HTMLElement | null>(null)
const open = ref(false)
const query = ref('')
const activeIndex = ref(0)
const replaceFrom = ref(0)
const replaceTo = ref(0)

const showHighlight = computed(() => props.type !== 'password')
const shellClass = computed(() => props.inputClass)
const fieldClass = computed(() => {
  if (!showHighlight.value) return props.inputClass
  return props.inputClass.replace(/\bui-input\b/g, '').trim() || 'w-full'
})
const highlightParts = computed(() => splitVariableHighlightParts(props.modelValue))
const filtered = computed(() => filter(query.value))

function preview(value: string) {
  if (value.length <= 48) return value
  return `${value.slice(0, 45)}…`
}

function syncMirrorScroll() {
  const input = inputRef.value
  const mirror = mirrorRef.value
  if (!input || !mirror) return
  mirror.scrollLeft = input.scrollLeft
}

function updateAutocompleteContext() {
  const el = inputRef.value
  if (!el) return
  const cursor = el.selectionStart ?? el.value.length
  const context = detectVariableAutocompleteContext(el.value, cursor)
  if (!context) {
    open.value = false
    return
  }
  replaceFrom.value = context.replaceFrom
  replaceTo.value = context.replaceTo
  query.value = context.query
  activeIndex.value = 0
  open.value = true
  syncMirrorScroll()
}

function onInput(e: Event) {
  const el = e.target as HTMLInputElement
  emit('update:modelValue', el.value)
  nextTick(() => {
    updateAutocompleteContext()
    syncMirrorScroll()
  })
}

function close() {
  open.value = false
}

function select(item: VarSuggestion) {
  const value = props.modelValue
  const insertion = `{{${item.name}}}`
  const newValue = value.slice(0, replaceFrom.value) + insertion + value.slice(replaceTo.value)
  emit('update:modelValue', newValue)
  open.value = false
  nextTick(() => {
    const el = inputRef.value
    if (!el) return
    const pos = replaceFrom.value + insertion.length
    el.focus()
    el.setSelectionRange(pos, pos)
    syncMirrorScroll()
  })
}

function onKeydown(e: KeyboardEvent) {
  if (!open.value || !filtered.value.length) return

  if (e.key === 'ArrowDown') {
    e.preventDefault()
    activeIndex.value = (activeIndex.value + 1) % filtered.value.length
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    activeIndex.value = (activeIndex.value - 1 + filtered.value.length) % filtered.value.length
  } else if (e.key === 'Enter' || e.key === 'Tab') {
    e.preventDefault()
    select(filtered.value[activeIndex.value])
  } else if (e.key === 'Escape') {
    e.preventDefault()
    close()
  }
}

watch(() => props.modelValue, () => {
  nextTick(syncMirrorScroll)
})

watch(() => props.collectionId, () => {
  if (open.value) activeIndex.value = 0
})
</script>

<style scoped>
.var-autocomplete-shell:focus-within {
  border-color: var(--accent);
  box-shadow: 0 0 0 2px color-mix(in srgb, var(--accent) 40%, transparent);
}

.var-autocomplete-stack {
  display: grid;
  width: 100%;
  min-width: 0;
}

.var-autocomplete-stack > * {
  grid-area: 1 / 1;
  min-width: 0;
}

.var-autocomplete-mirror {
  overflow: hidden;
  border: none;
  background: transparent;
  box-shadow: none;
}

.var-autocomplete-input {
  background: transparent !important;
  color: transparent !important;
  caret-color: var(--text-primary);
  border: none !important;
  box-shadow: none !important;
  outline: none;
  padding: 0;
  margin: 0;
  width: 100%;
}

.var-autocomplete-input:focus-visible {
  background: transparent !important;
  outline: none;
  border-color: transparent;
  box-shadow: none;
}

.var-autocomplete-text {
  color: var(--text-primary);
}

.var-autocomplete-token {
  color: var(--accent);
}

.var-autocomplete-placeholder {
  color: var(--text-secondary);
  opacity: 0.7;
}
</style>
