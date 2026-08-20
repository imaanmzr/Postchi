<template>
  <span class="inline-flex">
    <Button
      :class="buttonClass"
      :variant="variant"
      :title="title"
      @click.stop="open = true"
    >
      <slot>
        <Share2 :size="14" :stroke-width="2" aria-hidden="true" />
        <span>{{ label }}</span>
      </slot>
    </Button>
    <ShareDialog
      :open="open"
      :workspace-id="workspaceId"
      :kind="kind"
      :source-id="sourceId"
      :landing-request-id="landingRequestId"
      :default-title="defaultTitle"
      @close="open = false"
    />
  </span>
</template>

<script setup lang="ts">
import { Share2 } from 'lucide-vue-next'

withDefaults(defineProps<{
  workspaceId: string
  kind: 'request' | 'history' | 'catalog'
  sourceId: string
  landingRequestId?: string
  defaultTitle?: string
  label?: string
  title?: string
  buttonClass?: string
  variant?: 'primary' | 'default'
}>(), {
  label: 'Share',
  title: 'Share with teammates',
  buttonClass: 'text-xs inline-flex items-center gap-1.5',
  variant: 'default',
})

const open = ref(false)
</script>
