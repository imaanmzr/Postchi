<template>
  <span
    class="inline-block px-1.5 py-0.5 rounded font-mono text-[10px] font-semibold tracking-tight"
    :style="{ color: methodColor(method), background: 'var(--color-surface-2)' }"
  >
    {{ method }}
  </span>
</template>

<script setup lang="ts">
import { methodColor } from '~/utils/methodColor'

defineProps<{ method: string }>()
</script>
