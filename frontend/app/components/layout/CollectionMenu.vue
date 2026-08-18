<template>
  <div ref="root" class="relative">
    <button
      type="button"
      class="ui-btn ui-btn-ghost text-xs px-2 inline-flex items-center justify-center"
      title="Collection menu"
      aria-label="Collection menu"
      @click.stop="toggle"
    >
      <MoreVertical :size="16" :stroke-width="2" aria-hidden="true" />
    </button>

    <div
      v-if="open"
      class="fixed z-50 py-1 rounded shadow-lg text-sm min-w-[180px] ui-context-menu"
      :style="{ left: menuPos.x + 'px', top: menuPos.y + 'px' }"
    >
      <button
        v-for="item in menuItems"
        :key="item.id"
        type="button"
        class="flex items-center gap-2 w-full text-left px-3 py-1.5 hover:bg-[var(--color-grid)] disabled:opacity-40"
        :disabled="item.disabled"
        @click="item.action()"
      >
        <component
          :is="item.icon"
          class="w-4 shrink-0 opacity-70"
          :size="14"
          :stroke-width="2"
          aria-hidden="true"
        />
        <span>{{ item.label }}</span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import {
  Clock,
  FileCode2,
  FolderCog,
  MoreVertical,
  Variable,
  type LucideIcon,
} from 'lucide-vue-next'

const props = defineProps<{
  hasCollection: boolean
}>()

const emit = defineEmits<{
  history: []
  variables: []
  openapi: []
  'collection-settings': []
}>()

const open = ref(false)
const root = ref<HTMLElement | null>(null)
const menuPos = ref({ x: 0, y: 0 })

const menuItems = computed((): Array<{
  id: string
  icon: LucideIcon
  label: string
  disabled: boolean
  action: () => void
}> => [
  { id: 'history', icon: Clock, label: 'History', disabled: false, action: () => select('history') },
  { id: 'variables', icon: Variable, label: 'Variables', disabled: !props.hasCollection, action: () => select('variables') },
  { id: 'openapi', icon: FileCode2, label: 'OpenAPI', disabled: false, action: () => select('openapi') },
  { id: 'settings', icon: FolderCog, label: 'Collection Settings', disabled: !props.hasCollection, action: () => select('collection-settings') },
])

function toggle(e: MouseEvent) {
  if (open.value) {
    open.value = false
    return
  }
  const rect = (e.currentTarget as HTMLElement).getBoundingClientRect()
  menuPos.value = { x: rect.right - 180, y: rect.bottom + 4 }
  open.value = true
}

function select(action: 'history' | 'variables' | 'openapi' | 'collection-settings') {
  open.value = false
  emit(action)
}

function onDocClick(e: MouseEvent) {
  if (!open.value) return
  if (root.value?.contains(e.target as Node)) return
  open.value = false
}

onMounted(() => document.addEventListener('click', onDocClick))
onUnmounted(() => document.removeEventListener('click', onDocClick))
</script>
