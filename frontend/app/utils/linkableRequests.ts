export interface LinkedRequest {
  id: string
  name: string
  method: string
  url: string
  workspace_id: string
  workspace_name: string
}

interface WorkspaceRequest {
  id: string
  name: string
  method: string
  url: string
}

export async function fetchLinkableRequests(): Promise<LinkedRequest[]> {
  const wsStore = useWorkspaceStore()
  const api = useApi()
  if (!wsStore.workspaces.length) {
    await wsStore.fetchWorkspaces()
  }
  const batches = await Promise.all(
    wsStore.workspaces.map(async (ws) => {
      try {
        const requests = await api.get<WorkspaceRequest[]>(`/api/workspaces/${ws.id}/requests`)
        return requests.map(r => ({
          id: r.id,
          name: r.name,
          method: r.method,
          url: r.url,
          workspace_id: ws.id,
          workspace_name: ws.name,
        }))
      } catch {
        return [] as LinkedRequest[]
      }
    }),
  )
  return batches
    .flat()
    .sort((a, b) =>
      a.workspace_name.localeCompare(b.workspace_name)
      || a.name.localeCompare(b.name),
    )
}

export function linkedRequestSubtitle(req: LinkedRequest): string {
  return `${req.workspace_name} · ${req.method} ${req.url}`
}
