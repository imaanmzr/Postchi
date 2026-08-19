export function buildWorkspaceRequestUrl(workspaceId: string, requestId: string) {
  const q = new URLSearchParams({ request: requestId })
  return `/workspaces/${workspaceId}?${q}`
}

export function normalizeDocPath(path?: string | null): string {
  if (!path) return ''
  return path.toLowerCase().replace(/\s+/g, '').replace(/\\/g, '/')
}

export interface DocIdentity {
  id: string
  slug: string
  source_path?: string | null
}

export function isSameDoc(a: DocIdentity, b: DocIdentity): boolean {
  if (a.id === b.id || a.slug === b.slug) return true
  const paths = (doc: DocIdentity) => {
    const out = new Set<string>()
    const fromSource = normalizeDocPath(doc.source_path)
    if (fromSource) out.add(fromSource)
    const fromSlug = normalizeDocPath(doc.slug.replace(/-/g, '/'))
    if (fromSlug) out.add(fromSlug)
    return out
  }
  const aPaths = paths(a)
  for (const p of paths(b)) {
    if (aPaths.has(p)) return true
  }
  return false
}

export const DOC_ALREADY_LINKED_MESSAGE = 'This document is already linked to this request.'
