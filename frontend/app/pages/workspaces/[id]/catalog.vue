<template>
  <div class="flex flex-col h-screen overflow-hidden">
    <header
      class="px-4 py-3 border-b shrink-0"
      style="border-color: var(--color-border); background: var(--color-surface-1)"
    >
      <div class="flex items-center gap-3 flex-wrap">
        <h1 class="font-semibold text-sm inline-flex items-center gap-1.5">
          <BookOpen :size="16" class="opacity-80 shrink-0" aria-hidden="true" />
          API Reference
        </h1>
        <ShareButton
          v-if="workspaceId"
          :workspace-id="workspaceId"
          kind="catalog"
          :source-id="workspaceId"
          :landing-request-id="selectedRequestId"
          default-title="Workspace API Catalog"
          label="Share API Docs"
          class="ml-auto"
        />
        <NuxtLink
          :to="`/workspaces/${workspaceId}`"
          class="text-xs text-muted hover:text-default transition inline-flex items-center gap-1.5"
        >
          <ArrowLeft :size="14" aria-hidden="true" />
          Back to workspace
        </NuxtLink>
      </div>
      <p class="text-xs text-muted mt-1.5 max-w-3xl">
        Browse every endpoint in one place, see documentation coverage, and edit team notes or linked docs
        without leaving this view. Use <strong class="font-medium text-default">Open in request editor</strong>
        when you need to send the request or change params, headers, and scripts.
      </p>
    </header>
    <CatalogBrowser
      class="flex-1 min-h-0"
      :workspace-id="workspaceId"
      :initial-endpoint-id="selectedRequestId"
      @endpoint-selected="onEndpointSelected"
    />
  </div>
</template>

<script setup lang="ts">
import { ArrowLeft, BookOpen } from 'lucide-vue-next'

const route = useRoute()
const router = useRouter()
const workspaceId = computed(() => route.params.id as string)
const selectedRequestId = ref(
  typeof route.query.request === 'string' ? route.query.request : '',
)

watch(() => route.query.request, (requestId) => {
  selectedRequestId.value = typeof requestId === 'string' ? requestId : ''
})

function onEndpointSelected(requestId: string) {
  selectedRequestId.value = requestId
  void router.replace({
    query: {
      ...route.query,
      request: requestId,
    },
  })
}
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}
.hover\:text-default:hover,
.text-default {
  color: var(--color-text);
}
</style>
