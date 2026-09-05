export interface GitRepoFormFields {
  repo_url: string
  branch: string
  path_prefix: string
  link_template?: string
}

const knownRepoFolderNames = new Set([
  'bruno-collection',
  'bruno',
  'collections',
  'docs',
  'documentation',
  'openapi',
  'postman',
])

function isLikelyRepoFolder(name: string): boolean {
  if (name.includes('.')) return true
  return knownRepoFolderNames.has(name.toLowerCase())
}

export function parseGitLabTreeRef(rest: string): { branch: string, path: string } {
  const trimmed = rest.trim().replace(/^\/+|\/+$/g, '')
  if (!trimmed) return { branch: '', path: '' }
  const segments = trimmed.split('/').filter(Boolean)
  if (segments.length === 0) return { branch: '', path: '' }
  if (segments.length === 1) {
    return { branch: decodeURIComponent(segments[0]!), path: '' }
  }

  const first = segments[0]!
  if (first.includes('%2F') || first.includes('%2f')) {
    return {
      branch: decodeURIComponent(first),
      path: segments.slice(1).join('/'),
    }
  }

  const last = segments[segments.length - 1]!
  if (isLikelyRepoFolder(last)) {
    return {
      branch: decodeURIComponent(segments.slice(0, -1).join('/')),
      path: last,
    }
  }

  return {
    branch: decodeURIComponent(first),
    path: segments.slice(1).join('/'),
  }
}

export function normalizePathPrefix(branch: string, pathPrefix: string): string {
  const b = branch.trim().replace(/^\/+|\/+$/g, '')
  let p = pathPrefix.trim().replace(/^\/+|\/+$/g, '')
  if (!b || !p) return p
  const lastSegment = b.includes('/') ? b.slice(b.lastIndexOf('/') + 1) : b
  if (lastSegment && p.startsWith(`${lastSegment}/`)) {
    p = p.slice(lastSegment.length + 1)
  }
  return p
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
  const rest = url.slice(treeIdx + '/-/tree/'.length).split('?')[0]!
  const parsed = parseGitLabTreeRef(rest)
  if (parsed.branch) form.branch = parsed.branch
  form.path_prefix = normalizePathPrefix(parsed.branch, parsed.path.replace(/\/$/, ''))
}

export function gitRepoConfigPayload(form: GitRepoFormFields) {
  const branch = form.branch.trim() || 'main'
  const config: Record<string, string> = {
    repo_url: form.repo_url.trim(),
    branch,
    path_prefix: normalizePathPrefix(branch, form.path_prefix.trim()),
  }
  const linkTemplate = form.link_template?.trim()
  if (linkTemplate) {
    config.link_template = linkTemplate
  }
  return config
}
