<template>
  <div>
    <div class="flex border-b gap-1" style="border-color: var(--color-border)">
      <button
        v-for="tab in tabs"
        :key="tab.id"
        type="button"
        class="px-4 py-2 text-xs font-medium tracking-tight transition border-b-2 -mb-px"
        :class="modelValue === tab.id ? 'ui-subtab-active' : 'border-transparent text-muted hover:text-default'"
        @click="$emit('update:modelValue', tab.id)"
      >
        {{ tab.label }}
        <span v-if="tab.badge" class="ui-badge ml-1">{{ tab.badge }}</span>
      </button>
    </div>
    <div class="pt-3">
      <slot :name="modelValue" />
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
export interface TabItem { id: string; label: string; badge?: number | string }

defineProps<{ tabs: TabItem[]; modelValue: string }>()
defineEmits<{ 'update:modelValue': [value: string] }>()
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}
.hover\:text-default:hover {
  color: var(--color-text);
}
</style>
