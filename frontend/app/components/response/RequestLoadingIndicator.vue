<template>
  <div class="h-full flex flex-col items-center justify-center gap-3 text-sm">
    <div
      class="loading-spinner"
      role="status"
      aria-label="Sending request"
    />
    <p class="text-muted font-mono text-xs tabular-nums">{{ elapsedMs }}ms</p>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  elapsedMs: number
}>()
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}

.loading-spinner {
  width: 2rem;
  height: 2rem;
  border: 2px solid transparent;
  border-top-color: var(--color-text);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
