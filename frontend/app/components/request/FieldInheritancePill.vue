<template>
  <div v-if="request.template_id" class="mb-2">
    <button
      v-if="isOverridden"
      type="button"
      class="text-[10px] px-2 py-0.5 rounded"
      style="background: var(--method-patch); color: var(--color-bg)"
      @click="$emit('reset', field)"
    >
      Customized - reset to template
    </button>
    <span
      v-else
      class="text-[10px] px-2 py-0.5 rounded"
      style="background: var(--color-surface-2); color: var(--text-secondary)"
    >
      Inherited from template
    </span>
  </div>
</template>

<script setup lang="ts">
import type { RequestItem } from '~/stores/collections'

const props = defineProps<{ field: string; request: RequestItem }>()
defineEmits<{ reset: [field: string] }>()

const isOverridden = computed(() =>
  (props.request.overridden_fields || []).includes(props.field),
)
</script>
