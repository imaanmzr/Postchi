<template>
  <section class="h-full flex flex-col min-h-0 overflow-hidden" style="background: var(--color-surface-1)">
    <div
      class="px-3 py-2 border-b flex items-center justify-between shrink-0 gap-2"
      style="border-color: var(--color-border)"
    >
      <span class="text-xs font-medium">Linked requests</span>
      <Button class="text-xs shrink-0" @click="emit('link')">Link request</Button>
    </div>
    <ul v-if="requests.length" class="flex-1 overflow-y-auto p-2 space-y-2">
      <li
        v-for="req in requests"
        :key="req.id"
        class="p-2 rounded border text-xs"
        style="border-color: var(--color-border); background: var(--color-bg)"
      >
        <div class="flex items-start gap-2">
          <span
            class="text-[10px] font-mono font-semibold uppercase px-1.5 py-0.5 rounded shrink-0"
            :style="{ color: `var(--method-${req.method.toLowerCase()})` }"
          >{{ req.method }}</span>
          <div class="min-w-0 flex-1">
            <NuxtLink
              :to="buildWorkspaceRequestUrl(req.workspace_id, req.id)"
              class="font-medium truncate block hover:underline"
              :title="req.name"
            >
              {{ req.name }}
            </NuxtLink>
            <div class="text-[10px] text-muted truncate">{{ req.workspace_name }}</div>
            <div class="text-[10px] text-muted truncate font-mono">{{ req.url }}</div>
          </div>
          <button
            type="button"
            class="text-[10px] text-muted hover:text-default shrink-0"
            @click="emit('unlink', req.id)"
          >
            Remove
          </button>
        </div>
      </li>
    </ul>
    <p v-else class="p-3 text-xs text-muted leading-relaxed">
      Link API requests from any workspace you can access.
    </p>
  </section>
</template>

<script setup lang="ts">
import { buildWorkspaceRequestUrl } from '~/utils/docLinks'
import type { LinkedRequest } from '~/utils/linkableRequests'

defineProps<{
  requests: LinkedRequest[]
}>()

const emit = defineEmits<{
  link: []
  unlink: [requestId: string]
}>()
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}
</style>
