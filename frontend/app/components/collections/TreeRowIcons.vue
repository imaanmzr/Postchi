<template>
  <GripVertical
    v-if="showGrip"
    class="tree-drag-handle"
    :size="12"
    :stroke-width="2"
    aria-hidden="true"
  />
  <button
    v-if="showChevron"
    type="button"
    class="tree-chevron-btn"
    :aria-expanded="expanded"
    :aria-label="expanded ? 'Collapse folder' : 'Expand folder'"
    @click.stop="$emit('toggle')"
  >
    <ChevronRight class="tree-chevron" :class="{ 'tree-chevron--open': expanded }" :size="14" :stroke-width="2" />
  </button>
  <component
    :is="expanded ? FolderOpen : Folder"
    v-if="showFolder"
    class="tree-folder-icon"
    :size="14"
    :stroke-width="2"
    aria-hidden="true"
  />
</template>

<script setup lang="ts">
import { ChevronRight, Folder, FolderOpen, GripVertical } from 'lucide-vue-next'

withDefaults(defineProps<{
  showGrip?: boolean
  showChevron?: boolean
  showFolder?: boolean
  expanded?: boolean
}>(), {
  showGrip: true,
  showChevron: false,
  showFolder: false,
  expanded: false,
})

defineEmits<{ toggle: [] }>()
</script>
