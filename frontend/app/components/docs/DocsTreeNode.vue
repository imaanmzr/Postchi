<template>
  <div
    class="docs-tree-row flex items-center gap-1 w-full text-left text-xs cursor-pointer select-none transition-colors"
    :class="{
      'docs-tree-row--active': active,
      'docs-tree-row--folder': row.type === 'folder',
    }"
    :style="{ paddingLeft: `${row.depth * 12 + 8}px` }"
    @click="onClick"
    @contextmenu.prevent="emit('contextmenu', $event, row)"
  >
    <button
      v-if="row.type === 'folder'"
      type="button"
      class="docs-tree-chevron shrink-0 p-0.5 rounded hover:bg-surface-2"
      @click.stop="emit('toggle', row.path)"
    >
      <ChevronRight
        :size="14"
        class="transition-transform"
        :class="{ 'rotate-90': row.expanded }"
      />
    </button>
    <span v-else class="w-[18px] shrink-0" />
    <component
      :is="row.type === 'folder' ? Folder : FileText"
      :size="14"
      class="shrink-0 opacity-70"
    />
    <span
      class="truncate flex-1 min-w-0"
      :class="row.type === 'file' ? 'font-medium' : ''"
      v-html="labelHtml"
    />
  </div>
</template>

<script setup lang="ts">
import { ChevronRight, FileText, Folder } from 'lucide-vue-next'
import type { FlatTreeRow } from '~/utils/docsTree'
import { highlightName } from '~/utils/docsTree'

const props = defineProps<{
  row: FlatTreeRow
  active?: boolean
}>()

const emit = defineEmits<{
  click: [row: FlatTreeRow]
  toggle: [path: string]
  contextmenu: [event: MouseEvent, row: FlatTreeRow]
}>()

const labelHtml = computed(() => {
  if (props.row.type === 'file' && props.row.matchRanges?.length) {
    return highlightName(props.row.name, props.row.matchRanges)
  }
  return props.row.name.replace(/&/g, '&amp;').replace(/</g, '&lt;')
})

function onClick() {
  if (props.row.type === 'folder') {
    emit('toggle', props.row.path)
  } else {
    emit('click', props.row)
  }
}
</script>

<style scoped>
.docs-tree-row {
  height: 28px;
  color: var(--color-text-muted);
}
.docs-tree-row:hover {
  background: var(--color-surface-2);
  color: var(--color-text);
}
.docs-tree-row--active {
  background: var(--color-surface-2);
  color: var(--color-text);
}
.docs-tree-row :deep(.docs-tree-match) {
  background: color-mix(in srgb, var(--color-accent) 35%, transparent);
  color: var(--color-text);
  border-radius: 2px;
  padding: 0 1px;
}
</style>
