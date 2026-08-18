import { describe, expect, it, vi } from 'vitest'
import type { Collection, RequestItem } from '~/stores/collections'
import {
  dragItemsFromContainer,
  executeDragPersistence,
  getRelatedFolderId,
  isCollectionDescendant,
  isInvalidFolderDrop,
  planDragPersistence,
} from './collectionTreeOps'

const collections: Collection[] = [
  { id: 'root', workspace_id: 'ws', name: 'deduct-wallet-balance', sort_order: 0 },
  { id: 'child-folder', workspace_id: 'ws', name: 'New Folder', parent_id: 'root', sort_order: 0 },
  { id: 'nested', workspace_id: 'ws', name: 'Nested', parent_id: 'child-folder', sort_order: 0 },
]

const requests: RequestItem[] = [
  {
    id: 'req-post',
    collection_id: 'root',
    name: 'deduct-wallet-balance',
    method: 'POST',
    url: '',
    headers: [],
    params: [],
    body: { mode: 'none', raw: '', raw_lang: 'json' },
    auth: { type: 'inherit' },
    settings: { timeout_ms: 30000, follow_redirects: true, verify_ssl: true },
    pre_request_script: '',
    test_script: '',
    sort_order: 1,
  },
  {
    id: 'req-get',
    collection_id: 'root',
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
    sort_order: 2,
  },
]

function makeList(parentId: string, items: { id: string; type: 'folder' | 'request' }[]) {
  const container = document.createElement('div')
  container.className = 'tree-drop-container'
  container.setAttribute('data-collection-id', parentId)
  const list = document.createElement('div')
  list.className = 'tree-draggable-list'
  container.appendChild(list)
  for (const item of items) {
    const el = document.createElement('div')
    el.className = 'tree-draggable-item'
    el.setAttribute('data-item-id', item.id)
    el.setAttribute('data-item-type', item.type)
    if (item.type === 'folder') {
      const header = document.createElement('div')
      header.className = 'tree-folder-row'
      header.setAttribute('data-drop-folder-id', item.id)
      el.appendChild(header)
    }
    list.appendChild(el)
  }
  return { container, list }
}

function draggedItem(id: string, type: 'folder' | 'request') {
  const el = document.createElement('div')
  el.className = 'tree-draggable-item'
  el.setAttribute('data-item-id', id)
  el.setAttribute('data-item-type', type)
  return el
}

describe('isCollectionDescendant', () => {
  it('detects direct and nested descendants', () => {
    expect(isCollectionDescendant(collections, 'child-folder', 'root')).toBe(true)
    expect(isCollectionDescendant(collections, 'nested', 'root')).toBe(true)
    expect(isCollectionDescendant(collections, 'root', 'child-folder')).toBe(false)
  })

  it('does not loop on parent_id cycles', () => {
    const cyclic: Collection[] = [
      { id: 'a', workspace_id: 'ws', name: 'A', parent_id: 'b', sort_order: 0 },
      { id: 'b', workspace_id: 'ws', name: 'B', parent_id: 'a', sort_order: 0 },
    ]
    expect(isCollectionDescendant(cyclic, 'b', 'a')).toBe(true)
    expect(isCollectionDescendant(cyclic, 'a', 'b')).toBe(true)
  })
})

describe('isInvalidFolderDrop', () => {
  it('blocks dropping into self or descendants', () => {
    expect(isInvalidFolderDrop(collections, 'child-folder', 'child-folder')).toBe(true)
    expect(isInvalidFolderDrop(collections, 'child-folder', 'nested')).toBe(true)
    expect(isInvalidFolderDrop(collections, 'child-folder', 'root')).toBe(false)
  })
})

describe('dragItemsFromContainer', () => {
  it('reads draggable items from container wrapper', () => {
    const { list } = makeList('root', [
      { id: 'child-folder', type: 'folder' },
      { id: 'req-get', type: 'request' },
    ])
    expect(dragItemsFromContainer(list)).toEqual([
      { id: 'child-folder', type: 'folder' },
      { id: 'req-get', type: 'request' },
    ])
  })
})

describe('getRelatedFolderId', () => {
  it('finds folder id from folder header', () => {
    const { list } = makeList('root', [{ id: 'child-folder', type: 'folder' }])
    const folderHeader = list.querySelector('[data-drop-folder-id]') as HTMLElement
    expect(getRelatedFolderId(folderHeader)).toBe('child-folder')
  })
})

