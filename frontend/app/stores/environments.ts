import { defineStore } from 'pinia'
import type { VariablesSpec } from '~/components/shared/VarsTableEditor'

export interface EnvVariable {
  id?: string
  key: string
  value: string
  expr?: string
  phase: 'pre_request' | 'post_response'
  enabled: boolean
  type: string
  description: string
  is_secret: boolean
}

export interface Environment {
  id: string
  workspace_id: string
  name: string
  stage?: string
  variables?: EnvVariable[]
}

export const useEnvironmentsStore = defineStore('environments', {
  state: () => ({
    environments: [] as Environment[],
    activeId: null as string | null,
  }),
  getters: {
    active: (s) => s.environments.find(e => e.id === s.activeId) || null,
  },
  actions: {
    async fetch(workspaceId: string) {
      const api = useApi()
      this.environments = await api.get<Environment[]>(`/api/environments?workspace_id=${workspaceId}`)
    },
    async fetchOne(id: string) {
      const api = useApi()
      return api.get<Environment>(`/api/environments/${id}`)
    },
    async create(workspaceId: string, name: string) {
      const api = useApi()
      const env = await api.post<Environment>('/api/environments', { workspace_id: workspaceId, name })
      this.environments.push(env)
      return env
    },
    async update(id: string, data: Partial<Environment>) {
      const api = useApi()
      const env = await api.patch<Environment>(`/api/environments/${id}`, data)
      const i = this.environments.findIndex(e => e.id === id)
      if (i >= 0) this.environments[i] = { ...this.environments[i], ...env }
      return env
    },
    async delete(id: string) {
      const api = useApi()
      await api.delete(`/api/environments/${id}`)
      this.environments = this.environments.filter(e => e.id !== id)
      if (this.activeId === id) this.setActive(null)
    },
    envVarsToSpec(vars: EnvVariable[]): VariablesSpec {
      return {
        pre_request: vars.filter(v => v.phase === 'pre_request').map(v => ({
          enabled: v.enabled,
          name: v.key,
          value: v.value,
          type: v.type || 'string',
          description: v.description || '',
          secret: v.is_secret,
        })),
        post_response: vars.filter(v => v.phase === 'post_response').map(v => ({
          enabled: v.enabled,
          name: v.key,
          expr: v.expr || '',
          description: v.description || '',
        })),
      }
    },
    specToEnvVars(spec: VariablesSpec, existing: EnvVariable[] = []): EnvVariable[] {
      const vars: EnvVariable[] = []
      for (const row of spec.pre_request) {
        const prev = existing.find(e => e.key === row.name && e.phase === 'pre_request')
        vars.push({
          id: prev?.id,
          key: row.name,
          value: row.value,
          phase: 'pre_request',
          enabled: row.enabled,
          type: row.type,
          description: row.description,
          is_secret: row.secret,
        })
      }
      for (const row of spec.post_response) {
        const prev = existing.find(e => e.key === row.name && e.phase === 'post_response')
        vars.push({
          id: prev?.id,
          key: row.name,
          value: '',
          expr: row.expr,
          phase: 'post_response',
          enabled: row.enabled,
          type: 'string',
          description: row.description,
          is_secret: false,
        })
      }
      return vars
    },
    setActive(id: string | null) {
      this.activeId = id
      if (import.meta.client) {
        if (id) localStorage.setItem('active_environment', id)
        else localStorage.removeItem('active_environment')
      }
    },
    async hydrateActive() {
      if (!this.activeId) return
      const existing = this.environments.find(e => e.id === this.activeId)
      if (existing?.variables?.length) return
      try {
        const full = await this.fetchOne(this.activeId)
        const i = this.environments.findIndex(e => e.id === full.id)
        if (i >= 0) this.environments[i] = { ...this.environments[i], ...full }
      } catch {
        // ignore — suggestions still work without env vars
      }
    },
    loadActive() {
      if (import.meta.client) {
        this.activeId = localStorage.getItem('active_environment')
      }
    },
    async resolveVariables(envId: string, names: string[]) {
      const api = useApi()
      return api.post<{ existing: string[]; missing: string[] }>(`/api/environments/${envId}/resolve-variables`, { names })
    },
    async bulkSetVariables(envId: string, vars: { key: string; value: string; is_secret: boolean }[]) {
      const api = useApi()
      return api.post<Environment>(`/api/environments/${envId}/variables/bulk`, vars)
    },
  },
})
