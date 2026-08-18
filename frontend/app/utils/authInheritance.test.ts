import { describe, expect, it } from 'vitest'
import type { Collection } from '~/stores/collections'
import { inheritSourceLabel, resolveRequestInheritedAuth } from './authInheritance'

function col(id: string, name: string, parentId?: string, auth?: Collection['auth']): Collection {
  return {
    id,
    workspace_id: 'ws',
    name,
    parent_id: parentId,
    sort_order: 0,
    auth,
  }
}

describe('resolveRequestInheritedAuth', () => {
  const collections = [
    col('root', 'API', undefined, { type: 'bearer', config: { token: 'root' } }),
    col('folder', 'Auth Folder', 'root', { type: 'inherit' }),
    col('nested', 'Nested', 'folder', { type: 'basic', config: { username: 'u', password: 'p' } }),
  ]

  it('uses explicit request auth', () => {
    const result = resolveRequestInheritedAuth('nested', { type: 'apikey' }, collections)
    expect(result.auth.type).toBe('apikey')
    expect(result.source).toBeUndefined()
  })

  it('inherits nearest explicit folder auth', () => {
    const result = resolveRequestInheritedAuth('nested', { type: 'inherit' }, collections)
    expect(result.auth.type).toBe('basic')
    expect(result.source?.name).toBe('Nested')
  })

  it('skips inherit folders to root collection', () => {
    const result = resolveRequestInheritedAuth('folder', { type: 'inherit' }, collections)
    expect(result.auth.type).toBe('bearer')
    expect(result.source?.name).toBe('API')
    expect(inheritSourceLabel(result.source!)).toContain('collection "API"')
  })

  it('request none does not inherit', () => {
    const result = resolveRequestInheritedAuth('nested', { type: 'none' }, collections)
    expect(result.auth.type).toBe('none')
  })

  it('does not loop on parent_id cycles', () => {
    const cyclic = [
      col('a', 'A', 'b', { type: 'bearer', config: { token: 't' } }),
      col('b', 'B', 'a', { type: 'inherit' }),
    ]
    const result = resolveRequestInheritedAuth('a', { type: 'inherit' }, cyclic)
    expect(result.auth.type).toBe('bearer')
  })
})
