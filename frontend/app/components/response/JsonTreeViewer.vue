<template>
  <div class="h-full min-h-0 overflow-auto p-3 text-xs ui-input-editor">
    <JsonTreeNode
      v-for="child in rootChildren"
      :key="child.path"
      :label="child.label"
      :value="child.value"
      :path="child.path"
      :depth="0"
    />
  </div>
</template>

<script setup lang="ts">
const props = defineProps<{ data: unknown }>()

const rootChildren = computed(() => {
  const value = props.data

  if (value === null || typeof value !== 'object') {
    return [{ label: null, value, path: '$' }]
  }

  if (Array.isArray(value)) {
    return value.map((item, index) => ({
      label: index,
      value: item,
      path: `[${index}]`,
    }))
  }

  return Object.entries(value).map(([key, entryValue]) => ({
    label: key,
    value: entryValue,
    path: key,
  }))
})
</script>
