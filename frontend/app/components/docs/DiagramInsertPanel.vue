<template>
  <Teleport to="body">
    <div
      v-if="open"
      class="fixed inset-0 z-[100] flex items-start justify-center pt-[12vh] px-4"
      @click.self="emit('close')"
    >
      <div
        class="w-full max-w-lg rounded-lg border shadow-2xl overflow-hidden flex flex-col max-h-[70vh]"
        style="background: var(--color-surface-1); border-color: var(--color-border)"
      >
        <div class="px-4 py-3 border-b shrink-0" style="border-color: var(--color-border)">
          <h2 class="text-sm font-semibold mb-2">Link a user story</h2>
          <Input v-model="query" placeholder="Search stories…" class="text-sm" @keydown.escape="emit('close')" />
        </div>

        <ul class="flex-1 overflow-y-auto py-1 min-h-0">
          <li
            v-for="item in filtered"
            :key="item.slug"
            class="px-4 py-2.5 flex items-center gap-3 hover:bg-surface-2 transition cursor-pointer"
            @click="emit('insert', item)"
          >
            <div class="min-w-0 flex-1">
              <div class="text-sm font-medium truncate">{{ item.title }}</div>
              <div class="text-xs text-muted font-mono truncate">{{ item.slug }}</div>
            </div>
            <Button class="text-xs shrink-0" @click.stop="emit('insert', item)">Insert</Button>
          </li>
          <li v-if="!filtered.length && !creating" class="px-4 py-6 text-center text-sm text-muted">
            {{ diagrams.length ? 'No matching stories.' : 'No stories yet — create one below.' }}
          </li>
        </ul>

        <div class="px-4 py-3 border-t shrink-0 flex gap-2" style="border-color: var(--color-border)">
          <Input v-model="newTitle" placeholder="New story title" class="text-sm flex-1" />
          <Button variant="primary" class="text-xs shrink-0" :disabled="creating || !newTitle.trim()" @click="createAndInsert">
            {{ creating ? 'Creating…' : 'Create & insert' }}
          </Button>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import type { DiagramSummary } from '~/stores/diagrams'

const props = defineProps<{
  open: boolean
  workspaceId: string
  diagrams: DiagramSummary[]
}>()

const emit = defineEmits<{
  close: []
  insert: [diagram: DiagramSummary]
}>()

const diagramsStore = useDiagramsStore()
const query = ref('')
const newTitle = ref('')
const creating = ref(false)

const filtered = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return props.diagrams
  return props.diagrams.filter(d =>
    d.title.toLowerCase().includes(q) || d.slug.toLowerCase().includes(q),
  )
})

watch(() => props.open, (isOpen) => {
  if (isOpen) {
    query.value = ''
    newTitle.value = ''
  }
})

async function createAndInsert() {
  const title = newTitle.value.trim()
  if (!title || creating.value) return
  creating.value = true
  try {
    const diagram = await diagramsStore.createDiagram(props.workspaceId, title)
    emit('insert', {
      id: diagram.id,
      workspace_id: diagram.workspace_id,
      slug: diagram.slug,
      title: diagram.title,
      updated_at: diagram.updated_at,
    })
    emit('close')
  } finally {
    creating.value = false
  }
}
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}
.hover\:bg-surface-2:hover {
  background: var(--color-surface-2);
}
</style>
