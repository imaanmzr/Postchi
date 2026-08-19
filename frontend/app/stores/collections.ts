import { defineStore } from 'pinia'
import type { VariablesSpec } from '~/components/shared/VarsTableEditor'
import { formatRequestBody } from '~/utils/formatRequestBody'

export interface Collection {
  id: string
  workspace_id: string
  parent_id?: string | null
  name: string
  description?: string
  sort_order: number
  variables?: VariablesSpec
  headers?: { key: string; value: string; enabled: boolean }[]
  auth?: { type: string; config?: Record<string, unknown> }
  pre_request_script?: string
  test_script?: string
}

export interface RequestItem {
  id: string
  collection_id: string
  name: string
  method: string
  url: string
  headers: { key: string; value: string; enabled: boolean }[]
  params: { key: string; value: string; enabled: boolean }[]
  path_vars?: { key: string; value: string; enabled: boolean }[]
  body: { mode: string; raw: string; raw_lang: string }
  auth: { type: string; config?: Record<string, string> }
  settings: { timeout_ms: number; follow_redirects: boolean; verify_ssl: boolean }
  pre_request_script: string
  test_script: string
  sort_order?: number
  description?: string
  api_doc?: Record<string, unknown>
  docs_overridden?: boolean
  source_spec_id?: string
  source_operation_id?: string
  template_id?: string | null
  is_template?: boolean
  overridden_fields?: string[]
}

const emptyVars = (): VariablesSpec => ({ pre_request: [], post_response: [] })

