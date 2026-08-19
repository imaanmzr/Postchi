<template>
  <div
    v-if="open"
    class="fixed inset-0 z-50 flex justify-end"
    @click.self="$emit('close')"
  >
    <div
      class="w-full max-w-lg h-full flex flex-col border-l shadow-xl"
      style="background: var(--color-surface-1); border-color: var(--color-border)"
    >
      <header
        class="flex items-center gap-3 px-4 py-3 border-b shrink-0"
        style="border-color: var(--color-border)"
      >
        <h2 class="font-semibold text-sm flex-1">Link suggestions</h2>
        <Button
          class="text-xs"
          :disabled="docsStore.analyzingLinks"
          @click="analyze"
        >
          {{ docsStore.analyzingLinks ? 'Analyzing…' : 'Re-analyze' }}
        </Button>
        <button type="button" class="text-muted hover:text-default text-lg leading-none" @click="$emit('close')">×</button>
      </header>

      <div v-if="analyzeMessage" class="px-4 py-2 text-xs text-muted border-b" style="border-color: var(--color-border)">
        {{ analyzeMessage }}
      </div>

      <div class="px-4 py-2 border-b flex gap-2 shrink-0" style="border-color: var(--color-border)">
        <Button
          class="text-xs"
          :disabled="!highCount || acceptingAll"
          @click="acceptAllHigh"
        >
          {{ acceptingAll ? 'Accepting…' : `Accept all high (${highCount})` }}
        </Button>
      </div>

      <div v-if="loading" class="p-4 text-sm text-muted">Loading suggestions…</div>

      <ul v-else class="flex-1 overflow-auto ui-scroll-y divide-y" style="--tw-divide-opacity: 1; border-color: var(--color-border)">
        <li v-for="item in docsStore.suggestions" :key="item.id" class="px-4 py-3 space-y-2">
          <div class="flex items-start gap-2">
            <div class="flex-1 min-w-0">
              <div class="text-sm font-medium truncate">{{ item.doc_title }}</div>
              <div class="flex items-center gap-2 mt-1">
                <MethodBadge :method="item.method" class="shrink-0 scale-90" />
                <span class="text-xs truncate">{{ item.request_name }}</span>
              </div>
              <div class="text-[10px] text-muted truncate mt-0.5">{{ item.url }}</div>
              <div class="text-[10px] text-muted mt-1">
                <span
                  class="px-1.5 py-0.5 rounded mr-1"
                  :style="confidenceStyle(item.confidence)"
                >{{ item.confidence }}</span>
                {{ reasonLabel(item.reason) }}
              </div>
            </div>
            <div class="flex gap-1 shrink-0">
              <Button class="text-xs px-2 py-1" @click="accept(item.id)">Accept</Button>
              <Button class="text-xs px-2 py-1" variant="ghost" @click="reject(item.id)">Reject</Button>
            </div>
          </div>
        </li>
        <li v-if="!docsStore.suggestions.length" class="px-4 py-8 text-sm text-muted text-center">
          No pending suggestions. Run analyze to find doc ↔ API matches.
        </li>
      </ul>
    </div>
  </div>
</template>

<script setup lang="ts">
const props = defineProps<{
  open: boolean
  workspaceId: string
}>()

defineEmits<{ close: [] }>()

const docsStore = useDocsStore()
const toast = useToast()
const loading = ref(false)
const acceptingAll = ref(false)
const analyzeMessage = ref('')

const highCount = computed(() =>
  docsStore.suggestions.filter(s => s.confidence === 'high').length,
)

function confidenceStyle(confidence: string) {
  if (confidence === 'high') {
    return { background: 'var(--color-surface-2)', color: 'var(--color-accent)' }
  }
  return { background: 'var(--color-surface-2)', color: 'var(--color-text-muted)' }
}

function reasonLabel(reason: string) {
  const labels: Record<string, string> = {
    content_method_path: 'HTTP method/path in doc body',
    path_alignment: 'Doc path matches request path',
    title_similarity: 'Similar title/name',
    folder_alignment: 'Folder mirrors collection',
  }
  return labels[reason] ?? reason
}

watch(() => [props.open, props.workspaceId] as const, async ([open, wsId]) => {
  if (!open || !wsId) return
  loading.value = true
  try {
    await docsStore.fetchSuggestions(wsId, 'pending')
  } finally {
    loading.value = false
  }
}, { immediate: true })

async function analyze() {
  analyzeMessage.value = ''
  try {
    const result = await docsStore.analyzeLinks(props.workspaceId)
    analyzeMessage.value = `Found ${result.pending_total ?? 0} pending suggestion(s).`
    await docsStore.fetchGraph(props.workspaceId)
  } catch (e) {
    analyzeMessage.value = e instanceof Error ? e.message : 'Analyze failed'
  }
}

async function accept(id: string) {
  await docsStore.acceptSuggestion(props.workspaceId, id)
  await docsStore.fetchGraph(props.workspaceId)
  toast.show('Link accepted.')
}

async function reject(id: string) {
  await docsStore.rejectSuggestion(props.workspaceId, id)
  toast.show('Suggestion dismissed.')
}

async function acceptAllHigh() {
  acceptingAll.value = true
  try {
    const result = await docsStore.acceptAllHighSuggestions(props.workspaceId)
    await docsStore.fetchGraph(props.workspaceId)
    toast.show(`Accepted ${result.accepted ?? 0} link(s).`)
  } finally {
    acceptingAll.value = false
  }
}
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}
</style>
