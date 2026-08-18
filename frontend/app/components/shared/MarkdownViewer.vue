<template>
  <div
    ref="rootEl"
    class="markdown-viewer prose prose-sm max-w-none text-sm docs-prose"
    style="color: var(--color-text)"
    v-html="html"
    @click="onClick"
  />
</template>

<script setup lang="ts">
import { renderMarkdown, resolveDocSlug } from '~/utils/markdown'

const props = defineProps<{
  content: string
  docSlugs?: string[]
  docTitles?: Record<string, string>
}>()

const emit = defineEmits<{
  navigate: [slug: string]
}>()

const slugSet = computed(() => new Set(props.docSlugs || []))
const titleMap = computed(() => {
  const map = new Map<string, string>()
  for (const [slug, title] of Object.entries(props.docTitles || {})) {
    map.set(title.toLowerCase(), slug)
  }
  return map
})

const html = computed(() => renderMarkdown(props.content || '', {
  resolveLink: target => resolveDocSlug(target, slugSet.value, titleMap.value) ?? target,
}))

function onClick(e: MouseEvent) {
  const el = (e.target as HTMLElement).closest('a.wikilink') as HTMLAnchorElement | null
  if (!el) return
  e.preventDefault()
  const slug = el.dataset.docSlug
  if (slug) emit('navigate', slug)
}
</script>

<style scoped>
.markdown-viewer :deep(h1),
.markdown-viewer :deep(h2),
.markdown-viewer :deep(h3),
.markdown-viewer :deep(h4) {
  font-weight: 600;
  margin-top: 1.25em;
  margin-bottom: 0.5em;
  line-height: 1.3;
}
.markdown-viewer :deep(h1) { font-size: 1.5rem; }
.markdown-viewer :deep(h2) { font-size: 1.25rem; }
.markdown-viewer :deep(h3) { font-size: 1.1rem; }
.markdown-viewer :deep(p) {
  margin-bottom: 0.85em;
  line-height: 1.65;
}
.markdown-viewer :deep(ul),
.markdown-viewer :deep(ol) {
  margin-bottom: 0.85em;
  padding-left: 1.5em;
}
.markdown-viewer :deep(li) {
  margin-bottom: 0.35em;
}
.markdown-viewer :deep(li.task-list-item) {
  list-style: none;
  margin-left: -1.5em;
}
.markdown-viewer :deep(code) {
  font-family: var(--font-mono, monospace);
  font-size: 0.85em;
  padding: 0.1em 0.35em;
  border-radius: 4px;
  background: var(--color-surface-2);
}
.markdown-viewer :deep(pre) {
  padding: 0.85rem 1rem;
  border-radius: 8px;
  overflow-x: auto;
  background: var(--color-surface-2);
  margin-bottom: 1em;
  border: 1px solid var(--color-border);
}
.markdown-viewer :deep(pre code) {
  padding: 0;
  background: transparent;
}
.markdown-viewer :deep(table) {
  width: 100%;
  border-collapse: collapse;
  margin-bottom: 1em;
  font-size: 0.9em;
}
.markdown-viewer :deep(th),
.markdown-viewer :deep(td) {
  border: 1px solid var(--color-border);
  padding: 0.45em 0.65em;
  text-align: left;
}
.markdown-viewer :deep(th) {
  background: var(--color-surface-2);
  font-weight: 600;
}
.markdown-viewer :deep(blockquote) {
  border-left: 3px solid var(--color-accent);
  padding-left: 1em;
  margin: 0 0 1em;
  color: var(--color-text-muted);
}
.markdown-viewer :deep(a.wikilink) {
  color: var(--color-accent);
  text-decoration: none;
  border-bottom: 1px dashed color-mix(in srgb, var(--color-accent) 50%, transparent);
}
.markdown-viewer :deep(a.wikilink:hover) {
  border-bottom-style: solid;
}
.markdown-viewer :deep(hr) {
  border: none;
  border-top: 1px solid var(--color-border);
  margin: 1.5em 0;
}
.docs-prose {
  max-width: 72ch;
}
</style>
