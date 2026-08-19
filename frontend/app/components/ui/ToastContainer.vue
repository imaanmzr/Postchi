<template>
  <div
    class="fixed right-4 bottom-4 z-[120] flex w-[min(24rem,calc(100vw-2rem))] flex-col gap-2"
    aria-live="polite"
  >
    <TransitionGroup name="toast">
      <div
        v-for="toast in toasts"
        :key="toast.id"
        class="flex items-start gap-2 rounded-md border px-3 py-2.5 text-sm shadow-lg"
        :class="toast.kind === 'success' ? 'toast-success' : 'toast-error'"
        role="status"
      >
        <CheckCircle v-if="toast.kind === 'success'" :size="16" class="mt-0.5 shrink-0" aria-hidden="true" />
        <CircleAlert v-else :size="16" class="mt-0.5 shrink-0" aria-hidden="true" />
        <span class="min-w-0 flex-1">{{ toast.message }}</span>
        <button
          type="button"
          class="shrink-0 opacity-70 transition hover:opacity-100"
          aria-label="Dismiss notification"
          @click="dismiss(toast.id)"
        >
          ×
        </button>
      </div>
    </TransitionGroup>
  </div>
</template>

<script setup lang="ts">
import { CheckCircle, CircleAlert } from 'lucide-vue-next'

const { toasts, dismiss } = useToast()
</script>

<style scoped>
.toast-success {
  background: color-mix(in srgb, var(--color-success) 14%, var(--color-surface-1));
  border-color: color-mix(in srgb, var(--color-success) 45%, var(--color-border));
  color: var(--color-text);
}

.toast-error {
  background: color-mix(in srgb, var(--color-danger) 14%, var(--color-surface-1));
  border-color: color-mix(in srgb, var(--color-danger) 45%, var(--color-border));
  color: var(--color-text);
}

.toast-enter-active,
.toast-leave-active {
  transition: opacity 160ms ease, transform 160ms ease;
}

.toast-enter-from,
.toast-leave-to {
  opacity: 0;
  transform: translateY(0.5rem);
}
</style>
