import { defineStore } from 'pinia'

export interface Share {
  id: string
  workspace_id: string
  kind: 'request' | 'history' | 'catalog'
  source_id: string
  token: string
  title: string
  snapshot?: Record<string, unknown>
  visibility: 'workspace' | 'link'
  expires_at?: string
  created_by: string
  created_at?: string
  share_url?: string
}

export const useSharesStore = defineStore('shares', {
  state: () => ({
    shares: [] as Share[],
  }),
  actions: {
    async list(workspaceId: string) {
      const api = useApi()
      this.shares = await api.get<Share[]>(`/api/workspaces/${workspaceId}/shares`)
      return this.shares
    },
    async create(payload: {
      kind: 'request' | 'history' | 'catalog'
      source_id: string
      workspace_id: string
      title?: string
      visibility?: string
      ttl_hours?: number
    }) {
      const api = useApi()
      const share = await api.post<Share>('/api/shares', payload)
      this.shares.unshift(share)
      return share
    },
    async revoke(id: string) {
      const api = useApi()
      await api.delete(`/api/shares/${id}`)
      this.shares = this.shares.filter(s => s.id !== id)
    },
    async fetchByToken(token: string) {
      const api = useApi()
      return api.get<Share>(`/api/shares/${token}`)
    },
    async importShare(token: string, payload: { workspace_id: string; collection_id: string; target_environment_id?: string }) {
      const api = useApi()
      return api.post<{ id: string; request_id: string }>(`/api/shares/${token}/import`, payload)
    },
  },
})
