<template>
  <div class="space-y-1">
    <div
      class="progress-track"
      role="progressbar"
      :aria-label="label || 'In progress'"
      aria-busy="true"
    >
      <div class="progress-bar" />
    </div>
    <p v-if="label" class="progress-label">{{ label }}</p>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  label?: string
}>()
</script>

<style scoped>
.progress-track {
  height: 3px;
  border-radius: 9999px;
  overflow: hidden;
  background: var(--color-surface-2);
}

.progress-bar {
  height: 100%;
  width: 40%;
  border-radius: 9999px;
  background: var(--color-accent);
  animation: progress-indeterminate 1.2s ease-in-out infinite;
}

.progress-label {
  font-size: 0.75rem;
  line-height: 1rem;
  color: var(--color-text-muted);
}

@keyframes progress-indeterminate {
  0% {
    transform: translateX(-100%);
  }

  100% {
    transform: translateX(350%);
  }
}
</style>
