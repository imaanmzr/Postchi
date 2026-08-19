<template>
  <div ref="containerEl" class="relative w-full h-full overflow-hidden" style="background: var(--color-bg)">
    <svg ref="svgEl" class="w-full h-full" />
    <div
      v-if="tooltip"
      class="absolute z-10 max-w-xs rounded-md border px-3 py-2 text-xs shadow-lg pointer-events-none"
      :style="{
        left: `${tooltip.x}px`,
        top: `${tooltip.y}px`,
        background: 'var(--color-surface-1)',
        borderColor: 'var(--color-border)',
      }"
    >
      <div class="font-medium mb-1">{{ tooltip.label }}</div>
      <div class="text-muted line-clamp-3">{{ tooltip.preview }}</div>
    </div>
    <p v-if="!props.nodes.length" class="absolute inset-0 flex items-center justify-center text-sm text-muted">
      No documentation to graph yet.
    </p>
  </div>
</template>

<script setup lang="ts">
import { drag } from 'd3-drag'
import { forceCenter, forceLink, forceManyBody, forceSimulation, type SimulationLinkDatum, type SimulationNodeDatum } from 'd3-force'
import { select } from 'd3-selection'
import { zoom } from 'd3-zoom'
import type { DocGraphEdge, DocGraphNode } from '~/stores/docs'
import type { DocSummary } from '~/utils/docsTree'

const props = defineProps<{
  nodes: DocGraphNode[]
  edges: DocGraphEdge[]
  activeSlug?: string | null
  summaries?: DocSummary[]
}>()

const emit = defineEmits<{
  select: [node: DocGraphNode]
}>()

type SimNode = DocGraphNode & SimulationNodeDatum
type SimLink = SimulationLinkDatum<SimNode> & { type: string }

const containerEl = ref<HTMLElement>()
const svgEl = ref<SVGSVGElement>()
let simulation: ReturnType<typeof forceSimulation<SimNode>> | null = null
let simNodes: SimNode[] = []
let simLinks: SimLink[] = []
let zoomBehavior: ReturnType<typeof zoom<SVGSVGElement, unknown>> | null = null
let currentTransform = { x: 0, y: 0, k: 1 }

const tooltip = ref<{ x: number, y: number, label: string, preview: string } | null>(null)

const previewBySlug = computed(() => {
  const map = new Map<string, string>()
  // summaries don't have content; tooltip shows title only fallback
  for (const s of props.summaries ?? []) {
    map.set(s.slug, s.title)
  }
  return map
})

const colorForType = (type: string) => {
  if (type === 'operation') return 'var(--color-syntax-number)'
  if (type === 'request') return 'var(--color-syntax-function)'
  return 'var(--color-accent)'
}

function buildSimData() {
  const nodeMap = new Map(props.nodes.map(n => [n.id, { ...n } as SimNode]))
  simNodes = Array.from(nodeMap.values())
  simLinks = props.edges
    .map(e => ({ source: e.source, target: e.target, type: e.type }))
    .filter(l => nodeMap.has(String(l.source)) && nodeMap.has(String(l.target))) as SimLink[]
}

function ensureSimulation(width: number, height: number) {
  if (!simulation) {
    simulation = forceSimulation(simNodes)
      .force('link', forceLink<SimNode, SimLink>(simLinks).id(d => d.id).distance(90))
      .force('charge', forceManyBody().strength(-220))
      .force('center', forceCenter(width / 2, height / 2))
      .velocityDecay(0.35)
      .alphaDecay(0.02)
      .on('tick', onTick)
    simulation.on('end', () => simulation?.stop())
  } else {
    simulation.nodes(simNodes)
    const linkForce = simulation.force('link') as ReturnType<typeof forceLink<SimNode, SimLink>>
    linkForce?.links(simLinks)
    simulation.alpha(0.3).restart()
  }
}

function onTick() {
  if (!svgEl.value) return
  const g = select(svgEl.value).select<SVGGElement>('g.graph-root')
  g.selectAll<SVGLineElement, SimLink>('line')
    .attr('x1', d => (d.source as SimNode).x ?? 0)
    .attr('y1', d => (d.source as SimNode).y ?? 0)
    .attr('x2', d => (d.target as SimNode).x ?? 0)
    .attr('y2', d => (d.target as SimNode).y ?? 0)
  g.selectAll<SVGGElement, SimNode>('g.node')
    .attr('transform', d => `translate(${d.x ?? 0},${d.y ?? 0})`)
}

