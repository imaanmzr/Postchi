<template>
  <Dialog
    :open="open"
    :title="title"
    :confirm-label="confirmLabel"
    :destructive="destructive"
    @update:open="$emit('update:open', $event)"
    @confirm="$emit('confirm')"
  >
    {{ message }}
  </Dialog>
</template>

<script setup lang="ts">
defineProps<{
  open: boolean
  title?: string
  message?: string
  confirmLabel?: string
  destructive?: boolean
}>()
defineEmits<{ 'update:open': [value: boolean]; confirm: [] }>()
</script>
