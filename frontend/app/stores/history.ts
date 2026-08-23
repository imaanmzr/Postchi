import { defineStore } from 'pinia'
import { normalizeExecutionResult } from '~/utils/executionResponse'

export interface HistoryEntry {
  id: string
  workspace_id: string
  request_id?: string | null
  snapshot: {
    name?: string
    method?: string
    url?: string
    [key: string]: unknown
  }
  response: {
    status_code?: number
    body?: string
    headers?: Record<string, string>
    timing?: Record<string, number>
    test_results?: Array<{ name: string; passed: boolean; message?: string }>
    console?: string[]
    body_size?: number
    [key: string]: unknown
  }
  test_results?: Array<{ name: string; passed: boolean; message?: string }>
  executed_by: string
  executed_by_name?: string
  executed_by_email?: string
  executed_at: string
  duration_ms: number
  status_code: number
}

export const useHistoryStore = defineStore('history', {
  state: () => ({
    entries: [] as HistoryEntry[],
    selectedId: null as string | null,
  }),
  getters: {
    selected: (s) => s.entries.find(e => e.id === s.selectedId) ?? null,
  },
  actions: {
    async fetch(workspaceId: string, requestId?: string) {
      const api = useApi()
      let path = `/api/history?workspace_id=${workspaceId}`
      if (requestId) path += `&request_id=${requestId}`
      this.entries = await api.get<HistoryEntry[]>(path)
    },
    select(id: string | null) {
      this.selectedId = id
    },
  },
})

/** Build a ResponseViewer-compatible object from a history entry. */
export function historyEntryToResponse(entry: HistoryEntry) {
  const resp = entry.response || {}
  return normalizeExecutionResult({
    ...resp,
    status_code: entry.status_code ?? resp.status_code ?? 0,
    timing: resp.timing ?? { total_ms: entry.duration_ms },
    test_results: entry.test_results ?? resp.test_results,
  })
}
