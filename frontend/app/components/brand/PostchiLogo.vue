<template>
  <span
    class="postchi-logo inline-flex items-center shrink-0"
    :style="{ height: sizeValue }"
    v-html="markup"
  />
</template>

<script setup lang="ts">
import markSvg from '~/assets/brand/logo-mark.svg?raw'
import wordmarkSvg from '~/assets/brand/logo-wordmark.svg?raw'

const props = withDefaults(defineProps<{
  variant?: 'wordmark' | 'mark'
  height?: number | string
  title?: string
}>(), {
  variant: 'wordmark',
  height: 28,
  title: 'Postchi',
})

const markup = computed(() => {
  const svg = props.variant === 'mark' ? markSvg : wordmarkSvg
  return svg.replace('aria-label="Postchi"', `aria-label="${props.title}"`)
})

const sizeValue = computed(() =>
  typeof props.height === 'number' ? `${props.height}px` : props.height,
)
</script>

<style scoped>
.postchi-logo :deep(svg) {
  display: block;
  height: 100%;
  width: auto;
}
</style>
