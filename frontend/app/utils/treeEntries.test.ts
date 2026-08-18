import { describe, expect, it } from 'vitest'
import { mergeTreeEntries } from './treeEntries'
import type { RequestItem, TreeNode } from '~/stores/collections'

function node(id: string, sort: number, children: TreeNode[] = []): TreeNode {
  return {
    id,
    workspace_id: 'ws',
    name: id,
    sort_order: sort,
    children,
  }
}

function req(id: string, colId: string, sort: number): RequestItem {
  return {
    id,
    collection_id: colId,
    name: id,
    method: 'GET',
    url: '',
    headers: [],
    params: [],
    body: { mode: 'none', raw: '', raw_lang: 'json' },
    auth: { type: 'none' },
    settings: { timeout_ms: 30000, follow_redirects: true, verify_ssl: true },
    pre_request_script: '',
    test_script: '',
    sort_order: sort,
  }
}

describe('mergeTreeEntries', () => {
  it('orders folders and requests by sort_order', () => {
    const root = node('root', 0, [node('folder-b', 1), node('empty', 2)])
    const requests = [req('first', 'root', 0), req('last', 'root', 3)]
    const entries = mergeTreeEntries(root, requests)
    expect(entries.map(e => e.kind === 'folder' ? e.node.id : e.request.id)).toEqual([
      'first',
      'folder-b',
      'empty',
      'last',
    ])
  })
})
