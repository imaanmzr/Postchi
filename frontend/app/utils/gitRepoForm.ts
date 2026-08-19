export interface GitRepoFormFields {
  repo_url: string
  branch: string
  path_prefix: string
}

export function detectedGitProvider(repoUrl: string): string {
  const trimmed = repoUrl.trim()
  const lower = trimmed.toLowerCase()
  if (!lower.includes('://')) return ''
  if (lower.includes('gitlab') || lower.includes('/-/tree/') || lower.includes('/-/blob/')) return 'GitLab'
  if (lower.includes('github')) return 'GitHub'
  try {
    const host = new URL(trimmed).hostname.toLowerCase()
    if (host.includes('gitlab')) return 'GitLab'
    if (host.includes('github')) return 'GitHub'
    if (host.startsWith('git.')) return 'GitLab'
  } catch {
    return ''
  }
  return 'GitHub'
}

export function applyGitLabBrowseUrlHints(form: GitRepoFormFields) {
  const url = form.repo_url.trim()
  const treeIdx = url.indexOf('/-/tree/')
  if (treeIdx < 0) return
  const rest = url.slice(treeIdx + '/-/tree/'.length).split('?')[0]
  const slash = rest.indexOf('/')
  const branch = slash >= 0 ? rest.slice(0, slash) : rest
  const folder = slash >= 0 ? rest.slice(slash + 1) : ''
  if (branch) form.branch = branch
  form.path_prefix = folder.replace(/\/$/, '')
}

export function gitRepoConfigPayload(form: GitRepoFormFields) {
  return {
    repo_url: form.repo_url.trim(),
    branch: form.branch.trim() || 'main',
    path_prefix: form.path_prefix.trim(),
  }
}
