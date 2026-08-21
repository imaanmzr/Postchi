import { defineStore } from 'pinia'
import type { LinkedRequest } from '~/utils/linkableRequests'

export type { LinkedRequest }

export interface TestCase {
  id: string
  workspace_id: string
  title: string
  description: string
  sort_order: number
  requests?: LinkedRequest[]
  updated_at: string
}

export const useTestCasesStore = defineStore('testCases', {
  state: () => ({
    cases: [] as TestCase[],
    current: null as TestCase | null,
    loading: false,
    error: '',
  }),
  actions: {
    async fetchCases(workspaceId: string) {
      const api = useApi()
      this.loading = true
      this.error = ''
      try {
        this.cases = await api.get<TestCase[]>(`/api/workspaces/${workspaceId}/test-cases`)
      } catch (e: unknown) {
        this.error = e instanceof Error ? e.message : 'Failed to load test cases'
        throw e
      } finally {
        this.loading = false
      }
    },
    async fetchCase(workspaceId: string, testCaseId: string) {
      const api = useApi()
      const tc = await api.get<TestCase>(`/api/workspaces/${workspaceId}/test-cases/${testCaseId}`)
      this.current = tc
      const idx = this.cases.findIndex(c => c.id === testCaseId)
      if (idx >= 0) this.cases[idx] = tc
      return tc
    },
    async createCase(workspaceId: string, title: string, description = '') {
      const api = useApi()
      const tc = await api.post<TestCase>(`/api/workspaces/${workspaceId}/test-cases`, { title, description })
      this.cases.push(tc)
      this.current = tc
      return tc
    },
    async updateCase(workspaceId: string, testCaseId: string, data: Partial<Pick<TestCase, 'title' | 'description' | 'sort_order'>>) {
      const api = useApi()
      const tc = await api.patch<TestCase>(`/api/workspaces/${workspaceId}/test-cases/${testCaseId}`, data)
      this.current = tc
      const idx = this.cases.findIndex(c => c.id === testCaseId)
      if (idx >= 0) this.cases[idx] = tc
      return tc
    },
    async deleteCase(workspaceId: string, testCaseId: string) {
      const api = useApi()
      await api.delete(`/api/workspaces/${workspaceId}/test-cases/${testCaseId}`)
      this.cases = this.cases.filter(c => c.id !== testCaseId)
      if (this.current?.id === testCaseId) this.current = null
    },
    async linkRequest(workspaceId: string, testCaseId: string, requestId: string) {
      const api = useApi()
      await api.post(`/api/workspaces/${workspaceId}/test-cases/${testCaseId}/requests/${requestId}`)
      await this.fetchCase(workspaceId, testCaseId)
    },
    async unlinkRequest(workspaceId: string, testCaseId: string, requestId: string) {
      const api = useApi()
      await api.delete(`/api/workspaces/${workspaceId}/test-cases/${testCaseId}/requests/${requestId}`)
      await this.fetchCase(workspaceId, testCaseId)
    },
    clearCurrent() {
      this.current = null
    },
  },
})
