import { defineStore } from 'pinia'
import type { LinkedRequest } from '~/utils/linkableRequests'

export interface DiagramSummary {
  id: string
  workspace_id: string
  slug: string
  title: string
  updated_at: string
}

export interface Diagram extends DiagramSummary {
  content?: Record<string, unknown>
  requests?: LinkedRequest[]
}

export const useDiagramsStore = defineStore('diagrams', {
  state: () => ({
    summaries: [] as DiagramSummary[],
    current: null as Diagram | null,
    loading: false,
    error: '',
  }),
  actions: {
    async fetchDiagrams(workspaceId: string) {
      const api = useApi()
      this.loading = true
      this.error = ''
      try {
        this.summaries = await api.get<DiagramSummary[]>(`/api/workspaces/${workspaceId}/diagrams`)
      } catch (e: unknown) {
        this.error = e instanceof Error ? e.message : 'Failed to load diagrams'
        throw e
      } finally {
        this.loading = false
      }
    },
    async fetchDiagram(workspaceId: string, slug: string) {
      const api = useApi()
      this.error = ''
      const diagram = await api.get<Diagram>(`/api/workspaces/${workspaceId}/diagrams/${encodeURIComponent(slug)}`)
      this.current = diagram
      const idx = this.summaries.findIndex(d => d.slug === slug)
      if (idx >= 0) {
        this.summaries[idx] = { ...this.summaries[idx], title: diagram.title, updated_at: diagram.updated_at }
      }
      return diagram
    },
    async createDiagram(workspaceId: string, title: string, slug?: string) {
      const api = useApi()
      const diagram = await api.post<Diagram>(`/api/workspaces/${workspaceId}/diagrams`, { title, slug })
      this.summaries.unshift({
        id: diagram.id,
        workspace_id: diagram.workspace_id,
        slug: diagram.slug,
        title: diagram.title,
        updated_at: diagram.updated_at,
      })
      this.current = diagram
      return diagram
    },
    async updateDiagram(workspaceId: string, slug: string, data: { title?: string, content?: Record<string, unknown> }) {
      const api = useApi()
      const diagram = await api.patch<Diagram>(`/api/workspaces/${workspaceId}/diagrams/${encodeURIComponent(slug)}`, data)
      const prevRequests = this.current?.slug === slug ? this.current.requests : undefined
      this.current = { ...diagram, requests: prevRequests ?? diagram.requests }
      const idx = this.summaries.findIndex(d => d.slug === slug)
      if (idx >= 0) {
        this.summaries[idx] = {
          id: diagram.id,
          workspace_id: diagram.workspace_id,
          slug: diagram.slug,
          title: diagram.title,
          updated_at: diagram.updated_at,
        }
      }
      return this.current
    },
    async linkRequest(workspaceId: string, slug: string, requestId: string) {
      const api = useApi()
      await api.post(`/api/workspaces/${workspaceId}/diagrams/${encodeURIComponent(slug)}/requests/${requestId}`)
      return this.fetchDiagram(workspaceId, slug)
    },
    async unlinkRequest(workspaceId: string, slug: string, requestId: string) {
      const api = useApi()
      await api.delete(`/api/workspaces/${workspaceId}/diagrams/${encodeURIComponent(slug)}/requests/${requestId}`)
      return this.fetchDiagram(workspaceId, slug)
    },
    async deleteDiagram(workspaceId: string, slug: string) {
      const api = useApi()
      await api.delete(`/api/workspaces/${workspaceId}/diagrams/${encodeURIComponent(slug)}`)
      this.summaries = this.summaries.filter(d => d.slug !== slug)
      if (this.current?.slug === slug) this.current = null
    },
    clearCurrent() {
      this.current = null
    },
  },
})
