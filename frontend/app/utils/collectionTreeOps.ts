import type { Collection, RequestItem } from '~/stores/collections'
import type { TreeEntry } from '~/utils/treeEntries'

/** True when candidateId is ancestorId or nested under it. */
export function isCollectionDescendant(
  collections: Collection[],
  candidateId: string,
  ancestorId: string,
): boolean {
  if (candidateId === ancestorId) return true
  const visited = new Set<string>()
  let cur = collections.find(c => c.id === candidateId)
  while (cur?.parent_id) {
    if (cur.parent_id === ancestorId) return true
    if (visited.has(cur.id)) return false
    visited.add(cur.id)
    cur = collections.find(c => c.id === cur!.parent_id)
  }
  return false
}

export function parseTreeParentId(el: HTMLElement | null | undefined): string | null {
  if (!el) return null
  const container = el.closest('[data-collection-id]')
  if (!(container instanceof HTMLElement)) return null
  const raw = container.getAttribute('data-collection-id')
  return raw && raw.length > 0 ? raw : null
}

export function parseDraggedItem(el: HTMLElement | null | undefined): { id: string; type: 'folder' | 'request' } | null {
  if (!el) return null
  const itemEl = el.classList.contains('tree-draggable-item')
    ? el
    : el.closest('.tree-draggable-item')
  if (!(itemEl instanceof HTMLElement)) return null
  const id = itemEl.getAttribute('data-item-id')
  const type = itemEl.getAttribute('data-item-type')
  if (!id || (type !== 'folder' && type !== 'request')) return null
  return { id, type }
}

export type TreeDragItem = { id: string; type: 'folder' | 'request' }

export function dragItemsFromContainer(el: HTMLElement): TreeDragItem[] {
  const items: TreeDragItem[] = []
  for (const child of el.querySelectorAll(':scope > .tree-draggable-item')) {
    if (!(child instanceof HTMLElement)) continue
    const parsed = parseDraggedItem(child)
    if (parsed) items.push(parsed)
  }
  return items
}

export function getFolderChildItems(
  store: { collections: Collection[]; requests: RequestItem[] },
  folderId: string,
): TreeDragItem[] {
  const combined: { sort_order: number; item: TreeDragItem }[] = []
  for (const col of store.collections) {
    if (col.parent_id === folderId) {
      combined.push({ sort_order: col.sort_order ?? 0, item: { id: col.id, type: 'folder' } })
    }
  }
  for (const req of store.requests) {
    if (req.collection_id === folderId && !req.template_id) {
      combined.push({ sort_order: req.sort_order ?? 0, item: { id: req.id, type: 'request' } })
    }
  }
  combined.sort((a, b) => a.sort_order - b.sort_order)
  return combined.map(row => row.item)
}

export function getRelatedFolderId(related: HTMLElement | null | undefined): string | null {
  if (!related) return null
  const header = related.closest('[data-drop-folder-id]')
  if (header instanceof HTMLElement) {
    const folderId = header.getAttribute('data-drop-folder-id')
    if (folderId) return folderId
  }
  const item = related.closest('.tree-draggable-item[data-item-type="folder"]')
  if (item instanceof HTMLElement) {
    return item.getAttribute('data-item-id')
  }
  return null
}

export interface DragPersistPlan {
  destination: { parentId: string | null; items: TreeDragItem[] }
  source?: { parentId: string | null; items: TreeDragItem[] }
}

export function planDragPersistence(input: {
  fromEl: HTMLElement
  toEl: HTMLElement
  itemEl: HTMLElement
  relatedEl: HTMLElement | null
  hoverFolderId: string | null
  oldIndex: number
  newIndex: number
  collections: Collection[]
  requests: RequestItem[]
}): DragPersistPlan | 'invalid' | null {
  const dragged = parseDraggedItem(input.itemEl)
  if (!dragged) return null

  const fromParentId = parseTreeParentId(input.fromEl)
  const toParentId = parseTreeParentId(input.toEl)
  const relatedFolderId = getRelatedFolderId(input.relatedEl)

  let destinationParentId = input.hoverFolderId ?? toParentId

  if (!input.hoverFolderId && relatedFolderId && relatedFolderId !== dragged.id) {
    const relatedCol = input.collections.find(c => c.id === relatedFolderId)
    if (relatedCol) {
      const relatedParent = relatedCol.parent_id ?? null
      if (relatedParent === toParentId) {
        destinationParentId = relatedFolderId
      }
    }
  }

  if (dragged.type === 'folder' && destinationParentId && isInvalidFolderDrop(input.collections, dragged.id, destinationParentId)) {
    return 'invalid'
  }
  if (dragged.type === 'request' && !destinationParentId) {
    return 'invalid'
  }

  const parentChanged = destinationParentId !== fromParentId
  const nestIntoFolder = Boolean(destinationParentId && parentChanged)

  let destinationItems: TreeDragItem[]
  if (nestIntoFolder && destinationParentId) {
    destinationItems = getFolderChildItems(
      { collections: input.collections, requests: input.requests },
      destinationParentId,
    ).filter(item => item.id !== dragged.id)
    destinationItems.push(dragged)
  } else if (destinationParentId) {
    destinationItems = dragItemsFromContainer(input.toEl)
  } else {
    destinationItems = dragItemsFromContainer(input.toEl).filter(item => item.type === 'folder')
  }

  if (
    !parentChanged
    && input.fromEl === input.toEl
    && input.hoverFolderId == null
    && input.oldIndex === input.newIndex
  ) {
    return null
  }

  let source: DragPersistPlan['source']
  if (input.fromEl !== input.toEl || parentChanged) {
    source = {
      parentId: fromParentId,
      items: dragItemsFromContainer(input.fromEl).filter(item => item.id !== dragged.id),
    }
  }

  return {
    destination: { parentId: destinationParentId, items: destinationItems },
    source,
  }
}

