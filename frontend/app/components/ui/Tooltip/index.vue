<template>
  <div class="relative inline-block">
    <div @mouseenter="show = true" @mouseleave="show = false">
      <slot />
    </div>
    <Transition name="tooltip-fade">
      <div
        v-if="show"
        class="absolute z-50 px-2 py-1 text-xs rounded-md shadow-md whitespace-nowrap -top-8 left-1/2 -translate-x-1/2"
        style="background: var(--color-surface-2); color: var(--color-text); border: 1px solid var(--color-border)"
      >
        {{ text }}
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
defineProps<{ text: string }>()
const show = ref(false)
</script>

<style scoped>
.tooltip-fade-enter-active,
.tooltip-fade-leave-active {
  transition: opacity var(--duration-fast) var(--ease-out);
}
.tooltip-fade-enter-from,
.tooltip-fade-leave-to {
  opacity: 0;
}
</style>
