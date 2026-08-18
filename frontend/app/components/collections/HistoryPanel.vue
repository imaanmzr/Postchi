<template>
  <div class="flex flex-col h-full text-sm">
    <div
      class="flex items-center gap-2 px-3 py-2 border-b shrink-0"
      style="border-color: var(--color-border); background: var(--color-bg)"
    >
      <span class="text-muted text-xs">🕐</span>
      <span class="font-medium text-xs tracking-tight">History</span>
      <div class="flex-1" />
      <button
        type="button"
        class="ui-btn ui-btn-ghost text-xs px-2"
        title="Refresh history"
        :disabled="refreshing"
        @click="refresh"
      >
        ↻
      </button>
      <button
        type="button"
        class="ui-btn ui-btn-ghost text-xs px-2"
        title="Back to collections"
        @click="emit('close')"
      >
        ✕
      </button>
    </div>

    <div class="flex-1 ui-scroll-y">
      <p v-if="!entries.length" class="p-4 text-xs text-center text-muted">
        No requests executed yet. All teammates in this workspace share the same history.
      </p>
      <div
        v-for="entry in entries"
        :key="entry.id"
        class="history-row w-full border-b transition group"
        :class="selectedId === entry.id ? 'history-row-active' : ''"
        style="border-color: var(--color-border)"
      >
        <button
          type="button"
          class="w-full text-left px-3 py-2.5"
          @click="selectEntry(entry)"
        >
          <div class="text-[10px] text-muted mb-1 flex justify-between gap-2">
            <span>{{ formatRelativeTime(entry.executed_at) }}</span>
            <span>{{ executorLabel(entry) }}</span>
          </div>
          <div class="flex items-start gap-2 min-w-0">
            <MethodBadge :method="entryMethod(entry)" class="shrink-0 mt-0.5" />
            <span class="flex-1 text-xs truncate font-mono min-w-0" :title="entryUrl(entry)">
              {{ entryUrl(entry) }}
            </span>
            <span
              class="shrink-0 text-xs font-mono font-semibold"
              :class="statusClass(entry.status_code)"
            >
              {{ entry.status_code }}
            </span>
          </div>
        </button>
        <div class="px-3 pb-2 flex justify-end opacity-0 group-hover:opacity-100 transition">
          <ShareButton
            :workspace-id="workspaceId"
            kind="history"
            :source-id="entry.id"
            :default-title="shareTitle(entry)"
            label="Share"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { HistoryEntry } from '~/stores/history'
import { formatRelativeTime } from '~/utils/formatRelativeTime'

const props = defineProps<{
  workspaceId: string
  entries: HistoryEntry[]
  selectedId?: string | null
}>()

const emit = defineEmits<{
  close: []
  select: [entry: HistoryEntry]
}>()

const histStore = useHistoryStore()
const refreshing = ref(false)

function entryMethod(entry: HistoryEntry) {
  return entry.snapshot?.method || 'GET'
}

function entryUrl(entry: HistoryEntry) {
  return entry.snapshot?.url || entry.snapshot?.name || 'Unknown request'
}

function executorLabel(entry: HistoryEntry) {
  return entry.executed_by_name || entry.executed_by_email || 'Teammate'
}

function shareTitle(entry: HistoryEntry) {
  return entry.snapshot?.name || entry.snapshot?.url || 'Shared execution'
}

function statusClass(code: number) {
  if (code >= 200 && code < 300) return 'text-[var(--method-get)]'
  if (code >= 400) return 'text-[var(--method-delete)]'
  return 'text-[var(--method-put)]'
}

async function refresh() {
  refreshing.value = true
  try {
    await histStore.fetch(props.workspaceId)
  } finally {
    refreshing.value = false
  }
}

function selectEntry(entry: HistoryEntry) {
  emit('select', entry)
}
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}
.history-row:hover {
  background: var(--color-surface-2);
}
.history-row-active {
  background: var(--color-surface-2);
  border-left: 2px solid var(--color-accent);
}
</style>