function renderGraph() {
  if (!svgEl.value || !containerEl.value) return
  const width = containerEl.value.clientWidth || 800
  const height = containerEl.value.clientHeight || 600

  buildSimData()
  const svg = select(svgEl.value)
  svg.attr('viewBox', `0 0 ${width} ${height}`)

  let g = svg.select<SVGGElement>('g.graph-root')
  if (g.empty()) {
    g = svg.append('g').attr('class', 'graph-root')
    if (!zoomBehavior) {
      zoomBehavior = zoom<SVGSVGElement, unknown>()
        .scaleExtent([0.2, 4])
        .on('zoom', (event) => {
          currentTransform = event.transform
          g.attr('transform', event.transform)
        })
      svg.call(zoomBehavior)
      g.attr('transform', `translate(${currentTransform.x},${currentTransform.y}) scale(${currentTransform.k})`)
    }

    g.append('g').attr('class', 'links')
      .attr('stroke', 'var(--color-border)')
      .attr('stroke-opacity', 0.8)

    g.append('g').attr('class', 'nodes')
  }

  const linkSel = g.select('g.links')
    .selectAll<SVGLineElement, SimLink>('line')
    .data(simLinks, d => `${String(d.source)}->${String(d.target)}`)
    .join('line')
    .attr('stroke-width', d => (d.type === 'operation' || d.type === 'frontmatter') ? 1.5 : 1)
    .attr('stroke-dasharray', d => {
      if (d.type === 'suggested') return '2 4'
      if (d.type === 'operation' || d.type === 'frontmatter') return '4 3'
      return null
    })

  const nodeSel = g.select('g.nodes')
    .selectAll<SVGGElement, SimNode>('g.node')
    .data(simNodes, d => d.id)
    .join(
      enter => {
        const ng = enter.append('g')
          .attr('class', 'node')
          .attr('cursor', 'pointer')
          .call(drag<SVGGElement, SimNode>()
            .on('start', (event, d) => {
              if (!event.active) simulation?.alphaTarget(0.2).restart()
              d.fx = d.x
              d.fy = d.y
            })
            .on('drag', (event, d) => {
              d.fx = event.x
              d.fy = event.y
            })
            .on('end', (event, d) => {
              if (!event.active) simulation?.alphaTarget(0)
              d.fx = null
              d.fy = null
            }))
          .on('click', (_event, d) => {
            if (d.type === 'doc') emit('select', d)
          })
          .on('mouseenter', (event, d) => {
            if (d.type !== 'doc') return
            const rect = containerEl.value!.getBoundingClientRect()
            tooltip.value = {
              x: event.clientX - rect.left + 12,
              y: event.clientY - rect.top + 12,
              label: d.label,
              preview: previewBySlug.value.get(d.id) ?? d.label,
            }
          })
          .on('mouseleave', () => { tooltip.value = null })

        ng.append('circle')
          .attr('r', d => d.type === 'operation' ? 7 : 10)
          .attr('fill', d => colorForType(d.type))

        ng.append('text')
          .attr('x', 14)
          .attr('y', 4)
          .attr('font-size', 11)
          .attr('fill', 'var(--color-text)')

        return ng
      },
      update => update,
      exit => exit.remove(),
    )

  nodeSel.select('circle')
    .attr('fill', d => colorForType(d.type))
    .attr('stroke', d => d.id === props.activeSlug ? 'var(--color-text)' : 'transparent')
    .attr('stroke-width', 2)

  nodeSel.select('text')
    .text(d => d.label.length > 24 ? `${d.label.slice(0, 22)}…` : d.label)

  ensureSimulation(width, height)
  linkSel // used by join
}

watch(() => [props.nodes, props.edges, props.activeSlug], () => nextTick(renderGraph), { deep: true })

onMounted(() => {
  renderGraph()
  const ro = new ResizeObserver(() => {
    if (!containerEl.value || !simulation) return
    const width = containerEl.value.clientWidth || 800
    const height = containerEl.value.clientHeight || 600
    simulation.force('center', forceCenter(width / 2, height / 2))
    simulation.alpha(0.15).restart()
  })
  if (containerEl.value) ro.observe(containerEl.value)
  onUnmounted(() => {
    ro.disconnect()
    simulation?.stop()
  })
})
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}
</style>
