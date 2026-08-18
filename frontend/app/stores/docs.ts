import { defineStore } from 'pinia'
import type { DocSummary } from '~/utils/docsTree'

export interface WorkspaceDoc extends DocSummary {
  content_md: string
  linked_operation_ids: string[]
}

export interface DocGraphNode {
  id: string
  label: string
  type: 'doc' | 'operation' | 'request'
}

export interface DocGraphEdge {
  source: string
  target: string
  type: 'link' | 'operation' | 'manual'
}

export interface DocLinkItem {
  id: string
  request_id: string
  request_name: string
  method: string
  url: string
  source_operation_id: string
  collection_name: string
}

export interface LinkedWorkspaceDoc {
  id: string
  slug: string
  title: string
  content_md: string
  link_sources: ('frontmatter' | 'manual')[]
  link_id?: string
}

export interface DocsBundle {
  api_doc: Record<string, unknown>
  description: string
  linked_workspace_docs: LinkedWorkspaceDoc[]
}

export interface DocGraph {
  nodes: DocGraphNode[]
  edges: DocGraphEdge[]
}

export const useDocsStore = defineStore('docs', {
  state: () => ({
    summaries: [] as DocSummary[],
    /** Plain object for Vue/Pinia reactivity (Map mutations are not tracked reliably). */
    contentBySlug: {} as Record<string, WorkspaceDoc>,
    graph: null as DocGraph | null,
    loading: false,
    loadingDoc: false,
    loadingDocCount: 0,
    saving: false,
    error: null as string | null,
    search: '',
  }),
  getters: {
    docBySlug: (state) => (slug: string) => state.contentBySlug[slug] ?? null,
    summaryBySlug: (state) => (slug: string) => state.summaries.find(d => d.slug === slug) ?? null,
    docSlugs(state): string[] {
      return state.summaries.map(d => d.slug)
    },
    docTitles(state): Record<string, string> {
      return Object.fromEntries(state.summaries.map(d => [d.slug, d.title]))
    },
  },
  actions: {
    async fetchWorkspace(workspaceId: string) {
      const api = useApi()
      this.loading = true
      this.error = null
      try {
        this.summaries = await api.get<DocSummary[]>(
          `/api/workspaces/${workspaceId}/workspace-docs?summary=1`,
        )
      } catch (e: unknown) {
        this.error = e instanceof Error ? e.message : 'Failed to load docs'
        throw e
      } finally {
        this.loading = false
      }
    },
    async fetchDoc(workspaceId: string, slug: string, force = false): Promise<WorkspaceDoc | null> {
      if (!force && this.contentBySlug[slug]) {
        return this.contentBySlug[slug]!
      }
      const api = useApi()
      this.loadingDocCount++
      this.loadingDoc = true
      this.error = null
      try {
        const doc = await api.get<WorkspaceDoc>(
          `/api/workspaces/${workspaceId}/workspace-docs/${encodeURIComponent(slug)}`,
        )
        this.contentBySlug = { ...this.contentBySlug, [slug]: doc }
        const summaryIdx = this.summaries.findIndex(d => d.slug === slug)
        if (summaryIdx >= 0) {
          this.summaries[summaryIdx] = {
            id: doc.id,
            workspace_id: doc.workspace_id,
            slug: doc.slug,
            title: doc.title,
            source_path: doc.source_path,
            is_local: doc.is_local,
            updated_at: doc.updated_at,
          }
        }
        return doc
      } catch (e: unknown) {
        this.error = e instanceof Error ? e.message : 'Failed to load document'
        return null
      } finally {
        this.loadingDocCount = Math.max(0, this.loadingDocCount - 1)
        this.loadingDoc = this.loadingDocCount > 0
      }
    },
    async createLocalDoc(workspaceId: string, sourcePath: string, title?: string) {
      const api = useApi()
      const doc = await api.post<WorkspaceDoc>(`/api/workspaces/${workspaceId}/workspace-docs`, {
        source_path: sourcePath,
        title: title || undefined,
      })
      this.summaries.push({
        id: doc.id,
        workspace_id: doc.workspace_id,
        slug: doc.slug,
        title: doc.title,
        source_path: doc.source_path,
        is_local: doc.is_local,
        updated_at: doc.updated_at,
      })
      this.contentBySlug = { ...this.contentBySlug, [doc.slug]: doc }
      return doc
    },
    async fetchGraph(workspaceId: string) {
      const api = useApi()
      this.graph = await api.get<DocGraph>(`/api/workspaces/${workspaceId}/doc-graph`)
    },
    async updateDoc(workspaceId: string, slug: string, patch: { title?: string, content_md?: string }) {
      const api = useApi()
      this.saving = true
      try {
        const updated = await api.patch<WorkspaceDoc>(
          `/api/workspaces/${workspaceId}/workspace-docs/${encodeURIComponent(slug)}`,
          patch,
        )
        this.contentBySlug = { ...this.contentBySlug, [slug]: updated }
        const idx = this.summaries.findIndex(d => d.slug === slug)
        if (idx >= 0) {
          this.summaries[idx] = {
            ...this.summaries[idx]!,
            title: updated.title,
            updated_at: updated.updated_at,
          }
        }
        return updated
      } finally {
        this.saving = false
      }
    },
    invalidateCache(slug?: string) {
      if (!slug) {
        this.contentBySlug = {}
        return
      }
      const next = { ...this.contentBySlug }
      delete next[slug]
      this.contentBySlug = next
    },
    async fetchDocLinks(workspaceId: string, docId: string) {
      const api = useApi()
      return api.get<DocLinkItem[]>(
        `/api/workspaces/${workspaceId}/workspace-docs/${docId}/links`,
      )
    },
    async createDocLink(workspaceId: string, docId: string, body: { request_id?: string, operation_id?: string }) {
      const api = useApi()
      return api.post<DocLinkItem | DocLinkItem[]>(
        `/api/workspaces/${workspaceId}/workspace-docs/${docId}/links`,
        body,
      )
    },
    async deleteDocLink(workspaceId: string, docId: string, linkId: string) {
      const api = useApi()
      return api.delete(`/api/workspaces/${workspaceId}/workspace-docs/${docId}/links/${linkId}`)
    },
  },
})
