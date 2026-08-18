import type { RequestItem } from '~/stores/collections'

export function blankRequest(collectionId: string): RequestItem {
  return {
    id: '',
    collection_id: collectionId,
    name: 'New Request',
    method: 'GET',
    url: '',
    headers: [],
    params: [],
    body: { mode: 'none', raw: '', raw_lang: 'json' },
    auth: { type: 'inherit' },
    settings: { timeout_ms: 30000, follow_redirects: true, verify_ssl: true },
    pre_request_script: '',
    test_script: '',
  }
}
