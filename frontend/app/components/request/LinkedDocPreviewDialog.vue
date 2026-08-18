<template>
  <Teleport to="body">
    <div
      v-if="open && doc"
      class="fixed inset-0 z-[90] flex items-center justify-center p-4"
      @click.self="emit('close')"
    >
      <div class="absolute inset-0 ui-overlay backdrop-blur-sm" @click="emit('close')" />
      <div
        class="relative z-10 flex-none w-[min(100%,48rem)] rounded-lg flex flex-col shadow-2xl overflow-hidden"
        style="background: var(--color-surface-1); border: 1px solid var(--color-border); max-height: 85vh"
      >
        <div
          class="flex items-center gap-3 px-4 py-3 border-b shrink-0"
          style="border-color: var(--color-border)"
        >
          <div class="min-w-0 flex-1">
            <h2 class="text-base font-semibold truncate">{{ doc.title }}</h2>
            <p v-if="doc.source_path" class="text-xs text-muted truncate mt-0.5">{{ doc.source_path }}</p>
          </div>
          <div class="flex items-center gap-1.5 shrink-0">
            <NuxtLink
              v-if="workspaceId"
              :to="docWorkspaceUrl"
              class="ui-btn ui-btn-ghost text-xs px-2.5 py-1.5"
              @click="emit('close')"
            >
              Open in docs
            </NuxtLink>
            <button
              type="button"
              class="ui-btn ui-btn-ghost text-xs px-2 py-1.5 opacity-70 hover:opacity-100"
              aria-label="Close preview"
              @click="emit('close')"
            >
              ×
            </button>
          </div>
        </div>
        <div class="flex-1 min-h-0 ui-scroll-y p-6" style="background: var(--color-bg)">
          <MarkdownViewer :content="doc.content_md" />
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import type { LinkedWorkspaceDoc } from '~/stores/docs'

const props = defineProps<{
  open: boolean
  doc: LinkedWorkspaceDoc | null
  workspaceId?: string
}>()

const emit = defineEmits<{ close: [] }>()

const docWorkspaceUrl = computed(() => {
  if (!props.workspaceId || !props.doc) return '#'
  return `/workspaces/${props.workspaceId}/docs/${encodeURIComponent(props.doc.slug)}`
})
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}
</style>
