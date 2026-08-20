import { describe, expect, it } from 'vitest'
import type { Collection } from '~/stores/collections'
import {
  flattenCollectionOptions,
  importParentPayload,
  indentLabel,
  isImportParentValid,
} from '~/utils/importParent'

const collections: Collection[] = [
  { id: 'root', workspace_id: 'ws', name: 'Root', sort_order: 0 },
  { id: 'child', workspace_id: 'ws', parent_id: 'root', name: 'Child', sort_order: 0 },
]

describe('importParent', () => {
  it('builds payload for existing parent', () => {
    expect(importParentPayload({ mode: 'existing', parentId: 'child' })).toEqual({ parent_id: 'child' })
  })

  it('builds payload for new parent', () => {
    expect(importParentPayload({ mode: 'new', newParentName: ' Git ' })).toEqual({
      create_parent: { name: 'Git' },
    })
  })

  it('validates choices', () => {
    expect(isImportParentValid({ mode: 'root' })).toBe(true)
    expect(isImportParentValid({ mode: 'existing' })).toBe(false)
    expect(isImportParentValid({ mode: 'existing', parentId: 'root' })).toBe(true)
    expect(isImportParentValid({ mode: 'new', newParentName: 'X' })).toBe(true)
    expect(isImportParentValid({ mode: 'new', newParentName: '  ' })).toBe(false)
  })

  it('flattens collection tree with depth', () => {
    const options = flattenCollectionOptions(collections)
    expect(options).toEqual([
      { id: 'root', label: 'Root', depth: 0 },
      { id: 'child', label: 'Child', depth: 1 },
    ])
    expect(indentLabel('Child', 1)).toBe('  Child')
  })
})