export const useCollectionsStore = defineStore('collections', {
  state: () => ({
    collections: [] as Collection[],
    requests: [] as RequestItem[],
    activeRequest: null as RequestItem | null,
    activeCollection: null as Collection | null,
    activeCollectionId: null as string | null,
  }),
  getters: {
    tree: (s) => buildTree(s.collections),
    requestsByCollection: (s) => {
      const map: Record<string, RequestItem[]> = {}
      for (const r of s.requests) {
        if (r.template_id != null && r.template_id !== '') continue
        if (!map[r.collection_id]) map[r.collection_id] = []
        map[r.collection_id].push(r)
      }
      return map
    },
    variantsByTemplate: (s) => {
      const map: Record<string, RequestItem[]> = {}
      for (const r of s.requests) {
        if (!r.template_id) continue
        if (!map[r.template_id]) map[r.template_id] = []
        map[r.template_id].push(r)
      }
      return map
    },
  },
  actions: {
    async fetchCollections(workspaceId: string) {
      const api = useApi()
      this.collections = await api.get<Collection[]>(`/api/workspaces/${workspaceId}/collections`)
    },
    async fetchCollection(id: string) {
      const api = useApi()
      const col = await api.get<Collection>(`/api/collections/${id}`)
      this.activeCollection = col
      return col
    },
    async fetchRequests(collectionId: string) {
      const api = useApi()
      const list = await api.get<RequestItem[]>(`/api/requests?collection_id=${collectionId}`)
      const formatted = list.map(r => ({ ...r, body: formatRequestBody(r.body) }))
      const rest = this.requests.filter(r => r.collection_id !== collectionId)
      this.requests = [...rest, ...formatted]
      return formatted
    },
    async fetchAllRequests(workspaceId: string) {
      const api = useApi()
      const list = await api.get<RequestItem[]>(`/api/workspaces/${workspaceId}/requests`)
      this.requests = list.map(r => ({ ...r, body: formatRequestBody(r.body) }))
    },
    upsertRequest(req: RequestItem) {
      const normalized = { ...req, body: formatRequestBody(req.body) }
      const i = this.requests.findIndex(r => r.id === req.id)
      if (i >= 0) {
        this.requests[i] = { ...this.requests[i], ...normalized }
      } else {
        this.requests.push(normalized)
      }
    },
    async createCollection(workspaceId: string, data: Partial<Collection>) {
      const api = useApi()
      const col = await api.post<Collection>('/api/collections', { workspace_id: workspaceId, ...data })
      this.collections.push(col)
      return col
    },
    async updateCollection(id: string, data: Partial<Collection>) {
      const api = useApi()
      const col = await api.patch<Collection>(`/api/collections/${id}`, data)
      const i = this.collections.findIndex(c => c.id === id)
      if (i >= 0) this.collections[i] = { ...this.collections[i], ...col }
      if (this.activeCollection?.id === id) this.activeCollection = { ...this.activeCollection, ...col }
      return col
    },
    async deleteCollection(id: string) {
      const api = useApi()
      await api.delete(`/api/collections/${id}`)
      this.collections = this.collections.filter(c => c.id !== id)
      this.requests = this.requests.filter(r => r.collection_id !== id)
    },
    async duplicateCollection(id: string) {
      const api = useApi()
      return api.post<Collection>(`/api/collections/${id}/duplicate`, {})
    },
    async reorderCollections(items: { id: string; parent_id?: string | null; sort_order: number }[]) {
      const api = useApi()
      await api.patch('/api/collections/reorder', items)
      for (const item of items) {
        const col = this.collections.find(c => c.id === item.id)
        if (!col) continue
        col.sort_order = item.sort_order
        if (item.parent_id !== undefined) {
          col.parent_id = item.parent_id
        }
      }
    },
    async saveRequest(req: Partial<RequestItem>) {
      const api = useApi()
      const saved = req.id
        ? await api.patch<RequestItem>(`/api/requests/${req.id}`, req)
        : await api.post<RequestItem>('/api/requests', req)
      this.upsertRequest(saved)
      return saved
    },
    async deleteRequest(id: string) {
      const api = useApi()
      await api.delete(`/api/requests/${id}`)
      this.requests = this.requests.filter(r => r.id !== id)
    },
    async moveRequest(id: string, collectionId: string, sortOrder: number) {
      const api = useApi()
      await api.patch(`/api/requests/${id}/move`, { collection_id: collectionId, sort_order: sortOrder })
      const req = this.requests.find(r => r.id === id)
      if (req) {
        req.collection_id = collectionId
        req.sort_order = sortOrder
      }
    },
    async reorderRequests(items: { id: string; sort_order: number }[]) {
      const api = useApi()
      await api.patch('/api/requests/reorder', items)
      for (const item of items) {
        const req = this.requests.find(r => r.id === item.id)
        if (req) req.sort_order = item.sort_order
      }
    },
    async duplicateRequest(id: string) {
      const api = useApi()
      const dup = await api.post<RequestItem>(`/api/requests/${id}/duplicate`, {})
      this.upsertRequest(dup)
      return dup
    },
    async createChild(templateId: string, name?: string, overrides?: Partial<RequestItem>) {
      const api = useApi()
      const overridesMap: Record<string, unknown> = {}
      if (overrides) {
        for (const [k, v] of Object.entries(overrides)) {
          if (v !== undefined) overridesMap[k] = v
        }
      }
      const child = await api.post<RequestItem>(`/api/requests/${templateId}/children`, { name, overrides: overridesMap })
      this.requests.push(child)
      return child
    },
    async resetField(requestId: string, field: string) {
      const api = useApi()
      await api.post(`/api/requests/${requestId}/reset-field`, { field })
      const req = this.requests.find(r => r.id === requestId)
      if (req?.overridden_fields) {
        req.overridden_fields = req.overridden_fields.filter(f => f !== field)
      }
    },
    async promoteToTemplate(id: string) {
      const api = useApi()
      await api.post(`/api/requests/${id}/promote-to-template`, {})
      const req = this.requests.find(r => r.id === id)
      if (req) req.is_template = true
    },
    async listChildren(templateId: string) {
      const api = useApi()
      return api.get<{ id: string; name: string; method: string; overridden_fields: string[] }[]>(`/api/requests/${templateId}/children`)
    },
    setActiveRequest(req: RequestItem | null) {
      this.activeRequest = req
      if (req) this.activeCollectionId = req.collection_id
    },
    setActiveCollection(col: Collection | null) {
      this.activeCollection = col
      if (col) this.activeCollectionId = col.id
    },
  },
})

export interface TreeNode extends Collection {
  children: TreeNode[]
}

function buildTree(collections: Collection[]): TreeNode[] {
  const map = new Map<string, TreeNode>()
  const roots: TreeNode[] = []
  for (const c of collections) {
    map.set(c.id, { ...c, children: [] })
  }
  // A collection inside a parent_id cycle would otherwise vanish from the
  // tree (it has a parent, so it never becomes a root). Detect cycle members
  // and surface them as roots so the data stays visible and fixable.
  const inCycle = (id: string): boolean => {
    const visited = new Set<string>()
    let cur = map.get(id)
    while (cur?.parent_id) {
      if (visited.has(cur.id)) return true
      visited.add(cur.id)
      cur = map.get(cur.parent_id)
    }
    return false
  }
  for (const c of collections) {
    const node = map.get(c.id)!
    if (c.parent_id && map.has(c.parent_id) && !inCycle(c.id)) {
      map.get(c.parent_id)!.children.push(node)
    } else {
      roots.push(node)
    }
  }
  const sortNodes = (nodes: TreeNode[]) => {
    nodes.sort((a, b) => a.sort_order - b.sort_order)
    nodes.forEach(n => sortNodes(n.children))
  }
  sortNodes(roots)
  return roots
}

export type { TreeNode }
