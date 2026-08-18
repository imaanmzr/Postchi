<template>
  <aside
    v-if="backlinks.length"
    class="shrink-0 border-l w-52 flex flex-col min-h-0"
    style="border-color: var(--color-border); background: var(--color-surface-1)"
  >
    <div class="px-3 py-2 border-b text-xs font-medium shrink-0" style="border-color: var(--color-border)">
      Backlinks
      <span class="text-muted font-normal">({{ backlinks.length }})</span>
    </div>
    <ul class="flex-1 min-h-0 overflow-auto ui-scroll-y py-1">
      <li v-for="link in backlinks" :key="link.slug">
        <button
          type="button"
          class="w-full text-left px-3 py-2 text-xs hover:bg-surface-2 transition truncate"
          @click="emit('select', link.slug)"
        >
          {{ link.title }}
        </button>
      </li>
    </ul>
  </aside>
</template>

<script setup lang="ts">
import type { DocGraph } from '~/stores/docs'
import type { DocSummary } from '~/utils/docsTree'

const props = defineProps<{
  activeSlug: string | null
  summaries: DocSummary[]
  graph: DocGraph | null
}>()

const emit = defineEmits<{
  select: [slug: string]
}>()

const backlinks = computed(() => {
  if (!props.activeSlug || !props.graph) return []
  const bySlug = new Map(props.summaries.map(s => [s.slug, s]))
  const found = new Map<string, { slug: string, title: string }>()
  for (const edge of props.graph.edges) {
    if (edge.type !== 'link' || edge.target !== props.activeSlug) continue
    const s = bySlug.get(edge.source)
    if (s) found.set(s.slug, { slug: s.slug, title: s.title })
  }
  return Array.from(found.values()).sort((a, b) => a.title.localeCompare(b.title))
})
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}
</style>
