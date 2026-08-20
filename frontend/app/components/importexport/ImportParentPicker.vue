<template>
  <div class="space-y-2">
    <label class="text-sm block">Import destination</label>
    <Select v-model="mode">
      <option value="root">Workspace root (top level)</option>
      <option value="existing">Existing collection or folder</option>
      <option value="new">Create new collection</option>
    </Select>

    <div v-if="mode === 'existing'">
      <label class="text-xs block mb-1" style="color: var(--text-muted)">Parent collection or folder</label>
      <Select v-model="parentId">
        <option value="">Select destination…</option>
        <option v-for="opt in options" :key="opt.id" :value="opt.id">
          {{ indentLabel(opt.label, opt.depth) }}
        </option>
      </Select>
      <p v-if="!options.length" class="text-xs mt-1" style="color: var(--text-muted)">
        No collections yet. Choose workspace root or create a new collection.
      </p>
    </div>

    <div v-else-if="mode === 'new'">
      <label class="text-xs block mb-1" style="color: var(--text-muted)">New collection name</label>
      <Input v-model="newParentName" placeholder="e.g. Git Imports" />
      <p class="text-xs mt-1" style="color: var(--text-muted)">
        A new collection is created and imported folders are placed inside it.
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { Collection } from '~/stores/collections'
import {
  flattenCollectionOptions,
  indentLabel,
  loadImportParentChoice,
  saveImportParentChoice,
  type ImportParentChoice,
  type ImportParentMode,
} from '~/utils/importParent'

const props = defineProps<{
  workspaceId: string
  collections: Collection[]
}>()

const choice = defineModel<ImportParentChoice>({ required: true })

const mode = computed({
  get: () => choice.value.mode,
  set: (value: ImportParentMode) => {
    choice.value = { ...choice.value, mode: value }
  },
})

const parentId = computed({
  get: () => choice.value.parentId || '',
  set: (value: string) => {
    choice.value = { ...choice.value, parentId: value || undefined }
  },
})

const newParentName = computed({
  get: () => choice.value.newParentName || '',
  set: (value: string) => {
    choice.value = { ...choice.value, newParentName: value }
  },
})

const options = computed(() => flattenCollectionOptions(props.collections))

watch(
  () => props.workspaceId,
  (workspaceId) => {
    const saved = loadImportParentChoice(workspaceId)
    if (saved) choice.value = saved
  },
  { immediate: true },
)

watch(
  choice,
  (value) => {
    saveImportParentChoice(props.workspaceId, value)
  },
  { deep: true },
)
</script>
