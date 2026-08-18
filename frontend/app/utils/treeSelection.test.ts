import { describe, expect, it } from 'vitest'
import type { Collection } from '~/stores/collections'
import {
  allTreeSelectionKeys,
  collectionSubtreeIds,
  topLevelSelectedCollectionIds,
  treeSelectionKey,
} from './treeSelection'

const collections: Collection[] = [
  { id: 'root', workspace_id: 'ws', name: 'Root', sort_order: 0 },
  { id: 'child', workspace_id: 'ws', parent_id: 'root', name: 'Child', sort_order: 0 },
  { id: 'other', workspace_id: 'ws', name: 'Other', sort_order: 1 },
]

describe('treeSelection', () => {
  it('builds selection keys', () => {
    expect(treeSelectionKey('collection', 'abc')).toBe('collection:abc')
    expect(treeSelectionKey('request', 'xyz')).toBe('request:xyz')
  })

  it('lists all selectable keys', () => {
    const keys = allTreeSelectionKeys(collections, [
      {
        id: 'req1',
        collection_id: 'root',
        name: 'Req',
        method: 'GET',
        url: '',
        headers: [],
        params: [],
        body: { mode: 'raw', raw: '', raw_lang: 'json' },
        auth: { type: 'none' },
        settings: { timeout_ms: 0, follow_redirects: true, verify_ssl: true },
        pre_request_script: '',
        test_script: '',
      },
    ])
    expect(keys).toContain('collection:root')
    expect(keys).toContain('request:req1')
  })

  it('keeps only top-level selected collections', () => {
    const selected = new Set(['root', 'child'])
    expect(topLevelSelectedCollectionIds(selected, collections)).toEqual(['root'])
  })

  it('collects collection subtree ids', () => {
    const subtree = collectionSubtreeIds(['root'], collections)
    expect([...subtree].sort()).toEqual(['child', 'root'])
  })
})
