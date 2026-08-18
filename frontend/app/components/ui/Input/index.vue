<template>
  <input
    :value="modelValue"
    :type="type"
    :placeholder="placeholder"
    :class="cn('ui-input w-full', $attrs.class as string)"
    v-bind="$attrs"
    @input="$emit('update:modelValue', ($event.target as HTMLInputElement).value)"
  />
</template>

<script setup lang="ts">
import { cn } from '~/lib/utils'

defineProps<{ modelValue?: string; type?: string; placeholder?: string }>()
defineEmits<{ 'update:modelValue': [value: string] }>()
</script>
