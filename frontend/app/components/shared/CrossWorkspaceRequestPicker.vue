<template>
  <EntitySearchPicker
    :open="open"
    :items="availableItems"
    :get-key="(r: LinkedRequest) => r.id"
    :get-title="(r: LinkedRequest) => r.name"
    :get-subtitle="linkedRequestSubtitle"
    :search-keys="['name', 'method', 'url', 'workspace_name']"
    placeholder="Search requests across workspaces…"
    :empty-label="loading ? 'Loading requests…' : 'No requests found'"
    @select="emit('select', $event)"
    @close="emit('close')"
  />
</template>

<script setup lang="ts">
import {
  fetchLinkableRequests,
  linkedRequestSubtitle,
  type LinkedRequest,
} from '~/utils/linkableRequests'

const props = defineProps<{
  open: boolean
  excludeIds?: string[]
}>()

const emit = defineEmits<{
  close: []
  select: [item: LinkedRequest]
}>()

const items = ref<LinkedRequest[]>([])
const loading = ref(false)

const availableItems = computed(() => {
  const excluded = new Set(props.excludeIds || [])
  return items.value.filter(r => !excluded.has(r.id))
})

watch(() => props.open, async (isOpen) => {
  if (!isOpen) return
  loading.value = true
  try {
    items.value = await fetchLinkableRequests()
  } finally {
    loading.value = false
  }
})
</script>
