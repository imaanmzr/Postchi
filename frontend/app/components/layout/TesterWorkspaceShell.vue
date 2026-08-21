<template>
  <div class="h-screen flex flex-col overflow-hidden app-bg">
    <WorkspaceChrome
      :workspace-id="workspaceId"
      :workspace-name="workspaceName"
      workspace-type="tester"
      :nav-items="navItems"
    />
    <div class="flex flex-1 min-h-0 overflow-hidden">
      <ResizablePane :initial-width="280" side="right">
        <aside
          class="h-full border-r flex flex-col min-h-0 overflow-hidden"
          style="border-color: var(--color-border); background: var(--color-surface-1)"
        >
          <div class="p-3 border-b shrink-0" style="border-color: var(--color-border)">
            <Button variant="primary" class="w-full text-xs" @click="createCase">
              <Plus :size="14" class="mr-1.5" />
              New test case
            </Button>
          </div>
          <div v-if="testCasesStore.loading" class="p-4 text-xs text-muted">Loading…</div>
          <div v-else-if="!testCasesStore.cases.length" class="p-4 text-center">
            <p class="text-sm font-medium mb-1">Create your first test case</p>
            <p class="text-xs text-muted mb-3">Document scenarios and link them to API requests.</p>
            <Button variant="primary" class="text-xs" @click="createCase">Create test case</Button>
          </div>
          <ul v-else class="flex-1 overflow-y-auto p-2 space-y-1">
            <li v-for="item in testCasesStore.cases" :key="item.id">
              <button
                type="button"
                class="w-full text-left px-3 py-2 rounded text-sm transition"
                :style="item.id === activeId
                  ? { background: 'var(--color-surface-2)', fontWeight: 600 }
                  : { color: 'var(--color-text-muted)' }"
                @click="selectCase(item.id)"
              >
                <div class="truncate">{{ item.title }}</div>
                <div v-if="item.requests?.length" class="text-[10px] mt-0.5 opacity-70">
                  {{ item.requests.length }} linked request{{ item.requests.length === 1 ? '' : 's' }}
                </div>
              </button>
            </li>
          </ul>
        </aside>
      </ResizablePane>

      <main class="flex-1 min-w-0 overflow-hidden">
        <TestCaseEditor
          v-if="activeId"
          :workspace-id="workspaceId"
          :test-case-id="activeId"
        />
        <div v-else class="h-full flex items-center justify-center text-sm text-muted">
          Select or create a test case.
        </div>
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Plus } from 'lucide-vue-next'

const props = defineProps<{
  workspaceId: string
  workspaceName: string
}>()

const testCasesStore = useTestCasesStore()
const activeId = ref('')

const navItems = computed(() => [
  { label: 'Test Cases', to: `/workspaces/${props.workspaceId}`, match: 'exact' as const },
])

onMounted(async () => {
  await testCasesStore.fetchCases(props.workspaceId)
  if (testCasesStore.cases.length) {
    activeId.value = testCasesStore.cases[0].id
  }
})

async function createCase() {
  const title = `Test case ${testCasesStore.cases.length + 1}`
  const tc = await testCasesStore.createCase(props.workspaceId, title)
  activeId.value = tc.id
}

function selectCase(id: string) {
  activeId.value = id
}
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}
</style>
