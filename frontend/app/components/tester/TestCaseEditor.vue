<template>
  <div v-if="loading" class="h-full flex items-center justify-center text-sm text-muted">Loading…</div>
  <div v-else-if="!testCase" class="h-full flex items-center justify-center text-sm text-muted">Test case not found.</div>
  <div v-else class="h-full flex flex-col overflow-hidden">
    <div class="px-4 py-3 border-b shrink-0 flex items-center gap-3" style="border-color: var(--color-border)">
      <Input v-model="titleDraft" class="max-w-md font-medium" @blur="saveTitle" />
      <div class="flex-1" />
      <span class="text-[11px] uppercase tracking-wide text-muted">{{ saveStatusLabel }}</span>
      <Button class="text-xs" style="color: var(--method-delete)" @click="deleteCase">Delete</Button>
    </div>

    <div class="flex-1 min-h-0 grid grid-cols-1 lg:grid-cols-2">
      <section class="border-r flex flex-col min-h-0 overflow-hidden" style="border-color: var(--color-border)">
        <div class="px-4 py-2 border-b text-xs font-medium shrink-0" style="border-color: var(--color-border)">
          Description
        </div>
        <div class="flex-1 min-h-0 overflow-auto p-4">
          <textarea
            v-model="descriptionDraft"
            class="ui-input w-full h-full min-h-[240px] text-sm resize-none"
            placeholder="Steps, expected results, notes…"
            @input="onDescriptionInput"
          />
        </div>
      </section>

      <LinkedRequestsPanel
        :requests="testCase.requests || []"
        @link="showPicker = true"
        @unlink="unlink"
      />
    </div>

    <CrossWorkspaceRequestPicker
      :open="showPicker"
      :exclude-ids="linkedRequestIds"
      @select="linkRequest"
      @close="showPicker = false"
    />
  </div>
</template>

<script setup lang="ts">
import type { LinkedRequest } from '~/utils/linkableRequests'

const props = defineProps<{
  workspaceId: string
  testCaseId: string
}>()

const testCasesStore = useTestCasesStore()

const loading = ref(true)
const titleDraft = ref('')
const descriptionDraft = ref('')
const showPicker = ref(false)

const autosave = useDocAutosave({
  save: async () => {
    await testCasesStore.updateCase(props.workspaceId, props.testCaseId, {
      description: descriptionDraft.value,
    })
  },
})

const testCase = computed(() => testCasesStore.current)

const linkedRequestIds = computed(() => (testCase.value?.requests || []).map(r => r.id))

const saveStatusLabel = computed(() => {
  switch (autosave.status.value) {
    case 'saved': return 'Saved'
    case 'saving': return 'Saving…'
    case 'unsaved': return 'Unsaved'
    case 'error': return 'Save failed'
    default: return ''
  }
})

watch(() => props.testCaseId, loadCase, { immediate: true })

async function loadCase() {
  loading.value = true
  autosave.markSaved()
  try {
    const tc = await testCasesStore.fetchCase(props.workspaceId, props.testCaseId)
    titleDraft.value = tc.title
    descriptionDraft.value = tc.description
  } finally {
    loading.value = false
  }
}

async function saveTitle() {
  const title = titleDraft.value.trim()
  if (!title) return
  await testCasesStore.updateCase(props.workspaceId, props.testCaseId, { title })
}

function onDescriptionInput() {
  autosave.markUnsaved()
  autosave.debouncedSave()
}

async function linkRequest(req: LinkedRequest) {
  showPicker.value = false
  await testCasesStore.linkRequest(props.workspaceId, props.testCaseId, req.id)
}

async function unlink(requestId: string) {
  await testCasesStore.unlinkRequest(props.workspaceId, props.testCaseId, requestId)
}

async function deleteCase() {
  if (!confirm('Delete this test case?')) return
  await testCasesStore.deleteCase(props.workspaceId, props.testCaseId)
}
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}
</style>
