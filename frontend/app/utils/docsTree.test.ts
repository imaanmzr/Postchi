import { describe, expect, it } from 'vitest'
import { buildDocTree, flattenTree, fuzzySearchDocs } from './docsTree'

describe('docsTree', () => {
  const summaries = [
    { id: '1', workspace_id: 'w', slug: 'docs-a-b', title: 'B Doc', source_path: 'docs/a/b', is_local: false, updated_at: '' },
    { id: '2', workspace_id: 'w', slug: 'docs-a-c', title: 'C Doc', source_path: 'docs/a/c', is_local: false, updated_at: '' },
  ]

  it('builds nested folders from source_path', () => {
    const tree = buildDocTree(summaries)
    expect(tree).toHaveLength(1)
    expect(tree[0]!.type).toBe('folder')
    expect(tree[0]!.children).toHaveLength(1)
    expect(tree[0]!.children[0]!.children).toHaveLength(2)
  })

  it('flattens with expand state', () => {
    const tree = buildDocTree(summaries)
    const rows = flattenTree(tree, { docs: true, 'docs/a': true })
    expect(rows.some(r => r.type === 'file')).toBe(true)
  })

  it('fuzzy search returns matches', () => {
    const result = fuzzySearchDocs(summaries, 'B Doc')
    expect(result.matchingSlugs.has('docs-a-b')).toBe(true)
    expect(result.expandPaths.has('docs')).toBe(true)
    expect(result.expandPaths.has('docs/a')).toBe(true)
  })
})