export async function executeDragPersistence(
  store: CollectionsStoreLike,
  plan: DragPersistPlan,
) {
  const tasks: Promise<void>[] = []

  if (plan.destination.parentId) {
    tasks.push(persistFolderChildren(store, plan.destination.parentId, plan.destination.items))
  } else {
    tasks.push(persistRootFolders(
      store,
      plan.destination.items.filter(item => item.type === 'folder').map(item => item.id),
    ))
  }

  if (plan.source) {
    if (plan.source.parentId) {
      tasks.push(persistFolderChildren(store, plan.source.parentId, plan.source.items))
    } else {
      tasks.push(persistRootFolders(
        store,
        plan.source.items.filter(item => item.type === 'folder').map(item => item.id),
      ))
    }
  }

  await Promise.all(tasks)
}

interface CollectionsStoreLike {
  collections: Collection[]
  requests: RequestItem[]
  reorderCollections: (items: { id: string; parent_id?: string | null; sort_order: number }[]) => Promise<void>
  moveRequest: (id: string, collectionId: string, sortOrder: number) => Promise<void>
  reorderRequests: (items: { id: string; sort_order: number }[]) => Promise<void>
}

export async function persistFolderChildren(
  store: CollectionsStoreLike,
  parentCollectionId: string,
  items: TreeDragItem[],
) {
  const collectionUpdates: { id: string; parent_id: string | null; sort_order: number }[] = []
  const requestUpdates: { id: string; sort_order: number }[] = []
  const requestMoves: { id: string; collectionId: string; sortOrder: number }[] = []

  for (let index = 0; index < items.length; index++) {
    const item = items[index]
    if (item.type === 'folder') {
      collectionUpdates.push({
        id: item.id,
        parent_id: parentCollectionId,
        sort_order: index,
      })
      const col = store.collections.find(c => c.id === item.id)
      if (col) {
        col.parent_id = parentCollectionId
        col.sort_order = index
      }
    } else {
      const request = store.requests.find(r => r.id === item.id)
      if (request?.collection_id === parentCollectionId) {
        requestUpdates.push({ id: item.id, sort_order: index })
        if (request) request.sort_order = index
      } else {
        requestMoves.push({ id: item.id, collectionId: parentCollectionId, sortOrder: index })
        if (request) {
          request.collection_id = parentCollectionId
          request.sort_order = index
        }
      }
    }
  }

  if (collectionUpdates.length > 0) {
    await store.reorderCollections(collectionUpdates)
  }
  if (requestUpdates.length > 0) {
    await store.reorderRequests(requestUpdates)
  }
  for (const move of requestMoves) {
    await store.moveRequest(move.id, move.collectionId, move.sortOrder)
  }
}

export async function persistFolderChildrenFromEntries(
  store: CollectionsStoreLike,
  parentCollectionId: string,
  entries: TreeEntry[],
) {
  const items: TreeDragItem[] = entries.map(entry => ({
    id: entry.kind === 'folder' ? entry.node.id : entry.request.id,
    type: entry.kind === 'folder' ? 'folder' : 'request',
  }))
  await persistFolderChildren(store, parentCollectionId, items)
}

export async function persistRootFolders(
  store: CollectionsStoreLike,
  folderIds: string[],
) {
  const updates = folderIds.map((id, index) => ({
    id,
    parent_id: null as string | null,
    sort_order: index,
  }))
  for (const item of updates) {
    const col = store.collections.find(c => c.id === item.id)
    if (col) {
      col.parent_id = null
      col.sort_order = item.sort_order
    }
  }
  if (updates.length > 0) {
    await store.reorderCollections(updates)
  }
}

export function isInvalidFolderDrop(
  collections: Collection[],
  draggedFolderId: string,
  targetParentId: string | null,
): boolean {
  if (!targetParentId) return false
  if (targetParentId === draggedFolderId) return true
  return isCollectionDescendant(collections, targetParentId, draggedFolderId)
}
