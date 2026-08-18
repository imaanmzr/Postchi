<template>
  <div v-if="content" class="space-y-2">
    <div v-for="(media, mt) in content" :key="mt">
      <div class="text-xs font-mono text-muted mb-1">{{ mt }}</div>
      <pre v-if="media.schema" class="text-xs p-2 rounded overflow-auto" style="background: var(--color-surface-2)">{{ JSON.stringify(media.schema, null, 2) }}</pre>
      <pre v-if="media.example" class="text-xs p-2 rounded overflow-auto mt-1" style="background: var(--color-surface-2)">{{ JSON.stringify(media.example, null, 2) }}</pre>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{ content?: Record<string, { schema?: unknown; example?: unknown }> | null }>()
</script>
