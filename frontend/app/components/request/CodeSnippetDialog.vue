<template>
  <Teleport to="body">
    <div v-if="open" class="fixed inset-0 z-50 flex items-center justify-center">
      <div class="absolute inset-0 ui-overlay" @click="close" />
      <div
        class="relative z-10 w-full max-w-2xl rounded-lg flex flex-col"
        style="background: var(--surface); border: 1px solid var(--border); max-height: 85vh"
      >
        <div class="flex items-center gap-3 px-4 py-3 border-b" style="border-color: var(--border)">
          <h2 class="text-lg font-semibold flex-1">Generate Code</h2>
          <button class="text-lg leading-none opacity-60 hover:opacity-100" @click="close">×</button>
        </div>

        <div class="flex items-center gap-2 px-4 py-2 border-b flex-wrap" style="border-color: var(--border)">
          <select v-model="language" class="ui-input text-sm" @change="onLanguageChange">
            <option v-for="lang in languages" :key="lang.id" :value="lang.id">{{ lang.label }}</option>
          </select>
          <div v-if="language === 'shell'" class="flex gap-1">
            <button
              v-for="fmt in shellFormats"
              :key="fmt.id"
              class="px-2.5 py-1 text-xs rounded transition"
              :style="shellFormat === fmt.id ? { background: 'var(--color-accent)', color: 'var(--color-on-accent)' } : { background: 'var(--color-surface-2)' }"
              @click="shellFormat = fmt.id"
            >
              {{ fmt.label }}
            </button>
          </div>
          <div class="flex-1" />
          <label class="flex items-center gap-1.5 text-xs cursor-pointer" style="color: var(--text-secondary)">
            <input v-model="interpolate" type="checkbox" @change="fetchSnippet" />
            Interpolate Variables
          </label>
        </div>

        <div class="relative flex-1 min-h-0 overflow-hidden p-4">
          <p v-if="loading" class="text-sm" style="color: var(--text-secondary)">Generating…</p>
          <p v-else-if="error" class="text-sm" style="color: var(--method-delete)">{{ error }}</p>
          <div v-else class="relative h-full">
            <button
              class="absolute top-2 right-2 z-10 p-1.5 rounded text-xs opacity-70 hover:opacity-100"
              style="background: var(--color-surface-2)"
              :title="copied ? 'Copied!' : 'Copy'"
              @click="copySnippet"
            >
              {{ copied ? '✓' : '⧉' }}
            </button>
            <pre
              class="h-full overflow-auto rounded p-3 text-xs font-mono leading-relaxed"
              style="background: var(--color-bg); border: 1px solid var(--border); min-height: 200px; max-height: 50vh"
            >{{ snippet }}</pre>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { copyToClipboard } from '~/utils/copyToClipboard'

const props = defineProps<{
  open: boolean
  requestId: string
}>()
const emit = defineEmits<{ close: [] }>()

const envStore = useEnvironmentsStore()
const api = useApi()

const languages = [
  { id: 'shell', label: 'Shell' },
  { id: 'javascript', label: 'JavaScript' },
  { id: 'python', label: 'Python' },
  { id: 'go', label: 'Go' },
]
const shellFormats = [
  { id: 'curl', label: 'curl' },
  { id: 'httpie', label: 'httpie' },
  { id: 'wget', label: 'wget' },
]

const language = ref('shell')
const shellFormat = ref('curl')
const interpolate = ref(true)
const snippet = ref('')
const loading = ref(false)
const error = ref('')
const copied = ref(false)

const effectiveLang = computed(() => language.value === 'shell' ? shellFormat.value : language.value)

watch(() => props.open, (v) => {
  if (v && props.requestId) {
    language.value = 'shell'
    shellFormat.value = 'curl'
    interpolate.value = true
    error.value = ''
    copied.value = false
    fetchSnippet()
  }
})

watch([shellFormat], () => {
  if (props.open) fetchSnippet()
})

function onLanguageChange() {
  fetchSnippet()
}

async function fetchSnippet() {
  if (!props.requestId) return
  loading.value = true
  error.value = ''
  try {
    const params = new URLSearchParams({
      lang: effectiveLang.value,
      interpolate: interpolate.value ? 'true' : 'false',
    })
    if (interpolate.value && envStore.activeId) {
      params.set('environment_id', envStore.activeId)
    }
    const result = await api.get<{ snippet: string }>(`/api/requests/${props.requestId}/snippet?${params}`)
    snippet.value = result.snippet
  } catch (e: any) {
    error.value = e.message || 'Failed to generate code'
    snippet.value = ''
  } finally {
    loading.value = false
  }
}

async function copySnippet() {
  if (!snippet.value) return
  const ok = await copyToClipboard(snippet.value)
  if (ok) {
    copied.value = true
    setTimeout(() => { copied.value = false }, 1500)
  }
}

function close() {
  emit('close')
}
</script>
