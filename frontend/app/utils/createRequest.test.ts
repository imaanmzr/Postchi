import { describe, expect, it, vi } from 'vitest'
import type { Collection } from '~/stores/collections'
import {
  createFolderAtTarget,
  createRequestAtTarget,
  resolveFolderParentId,
} from './createRequest'

function mockStore(collections: Collection[] = []) {
  return {
    collections,
    requests: [],
    createCollection: vi.fn(async (_ws: string, data: Partial<Collection>) => {
      const col: Collection = {
        id: 'new-col',
        workspace_id: 'ws',
        name: data.name || 'New Folder',
        parent_id: data.parent_id ?? null,
        sort_order: 0,
      }
      collections.push(col)
      return col
    }),
    saveRequest: vi.fn(async (req: { collection_id: string }) => ({
      id: 'new-req',
      collection_id: req.collection_id,
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
    })),
  }
}

describe('resolveFolderParentId', () => {
  it('returns null for workspace target', () => {
    expect(resolveFolderParentId('workspace')).toBeNull()
  })

  it('returns folder id for nested target', () => {
    expect(resolveFolderParentId('folder-1')).toBe('folder-1')
  })
})

describe('createFolderAtTarget', () => {
  it('creates a top-level folder at workspace root', async () => {
    const store = mockStore()
    await createFolderAtTarget(store as any, 'ws', 'workspace')
    expect(store.createCollection).toHaveBeenCalledWith('ws', { name: 'New Folder' })
  })

  it('creates a nested folder under selected parent', async () => {
    const store = mockStore()
    await createFolderAtTarget(store as any, 'ws', 'parent-id')
    expect(store.createCollection).toHaveBeenCalledWith('ws', {
      name: 'New Folder',
      parent_id: 'parent-id',
    })
  })
})

describe('createRequestAtTarget', () => {
  it('creates a standalone root folder and request at workspace level', async () => {
    const store = mockStore()
    const saved = await createRequestAtTarget(store as any, 'ws', 'workspace')
    expect(store.createCollection).toHaveBeenCalledWith('ws', { name: 'New Folder' })
    expect(store.saveRequest).toHaveBeenCalledWith(expect.objectContaining({ collection_id: 'new-col' }))
    expect(saved.collection_id).toBe('new-col')
  })

  it('creates a request inside the selected folder', async () => {
    const store = mockStore([{ id: 'folder-1', workspace_id: 'ws', name: 'A', sort_order: 0 }])
    await createRequestAtTarget(store as any, 'ws', 'folder-1')
    expect(store.createCollection).not.toHaveBeenCalled()
    expect(store.saveRequest).toHaveBeenCalledWith(expect.objectContaining({ collection_id: 'folder-1' }))
  })
})
