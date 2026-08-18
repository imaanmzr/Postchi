<template>
  <button
    :type="type"
    :class="cn('ui-btn', variant === 'primary' ? 'ui-btn-primary' : 'ui-btn-ghost', $attrs.class as string)"
    v-bind="$attrs"
  >
    <slot />
  </button>
</template>

<script setup lang="ts">
import { cn } from '~/lib/utils'

withDefaults(defineProps<{ variant?: 'primary' | 'ghost'; type?: 'button' | 'submit' }>(), {
  variant: 'ghost',
  type: 'button',
})
</script>
