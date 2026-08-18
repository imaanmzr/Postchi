import { describe, expect, it } from 'vitest'
import type { RequestItem } from './collections'
import { extractPlaceholdersFromRequest } from '~/utils/placeholders'

function req(id: string, colId: string, templateId?: string): RequestItem {
  return {
    id,
    collection_id: colId,
    name: id,
    method: 'GET',
    url: templateId ? '' : 'https://example.com',
    headers: [],
    params: [],
    body: { mode: 'none', raw: '', raw_lang: 'json' },
    auth: { type: 'none' },
    settings: { timeout_ms: 30000, follow_redirects: true, verify_ssl: true },
    pre_request_script: '',
    test_script: '',
    template_id: templateId,
  }
}

function requestsByCollection(requests: RequestItem[]) {
  const map: Record<string, RequestItem[]> = {}
  for (const r of requests) {
    if (r.template_id) continue
    if (!map[r.collection_id]) map[r.collection_id] = []
    map[r.collection_id].push(r)
  }
  return map
}

function variantsByTemplate(requests: RequestItem[]) {
  const map: Record<string, RequestItem[]> = {}
  for (const r of requests) {
    if (!r.template_id) continue
    if (!map[r.template_id]) map[r.template_id] = []
    map[r.template_id].push(r)
  }
  return map
}

describe('requestsByCollection', () => {
  it('groups top-level requests by collection including nested folders', () => {
    const requests = [
      req('a', 'col-parent'),
      req('b', 'col-child'),
      req('variant', 'col-parent', 'a'),
    ]
    const map = requestsByCollection(requests)
    expect(map['col-parent']?.map(r => r.id)).toEqual(['a'])
    expect(map['col-child']?.map(r => r.id)).toEqual(['b'])
  })

  it('excludes template variants from folder lists', () => {
    const requests = [req('tpl', 'col1'), req('v1', 'col1', 'tpl')]
    const map = requestsByCollection(requests)
    expect(map['col1']?.length).toBe(1)
    expect(map['col1'][0].id).toBe('tpl')
  })
})

describe('variantsByTemplate', () => {
  it('groups children under template id', () => {
    const requests = [req('tpl', 'col1'), req('v1', 'col1', 'tpl'), req('v2', 'col1', 'tpl')]
    const map = variantsByTemplate(requests)
    expect(map['tpl']?.map(r => r.id)).toEqual(['v1', 'v2'])
  })
})

describe('extractPlaceholdersFromRequest', () => {
  it('finds placeholders in url and headers', () => {
    const names = extractPlaceholdersFromRequest({
      url: 'https://{{host}}/api',
      headers: [{ key: 'Authorization', value: 'Bearer {{token}}', enabled: true }],
      params: [],
      body: { mode: 'raw', raw: '', raw_lang: 'json' },
      auth: { type: 'none' },
    })
    expect(names).toContain('host')
    expect(names).toContain('token')
  })
})
