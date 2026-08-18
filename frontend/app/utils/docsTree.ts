import Fuse from 'fuse.js'

export interface DocSummary {
  id: string
  workspace_id: string
  slug: string
  title: string
  source_path: string
  is_local: boolean
  updated_at: string
}

export type TreeNodeType = 'folder' | 'file'

export interface DocsTreeNode {
  id: string
  name: string
  type: TreeNodeType
  path: string
  doc?: DocSummary
  children: DocsTreeNode[]
}

export interface FlatTreeRow {
  id: string
  name: string
  type: TreeNodeType
  path: string
  depth: number
  doc?: DocSummary
  hasChildren: boolean
  expanded: boolean
  matchRanges?: [number, number][]
}

export function buildDocTree(summaries: DocSummary[]): DocsTreeNode[] {
  const root: DocsTreeNode[] = []
  const folderMap = new Map<string, DocsTreeNode>()

  const sorted = [...summaries].sort((a, b) =>
    a.source_path.localeCompare(b.source_path, undefined, { sensitivity: 'base' }),
  )

  for (const doc of sorted) {
    const sourcePath = doc.source_path || doc.slug.replace(/-/g, '/')
    const parts = sourcePath.split('/').filter(Boolean)
    if (parts.length === 0) continue

    let parentPath = ''
    let siblings = root

    for (let i = 0; i < parts.length - 1; i++) {
      const part = parts[i]!
      parentPath = parentPath ? `${parentPath}/${part}` : part
      let folder = folderMap.get(parentPath)
      if (!folder) {
        folder = {
          id: `folder:${parentPath}`,
          name: part,
          type: 'folder',
          path: parentPath,
          children: [],
        }
        folderMap.set(parentPath, folder)
        siblings.push(folder)
      }
      siblings = folder.children
    }

    const fileName = parts[parts.length - 1]!
    const displayName = doc.title && doc.title !== doc.source_path ? doc.title : fileName
    siblings.push({
      id: doc.slug,
      name: displayName,
      type: 'file',
      path: sourcePath,
      doc,
      children: [],
    })
  }

  sortTreeNodes(root)
  return root
}

function sortTreeNodes(nodes: DocsTreeNode[]) {
  nodes.sort((a, b) => {
    if (a.type !== b.type) return a.type === 'folder' ? -1 : 1
    return a.name.localeCompare(b.name, undefined, { sensitivity: 'base' })
  })
  for (const node of nodes) {
    if (node.children.length) sortTreeNodes(node.children)
  }
}

export function flattenTree(
  nodes: DocsTreeNode[],
  expanded: Record<string, boolean>,
  depth = 0,
): FlatTreeRow[] {
  const rows: FlatTreeRow[] = []
  for (const node of nodes) {
    const isFolder = node.type === 'folder'
    const expandedState = expanded[node.path] === true
    rows.push({
      id: node.id,
      name: node.name,
      type: node.type,
      path: node.path,
      depth,
      doc: node.doc,
      hasChildren: isFolder && node.children.length > 0,
      expanded: expandedState,
    })
    if (isFolder && expandedState) {
      rows.push(...flattenTree(node.children, expanded, depth + 1))
    }
  }
  return rows
}

export interface FuzzySearchResult {
  matchingSlugs: Set<string>
  expandPaths: Set<string>
  highlights: Map<string, [number, number][]>
}

export function fuzzySearchDocs(summaries: DocSummary[], query: string): FuzzySearchResult {
  const empty: FuzzySearchResult = {
    matchingSlugs: new Set(),
    expandPaths: new Set(),
    highlights: new Map(),
  }
  const q = query.trim()
  if (!q) return empty

  const fuse = new Fuse(summaries, {
    keys: [
      { name: 'title', weight: 0.5 },
      { name: 'source_path', weight: 0.35 },
      { name: 'slug', weight: 0.15 },
    ],
    threshold: 0.4,
    includeMatches: true,
  })

  const results = fuse.search(q)
  const matchingSlugs = new Set<string>()
  const expandPaths = new Set<string>()
  const highlights = new Map<string, [number, number][]>()

  for (const result of results) {
    const doc = result.item
    matchingSlugs.add(doc.slug)
    const path = doc.source_path || doc.slug
    const parts = path.split('/').filter(Boolean)
    for (let i = 0; i < parts.length - 1; i++) {
      expandPaths.add(parts.slice(0, i + 1).join('/'))
    }
    if (result.matches) {
      for (const match of result.matches) {
        if (match.key === 'title' && match.indices?.length) {
          highlights.set(doc.slug, match.indices as [number, number][])
        }
      }
    }
  }

  return { matchingSlugs, expandPaths, highlights }
}

export function filterTreeForSearch(
  nodes: DocsTreeNode[],
  matchingSlugs: Set<string>,
  expandPaths: Set<string>,
): DocsTreeNode[] {
  if (matchingSlugs.size === 0) return nodes

  const filterNode = (node: DocsTreeNode): DocsTreeNode | null => {
    if (node.type === 'file') {
      return node.doc && matchingSlugs.has(node.doc.slug) ? node : null
    }
    const children = node.children
      .map(filterNode)
      .filter((n): n is DocsTreeNode => n !== null)
    if (children.length > 0 || expandPaths.has(node.path)) {
      return { ...node, children }
    }
    return null
  }

  return nodes.map(filterNode).filter((n): n is DocsTreeNode => n !== null)
}

export function ancestorFolderPaths(sourcePath: string): string[] {
  const parts = sourcePath.split('/').filter(Boolean)
  const paths: string[] = []
  for (let i = 0; i < parts.length - 1; i++) {
    paths.push(parts.slice(0, i + 1).join('/'))
  }
  return paths
}

export function highlightName(name: string, ranges?: [number, number][]): string {
  if (!ranges?.length) return escapeHtml(name)
  let out = ''
  let last = 0
  for (const [start, end] of ranges) {
    out += escapeHtml(name.slice(last, start))
    out += `<mark class="docs-tree-match">${escapeHtml(name.slice(start, end + 1))}</mark>`
    last = end + 1
  }
  out += escapeHtml(name.slice(last))
  return out
}

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
}