describe('planDragPersistence', () => {
  it('nests a request under a sibling folder when dropped on the folder row', () => {
    const { list } = makeList('root', [
      { id: 'child-folder', type: 'folder' },
      { id: 'req-post', type: 'request' },
      { id: 'req-get', type: 'request' },
    ])
    const folderHeader = list.querySelector('[data-drop-folder-id]') as HTMLElement
    const item = draggedItem('req-get', 'request')

    const plan = planDragPersistence({
      fromEl: list,
      toEl: list,
      itemEl: item,
      relatedEl: folderHeader,
      hoverFolderId: null,
      oldIndex: 2,
      newIndex: 1,
      collections,
      requests,
    })

    expect(plan).not.toBe('invalid')
    expect(plan).toMatchObject({
      destination: {
        parentId: 'child-folder',
        items: [
          { id: 'nested', type: 'folder' },
          { id: 'req-get', type: 'request' },
        ],
      },
      source: {
        parentId: 'root',
        items: [
          { id: 'child-folder', type: 'folder' },
          { id: 'req-post', type: 'request' },
        ],
      },
    })
  })

  it('moves a request into a folder when hoverFolderId is set before drag end', () => {
    const from = makeList('root', [
      { id: 'child-folder', type: 'folder' },
      { id: 'req-get', type: 'request' },
    ])
    const to = makeList('child-folder', [])
    const item = draggedItem('req-get', 'request')

    const plan = planDragPersistence({
      fromEl: from.list,
      toEl: to.list,
      itemEl: item,
      relatedEl: null,
      hoverFolderId: 'child-folder',
      oldIndex: 1,
      newIndex: 0,
      collections,
      requests,
    })

    expect(plan).toMatchObject({
      destination: {
        parentId: 'child-folder',
        items: [
          { id: 'nested', type: 'folder' },
          { id: 'req-get', type: 'request' },
        ],
      },
      source: {
        parentId: 'root',
        items: [{ id: 'child-folder', type: 'folder' }],
      },
    })
  })

  it('moves a folder under another folder', () => {
    const from = makeList('root', [
      { id: 'child-folder', type: 'folder' },
      { id: 'nested', type: 'folder' },
    ])
    const to = makeList('child-folder', [])
    const item = draggedItem('nested', 'folder')
    const folderHeader = from.list.querySelector('[data-drop-folder-id]') as HTMLElement

    const plan = planDragPersistence({
      fromEl: from.list,
      toEl: to.list,
      itemEl: item,
      relatedEl: folderHeader,
      hoverFolderId: 'child-folder',
      oldIndex: 1,
      newIndex: 0,
      collections,
      requests,
    })

    expect(plan).toMatchObject({
      destination: {
        parentId: 'child-folder',
        items: [
          { id: 'nested', type: 'folder' },
        ],
      },
      source: {
        parentId: 'root',
        items: [{ id: 'child-folder', type: 'folder' }],
      },
    })
  })

  it('reorders within the same folder without changing parent', () => {
    const { list } = makeList('root', [
      { id: 'req-get', type: 'request' },
      { id: 'req-post', type: 'request' },
    ])
    const item = list.children[0] as HTMLElement

    const plan = planDragPersistence({
      fromEl: list,
      toEl: list,
      itemEl: item,
      relatedEl: list.children[1] as HTMLElement,
      hoverFolderId: null,
      oldIndex: 1,
      newIndex: 0,
      collections,
      requests,
    })

    expect(plan).toMatchObject({
      destination: {
        parentId: 'root',
        items: [
          { id: 'req-get', type: 'request' },
          { id: 'req-post', type: 'request' },
        ],
      },
    })
    expect(plan && plan !== 'invalid' && plan.source).toBeUndefined()
  })

  it('returns null for no-op drags', () => {
    const { list } = makeList('root', [{ id: 'req-get', type: 'request' }])
    const item = draggedItem('req-get', 'request')
    const plan = planDragPersistence({
      fromEl: list,
      toEl: list,
      itemEl: item,
      relatedEl: item,
      hoverFolderId: null,
      oldIndex: 0,
      newIndex: 0,
      collections,
      requests,
    })
    expect(plan).toBeNull()
  })

  it('rejects circular folder nesting', () => {
    const { list } = makeList('root', [{ id: 'child-folder', type: 'folder' }])
    const item = draggedItem('child-folder', 'folder')
    const plan = planDragPersistence({
      fromEl: list,
      toEl: makeList('nested', []).list,
      itemEl: item,
      relatedEl: null,
      hoverFolderId: 'nested',
      oldIndex: 0,
      newIndex: 0,
      collections,
      requests,
    })
    expect(plan).toBe('invalid')
  })
})

describe('executeDragPersistence', () => {
  it('calls moveRequest when nesting a request under a folder', async () => {
    const store = {
      collections: [...collections],
      requests: [...requests],
      reorderCollections: vi.fn().mockResolvedValue(undefined),
      reorderRequests: vi.fn().mockResolvedValue(undefined),
      moveRequest: vi.fn().mockResolvedValue(undefined),
    }

    await executeDragPersistence(store, {
      destination: {
        parentId: 'child-folder',
        items: [{ id: 'req-get', type: 'request' }],
      },
      source: {
        parentId: 'root',
        items: [
          { id: 'child-folder', type: 'folder' },
          { id: 'req-post', type: 'request' },
        ],
      },
    })

    expect(store.moveRequest).toHaveBeenCalledWith('req-get', 'child-folder', 0)
    expect(store.reorderCollections).toHaveBeenCalled()
  })
})
