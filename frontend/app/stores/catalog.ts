import { defineStore } from 'pinia'

export interface CatalogCollection {
  id: string
  name: string
  description: string
  request_count: number
  documented_count: number
}

export interface CatalogEndpoint {
  id: string
  collection_id: string
  collection_name: string
  name: string
  method: string
  url: string
  description: string
  tags: string[]
  response_codes: string[]
  source_spec_id?: string
  source_operation_id?: string
  api_doc: Record<string, unknown>
  docs_complete: boolean
}

export interface CatalogResponse {
  collections: CatalogCollection[]
  endpoints: CatalogEndpoint[]
}

export const useCatalogStore = defineStore('catalog', {
  state: () => ({
    data: null as CatalogResponse | null,
    loading: false,
    filters: {
      q: '',
      tag: '',
      method: '',
      undocumented: false,
      spec_id: '',
    },
  }),
  actions: {
    async fetchWorkspace(workspaceId: string, filters?: Partial<typeof this.filters>) {
      const api = useApi()
      this.loading = true
      const f = { ...this.filters, ...filters }
      const params = new URLSearchParams()
      if (f.q) params.set('q', f.q)
      if (f.tag) params.set('tag', f.tag)
      if (f.method) params.set('method', f.method)
      if (f.undocumented) params.set('undocumented', 'true')
      if (f.spec_id) params.set('spec_id', f.spec_id)
      const qs = params.toString()
      try {
        this.data = await api.get<CatalogResponse>(`/api/workspaces/${workspaceId}/catalog${qs ? `?${qs}` : ''}`)
      } finally {
        this.loading = false
      }
    },
    setFilters(filters: Partial<typeof this.filters>) {
      this.filters = { ...this.filters, ...filters }
    },
  },
})
