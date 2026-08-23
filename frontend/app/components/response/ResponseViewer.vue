<template>
  <div class="flex flex-col h-full min-h-0 min-w-0" style="background: var(--color-surface-1)">
    <div
      class="flex items-center gap-2 px-3 py-2 border-b text-sm flex-shrink-0 font-mono"
      style="border-color: var(--color-border)"
    >
      <span :class="statusClass">{{ statusLabel }}</span>
      <span class="text-muted">{{ response.timing?.total_ms ?? 0 }}ms</span>
      <span class="text-muted">{{ formatSize(response.body_size) }}</span>
      <div class="flex-1" />
      <ShareButton
        v-if="shareKind && shareSourceId && workspaceId"
        :workspace-id="workspaceId"
        :kind="shareKind"
        :source-id="shareSourceId"
        :default-title="shareTitle"
        label="Share"
        title="Share with teammates"
      />
      <div class="flex gap-1">
        <button
          v-for="tab in viewTabs"
          :key="tab"
          type="button"
          class="text-[10px] px-2 py-0.5 rounded-md transition font-medium"
          :class="viewTab === tab ? 'tab-active' : 'text-muted hover:text-default'"
          @click="viewTab = tab"
        >
          {{ tab }}
        </button>
      </div>
    </div>

    <div
      class="flex-1 min-h-0 min-w-0 flex flex-col text-sm ui-input-editor"
      :class="viewTab === 'Body' ? 'overflow-hidden' : 'overflow-auto p-3 break-words'"
    >
      <ResponseBodyViewer
        v-if="viewTab === 'Body'"
        class="h-full w-full min-h-0 min-w-0"
        :body="response.body"
        :headers="response.headers"
      />
      <div v-else-if="viewTab === 'Headers'" class="break-words">
        <div v-for="(v, k) in response.headers" :key="k" class="mb-1 break-words">
          <span style="color: var(--color-accent)">{{ k }}</span>: {{ v }}
        </div>
      </div>
      <div v-else-if="viewTab === 'Tests'">
        <div
          v-for="(t, i) in response.test_results || []"
          :key="i"
          class="mb-1"
          :style="{ color: t.passed ? 'var(--method-get)' : 'var(--method-delete)' }"
        >
          {{ t.passed ? '✓' : '✗' }} {{ t.name }}
          <span v-if="t.message" class="text-muted"> - {{ t.message }}</span>
        </div>
        <p v-if="!response.test_results?.length" class="text-muted">No tests ran</p>
      </div>
      <div v-else-if="viewTab === 'Console'" class="break-words">
        <div v-for="(line, i) in response.console || []" :key="i" class="break-words">{{ line }}</div>
        <p v-if="!response.console?.length" class="text-muted">No console output</p>
      </div>
      <div v-else-if="viewTab === 'Timing'">
        <div v-for="(v, k) in response.timing" :key="k">{{ k }}: {{ v }}ms</div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
const props = defineProps<{
  response: any
  workspaceId?: string
  shareKind?: 'request' | 'history'
  shareSourceId?: string
  shareTitle?: string
}>()
const viewTabs = ['Body', 'Headers', 'Tests', 'Console', 'Timing']
const viewTab = ref('Body')

const statusLabel = computed(() => {
  if (props.response.error) return 'Error'
  const c = props.response.status_code
  if (!c) return 'Error'
  return String(c)
})

const statusClass = computed(() => {
  if (props.response.error || !props.response.status_code) {
    return 'font-bold text-[var(--method-delete)]'
  }
  const c = props.response.status_code
  if (c >= 200 && c < 300) return 'font-bold text-[var(--method-get)]'
  if (c >= 400) return 'font-bold text-[var(--method-delete)]'
  return 'font-bold text-[var(--method-put)]'
})

function formatSize(n: number) {
  if (!n) return '0 B'
  if (n < 1024) return `${n} B`
  return `${(n / 1024).toFixed(1)} KB`
}
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}
.tab-active {
  background: var(--btn-primary);
  color: var(--color-on-accent);
}
.hover\:text-default:hover {
  color: var(--color-text);
}
</style>
