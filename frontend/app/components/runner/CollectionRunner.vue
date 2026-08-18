<template>
  <div class="space-y-2">
    <p v-if="!collectionId" class="text-xs" style="color: var(--text-secondary)">Select a collection to run.</p>
    <Button
      variant="primary"
      class="w-full"
      :disabled="!collectionId || running"
      @click="run"
    >
      {{ running ? 'Running…' : 'Run Collection' }}
    </Button>
    <div v-if="report" class="text-xs space-y-1">
      <div class="flex gap-2">
        <span style="color: var(--method-get)">{{ report.passed }} passed</span>
        <span style="color: var(--method-delete)">{{ report.failed }} failed</span>
      </div>
      <div
        v-for="r in report.results"
        :key="r.request_id"
        class="p-1 rounded"
        style="background: var(--color-surface-2)"
      >
        <span :style="{ color: r.passed ? 'var(--method-get)' : 'var(--method-delete)' }">{{ r.passed ? '✓' : '✗' }}</span>
        {{ r.name }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
const props = defineProps<{ collectionId: string | null; workspaceId: string }>()
const envStore = useEnvironmentsStore()
const running = ref(false)
const report = ref<any>(null)

async function run() {
  if (!props.collectionId) return
  running.value = true
  try {
    const api = useApi()
    report.value = await api.post(`/api/collections/${props.collectionId}/run`, {
      environment_id: envStore.activeId,
    })
  } finally {
    running.value = false
  }
}
</script>
