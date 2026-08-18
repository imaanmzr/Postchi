import { describe, expect, it } from 'vitest'
import { buildVarSuggestions, collectionAncestorChain, filterVarSuggestions } from './variableSuggestions'

describe('variableSuggestions', () => {
  const collections = [
    { id: 'root', workspace_id: 'ws', name: 'Root', sort_order: 0, variables: { pre_request: [{ enabled: true, name: 'baseUrl', value: 'https://root', type: 'string' }] } },
    { id: 'child', workspace_id: 'ws', parent_id: 'root', name: 'Child', sort_order: 0, variables: { pre_request: [{ enabled: true, name: 'token', value: 'abc', type: 'string' }] } },
  ]

  it('walks collection parent chain', () => {
    const chain = collectionAncestorChain(collections as any, 'child')
    expect(chain.map(c => c.id)).toEqual(['root', 'child'])
  })

  it('merges vars from workspace, collection chain, and environment', () => {
    const suggestions = buildVarSuggestions({
      workspaceVars: { wsKey: 'ws-val' },
      collections: collections as any,
      collectionId: 'child',
      envVariables: [{
        key: 'localBaseUrl',
        value: 'http://localhost:8080',
        phase: 'pre_request',
        enabled: true,
        type: 'string',
        description: '',
        is_secret: false,
      }],
    })
    const names = suggestions.map(s => s.name)
    expect(names).toContain('$timestamp')
    expect(names).toContain('wsKey')
    expect(names).toContain('baseUrl')
    expect(names).toContain('token')
    expect(names).toContain('localBaseUrl')
  })

  it('filters by partial name', () => {
    const all = buildVarSuggestions({
      collections: collections as any,
      collectionId: 'child',
    })
    const filtered = filterVarSuggestions(all, 'base')
    expect(filtered.some(s => s.name === 'baseUrl')).toBe(true)
    expect(filtered.some(s => s.name === 'token')).toBe(false)
  })
})
