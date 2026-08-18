<template>
  <select
    :value="modelValue"
    :class="cn('ui-input w-full', $attrs.class as string)"
    v-bind="$attrs"
    @change="$emit('update:modelValue', ($event.target as HTMLSelectElement).value)"
  >
    <slot />
  </select>
</template>

<script setup lang="ts">
import { cn } from '~/lib/utils'

defineProps<{ modelValue?: string }>()
defineEmits<{ 'update:modelValue': [value: string] }>()
</script>
