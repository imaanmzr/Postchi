<template>
  <Teleport to="body">
    <Transition name="modal-fade">
      <div v-if="open" class="fixed inset-0 z-50 flex items-center justify-center p-4">
        <div class="absolute inset-0 ui-overlay backdrop-blur-sm" @click="$emit('update:open', false)" />
        <div
          class="relative z-10 w-full max-w-md rounded-lg p-5 shadow-md"
          style="background: var(--color-surface-1); border: 1px solid var(--color-border)"
        >
          <h3 v-if="title" class="text-lg font-semibold tracking-tight mb-2">{{ title }}</h3>
          <p class="text-sm mb-4 text-muted"><slot /></p>
          <div class="flex justify-end gap-2">
            <Button @click="$emit('update:open', false)">Cancel</Button>
            <Button
              variant="primary"
              :class="destructive ? '!bg-[var(--method-delete)]' : ''"
              @click="onConfirm"
            >
              {{ confirmLabel }}
            </Button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
defineProps<{
  open: boolean
  title?: string
  confirmLabel?: string
  destructive?: boolean
}>()
const emit = defineEmits<{ 'update:open': [value: boolean]; confirm: [] }>()

function onConfirm() {
  emit('update:open', false)
  emit('confirm')
}
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}
.modal-fade-enter-active,
.modal-fade-leave-active {
  transition: opacity var(--duration-normal) var(--ease-out);
}
.modal-fade-enter-active .relative,
.modal-fade-leave-active .relative {
  transition: transform var(--duration-normal) var(--ease-out), opacity var(--duration-normal) var(--ease-out);
}
.modal-fade-enter-from,
.modal-fade-leave-to {
  opacity: 0;
}
.modal-fade-enter-from .relative,
.modal-fade-leave-to .relative {
  transform: scale(0.96);
  opacity: 0;
}
</style>
