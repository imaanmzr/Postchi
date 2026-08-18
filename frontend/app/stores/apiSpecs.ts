import { defineStore } from 'pinia'

export interface ApiSpec {
  id: string
  workspace_id: string
  collection_id?: string
  name: string
  source_type: string
  spec_url: string
  spec_hash: string
  base_url_var: string
  last_synced_at?: string
  created_at: string
  updated_at: string
}

export interface SyncDiff {
  added: SyncItem[]
  updated: SyncItem[]
  removed: SyncItem[]
}

export interface SyncItem {
  operation_id: string
  name: string
  method?: string
  path?: string
}

export function normalizeSyncDiff(diff: Partial<SyncDiff> | null | undefined): SyncDiff {
  return {
    added: diff?.added ?? [],
    updated: diff?.updated ?? [],
    removed: diff?.removed ?? [],
  }
}

export interface EnvURL {
  environment_id: string
  base_url: string
}

export const useApiSpecsStore = defineStore('apiSpecs', {
  state: () => ({
    specs: [] as ApiSpec[],
  }),
  actions: {
    async list(workspaceId: string) {
      const api = useApi()
      this.specs = await api.get<ApiSpec[]>(`/api/workspaces/${workspaceId}/api-specs`)
      return this.specs
    },
    async create(workspaceId: string, data: { name: string; spec_url: string; collection_id?: string; base_url_var?: string }) {
      const api = useApi()
      const spec = await api.post<ApiSpec>(`/api/workspaces/${workspaceId}/api-specs`, data)
      this.specs.push(spec)
      return spec
    },
    async update(id: string, data: Partial<ApiSpec>) {
      const api = useApi()
      const spec = await api.patch<ApiSpec>(`/api/api-specs/${id}`, data)
      const i = this.specs.findIndex(s => s.id === id)
      if (i >= 0) this.specs[i] = { ...this.specs[i], ...spec }
      return spec
    },
    async delete(id: string) {
      const api = useApi()
      await api.delete(`/api/api-specs/${id}`)
      this.specs = this.specs.filter(s => s.id !== id)
    },
    async setEnvironmentUrls(id: string, urls: EnvURL[]) {
      const api = useApi()
      await api.put(`/api/api-specs/${id}/environment-urls`, urls)
    },
    async sync(id: string, apply: boolean) {
      const api = useApi()
      const diff = await api.post<SyncDiff>(`/api/api-specs/${id}/sync`, { apply })
      return normalizeSyncDiff(diff)
    },
    async upload(workspaceId: string, data: ArrayBuffer | string, name?: string, collectionId?: string) {
      const api = useApi()
      const qs = new URLSearchParams()
      if (name) qs.set('name', name)
      if (collectionId) qs.set('collection_id', collectionId)
      const path = `/api/workspaces/${workspaceId}/api-specs/upload${qs.toString() ? `?${qs}` : ''}`
      return api.uploadRaw<ApiSpec>(path, data, 'application/json')
    },
    async reupload(id: string, data: ArrayBuffer | string, apply = true) {
      const api = useApi()
      const diff = await api.uploadRaw<SyncDiff>(`/api/api-specs/${id}/reupload?apply=${apply}`, data, 'application/json')
      return normalizeSyncDiff(diff)
    },
  },
})
