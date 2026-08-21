import { defineStore } from 'pinia'

export interface Workspace {
  id: string
  name: string
  description?: string
  role?: string
  variables?: Record<string, unknown>
}

export interface Member {
  user_id: string
  email: string
  display_name: string
  role: string
}

export interface PendingInvite {
  id: string
  workspace_id: string
  email: string
  role: string
  expires_at: string
  invite_url?: string
}

export type AddTeamResult =
  | { outcome: 'added', member: Member }
  | { outcome: 'invited', invite: PendingInvite, invite_url: string, email_sent: boolean }

export const useWorkspaceStore = defineStore('workspace', {
  state: () => ({
    workspaces: [] as Workspace[],
    current: null as Workspace | null,
    members: [] as Member[],
    pendingInvites: [] as PendingInvite[],
  }),
  actions: {
    async fetchWorkspaces() {
      const api = useApi()
      this.workspaces = await api.get<Workspace[]>('/api/workspaces')
    },
    async fetchWorkspace(id: string) {
      const api = useApi()
      const ws = await api.get<Workspace>(`/api/workspaces/${id}`)
      const fromList = this.workspaces.find(w => w.id === id)
      if (!ws.role && fromList?.role) ws.role = fromList.role
      this.current = ws
      return ws
    },
    async create(name: string, description = '') {
      const api = useApi()
      const ws = await api.post<Workspace>('/api/workspaces', { name, description })
      this.workspaces.unshift(ws)
      return ws
    },
    async update(id: string, data: Partial<Workspace>) {
      const api = useApi()
      const ws = await api.patch<Workspace>(`/api/workspaces/${id}`, data)
      const i = this.workspaces.findIndex(w => w.id === id)
      if (i >= 0) this.workspaces[i] = { ...this.workspaces[i], ...ws }
      if (this.current?.id === id) this.current = { ...this.current, ...ws }
      return ws
    },
    async delete(id: string) {
      const api = useApi()
      await api.delete(`/api/workspaces/${id}`)
      this.workspaces = this.workspaces.filter(w => w.id !== id)
    },
    async fetchMembers(workspaceId: string) {
      const api = useApi()
      this.members = await api.get<Member[]>(`/api/workspaces/${workspaceId}/members`)
    },
    async addMember(workspaceId: string, email: string, role: string, sendEmail?: boolean) {
      const api = useApi()
      const body: { email: string, role: string, send_email?: boolean } = { email, role }
      if (sendEmail !== undefined) body.send_email = sendEmail
      const result = await api.post<AddTeamResult>(`/api/workspaces/${workspaceId}/invites`, body)
      if (result.outcome === 'added') {
        await this.fetchMembers(workspaceId)
      } else {
        await this.fetchPendingInvites(workspaceId)
      }
      return result
    },
    async fetchPendingInvites(workspaceId: string) {
      const api = useApi()
      this.pendingInvites = await api.get<PendingInvite[]>(`/api/workspaces/${workspaceId}/invites`)
    },
    async revokeInvite(workspaceId: string, inviteId: string) {
      const api = useApi()
      await api.delete(`/api/workspaces/${workspaceId}/invites/${inviteId}`)
      await this.fetchPendingInvites(workspaceId)
    },
    async updateMember(workspaceId: string, userId: string, role: string) {
      const api = useApi()
      await api.patch(`/api/workspaces/${workspaceId}/members/${userId}`, { role })
      await this.fetchMembers(workspaceId)
    },
    async removeMember(workspaceId: string, userId: string) {
      const api = useApi()
      await api.delete(`/api/workspaces/${workspaceId}/members/${userId}`)
      await this.fetchMembers(workspaceId)
    },
    setCurrent(ws: Workspace | null) {
      this.current = ws
    },
  },
})
