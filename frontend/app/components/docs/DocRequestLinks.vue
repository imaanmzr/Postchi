<template>
  <aside
    class="shrink-0 border-l w-56 flex flex-col min-h-0"
    style="border-color: var(--color-border); background: var(--color-surface-1)"
  >
    <div class="px-3 py-2 border-b shrink-0" style="border-color: var(--color-border)">
      <div class="flex items-center justify-between gap-2">
        <span class="text-xs font-medium">
          Linked requests
          <span v-if="links.length" class="text-muted font-normal">({{ links.length }})</span>
        </span>
        <button
          type="button"
          class="text-[10px] px-2 py-0.5 rounded hover:bg-surface-2 transition"
          style="color: var(--color-accent)"
          @click="requestPickerOpen = true"
        >
          Link
        </button>
      </div>
    </div>

    <div v-if="loading" class="px-3 py-2 text-xs text-muted">Loading…</div>

    <ul v-else class="flex-1 min-h-0 overflow-auto ui-scroll-y py-1">
      <li v-for="link in links" :key="link.request_id" class="group px-2 py-1">
        <div class="flex items-center gap-1.5 rounded hover:bg-surface-2">
          <NuxtLink
            :to="requestUrl(link.request_id)"
            class="flex items-center gap-1.5 min-w-0 flex-1 rounded px-1 py-1.5 transition"
          >
            <MethodBadge :method="link.method" class="shrink-0 scale-90" />
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-1 flex-wrap">
                <span class="text-xs font-medium truncate">{{ link.request_name }}</span>
                <span
                  v-if="link.link_sources.includes('frontmatter')"
                  class="text-[9px] px-1 py-0.5 rounded"
                  style="background: var(--color-surface-2); color: var(--color-text-muted)"
                >auto</span>
                <span
                  v-if="link.link_sources.includes('manual')"
                  class="text-[9px] px-1 py-0.5 rounded"
                  style="background: var(--color-surface-2); color: var(--color-accent)"
                >manual</span>
                <span
                  v-if="link.link_sources.includes('suggested')"
                  class="text-[9px] px-1 py-0.5 rounded"
                  style="background: var(--color-surface-2); color: var(--color-syntax-number)"
                >suggested</span>
              </div>
              <div class="text-[10px] text-muted truncate">{{ link.collection_name }}</div>
            </div>
          </NuxtLink>
          <div class="flex flex-col gap-0.5 shrink-0">
            <template v-if="link.link_sources.includes('suggested') && link.suggestion_id">
              <button
                type="button"
                class="text-[10px] px-1"
                style="color: var(--color-accent)"
                title="Accept suggestion"
                @click="accept(link)"
              >
                ✓
              </button>
              <button
                type="button"
                class="text-[10px] px-1 text-muted hover:text-default"
                title="Reject suggestion"
                @click="reject(link)"
              >
                ×
              </button>
            </template>
            <button
              v-else-if="link.link_id"
              type="button"
              class="text-[10px] text-muted opacity-0 group-hover:opacity-100 hover:text-default px-1"
              title="Remove manual link"
              @click="unlink(link)"
            >
              ×
            </button>
          </div>
        </div>
      </li>
      <li v-if="!links.length" class="px-3 py-2 text-xs text-muted">No linked requests.</li>
    </ul>

    <EntitySearchPicker
      :open="requestPickerOpen"
      :items="pickerRequests"
      :get-key="(r: WorkspaceRequest) => r.id"
      :get-title="(r: WorkspaceRequest) => r.name"
      :get-subtitle="(r: WorkspaceRequest) => `${r.method} ${r.url}`"
      :search-keys="['name', 'url', 'method', 'source_operation_id']"
      placeholder="Search requests…"
      @close="requestPickerOpen = false"
      @select="onRequestSelected"
    >
      <template #item="{ item }">
        <div class="flex items-center gap-2">
          <MethodBadge :method="item.method" class="shrink-0 scale-90" />
          <div class="min-w-0">
            <div class="font-medium truncate">{{ item.name }}</div>
            <div class="text-xs text-muted truncate">{{ item.url }}</div>
          </div>
        </div>
      </template>
    </EntitySearchPicker>
  </aside>
</template>

<script setup lang="ts">
import type { DocRequestLinkItem } from '~/stores/docs'
import type { RequestItem } from '~/stores/collections'
import { buildWorkspaceRequestUrl } from '~/utils/docLinks'

type WorkspaceRequest = RequestItem & { collection_name?: string }

const props = defineProps<{
  workspaceId: string
  docId: string
}>()

const docsStore = useDocsStore()
const api = useApi()
const toast = useToast()
const links = ref<DocRequestLinkItem[]>([])
const loading = ref(false)
const requestPickerOpen = ref(false)
const workspaceRequests = ref<WorkspaceRequest[]>([])

const linkedRequestIds = computed(() => new Set(links.value.map(l => l.request_id)))

const pickerRequests = computed(() =>
  workspaceRequests.value.filter(r => !linkedRequestIds.value.has(r.id)),
)

function requestUrl(requestId: string) {
  return buildWorkspaceRequestUrl(props.workspaceId, requestId)
}

async function fetchLinks() {
  if (!props.docId) return
  loading.value = true
  try {
    links.value = await docsStore.fetchDocRequestLinks(props.workspaceId, props.docId)
  } catch {
    links.value = []
  } finally {
    loading.value = false
  }
}

async function fetchRequests() {
  try {
    workspaceRequests.value = await api.get<WorkspaceRequest[]>(
      `/api/workspaces/${props.workspaceId}/requests`,
    )
  } catch {
    workspaceRequests.value = []
  }
}

watch(() => [props.workspaceId, props.docId] as const, () => {
  fetchLinks()
}, { immediate: true })

watch(requestPickerOpen, (open) => {
  if (open && !workspaceRequests.value.length) fetchRequests()
})

async function onRequestSelected(req: WorkspaceRequest) {
  await docsStore.createDocLink(props.workspaceId, props.docId, { request_id: req.id })
  await fetchLinks()
  await docsStore.fetchGraph(props.workspaceId)
  toast.show(`Linked "${req.name}" successfully.`)
}

async function unlink(link: DocRequestLinkItem) {
  if (!link.link_id) return
  await docsStore.deleteDocLink(props.workspaceId, props.docId, link.link_id)
  await fetchLinks()
  await docsStore.fetchGraph(props.workspaceId)
}

async function accept(link: DocRequestLinkItem) {
  if (!link.suggestion_id) return
  await docsStore.acceptSuggestion(props.workspaceId, link.suggestion_id)
  await fetchLinks()
  await docsStore.fetchGraph(props.workspaceId)
  toast.show('Link accepted.')
}

async function reject(link: DocRequestLinkItem) {
  if (!link.suggestion_id) return
  await docsStore.rejectSuggestion(props.workspaceId, link.suggestion_id)
  await fetchLinks()
  toast.show('Suggestion dismissed.')
}
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}
.hover\:bg-surface-2:hover {
  background: var(--color-surface-2);
}
</style>
