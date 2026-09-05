export interface GitBranch {
  name: string
  is_default?: boolean
  protected?: boolean
}

export interface GitBranchListResponse {
  branches: GitBranch[]
  cached: boolean
  fetched_at: string
}

export type GitSourceKind = 'doc' | 'bruno'

function buildQuery(search?: string, refresh?: boolean) {
  const params = new URLSearchParams()
  if (search?.trim()) params.set('search', search.trim())
  if (refresh) params.set('refresh', 'true')
  const q = params.toString()
  return q ? `?${q}` : ''
}

export function useGitBranches() {
  const api = useApi()

  const branches = ref<GitBranch[]>([])
  const loading = ref(false)
  const error = ref('')
  const cached = ref(false)
  const fetchedAt = ref<string | null>(null)

  let debounceTimer: ReturnType<typeof setTimeout> | null = null

  async function fetchForSource(
    workspaceId: string,
    kind: GitSourceKind,
    sourceId: string,
    search = '',
    refresh = false,
  ) {
    loading.value = true
    error.value = ''
    try {
      const base = kind === 'doc' ? 'doc-sources' : 'bruno-sources'
      const result = await api.get<GitBranchListResponse>(
        `/api/workspaces/${workspaceId}/${base}/${sourceId}/branches${buildQuery(search, refresh)}`,
      )
      branches.value = result.branches ?? []
      cached.value = !!result.cached
      fetchedAt.value = result.fetched_at ?? null
    } catch (e) {
      branches.value = []
      error.value = e instanceof Error ? e.message : 'Failed to load branches'
    } finally {
      loading.value = false
    }
  }

  async function previewBranches(
    workspaceId: string,
    repoUrl: string,
    accessToken = '',
    search = '',
    refresh = false,
  ) {
    loading.value = true
    error.value = ''
    try {
      const result = await api.post<GitBranchListResponse>(
        `/api/workspaces/${workspaceId}/git/branches/preview${buildQuery(search, refresh)}`,
        {
          repo_url: repoUrl.trim(),
          access_token: accessToken.trim() || undefined,
        },
      )
      branches.value = result.branches ?? []
      cached.value = !!result.cached
      fetchedAt.value = result.fetched_at ?? null
    } catch (e) {
      branches.value = []
      error.value = e instanceof Error ? e.message : 'Failed to load branches'
    } finally {
      loading.value = false
    }
  }

  function schedulePreview(
    workspaceId: string,
    repoUrl: string,
    accessToken = '',
    search = '',
    refresh = false,
    delayMs = 300,
  ) {
    if (debounceTimer) clearTimeout(debounceTimer)
    debounceTimer = setTimeout(() => {
      if (!repoUrl.trim()) {
        branches.value = []
        error.value = ''
        cached.value = false
        fetchedAt.value = null
        return
      }
      void previewBranches(workspaceId, repoUrl, accessToken, search, refresh)
    }, delayMs)
  }

  function scheduleForSource(
    workspaceId: string,
    kind: GitSourceKind,
    sourceId: string,
    search = '',
    refresh = false,
    delayMs = 300,
  ) {
    if (debounceTimer) clearTimeout(debounceTimer)
    debounceTimer = setTimeout(() => {
      void fetchForSource(workspaceId, kind, sourceId, search, refresh)
    }, delayMs)
  }

  return {
    branches,
    loading,
    error,
    cached,
    fetchedAt,
    fetchForSource,
    previewBranches,
    schedulePreview,
    scheduleForSource,
  }
}
