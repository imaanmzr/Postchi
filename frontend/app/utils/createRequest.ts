import type { useCollectionsStore } from '~/stores/collections'
import { blankRequest } from '~/utils/requestDefaults'

type CollectionsStore = ReturnType<typeof useCollectionsStore>

/** `workspace` = top-level (parent_id null for folders). */
export type CreationTarget = 'workspace' | string

export function resolveFolderParentId(target: CreationTarget): string | null | undefined {
  return target === 'workspace' ? null : target
}

export async function ensureTargetCollectionId(
  store: CollectionsStore,
  workspaceId: string,
  target: CreationTarget,
): Promise<string> {
  if (target !== 'workspace') {
    if (store.collections.some(c => c.id === target)) {
      return target
    }
  }

  const roots = store.collections.filter(c => !c.parent_id)
  if (roots.length === 1) return roots[0].id

  const created = await store.createCollection(workspaceId, { name: 'My Collection' })
  return created.id
}

export async function createFolderAtTarget(
  store: CollectionsStore,
  workspaceId: string,
  target: CreationTarget,
  name = 'New Folder',
) {
  const parentId = resolveFolderParentId(target)
  return store.createCollection(workspaceId, {
    name,
    ...(parentId ? { parent_id: parentId } : {}),
  })
}

/** At workspace root, create a standalone top-level folder then the request inside it. */
export async function createRequestAtTarget(
  store: CollectionsStore,
  workspaceId: string,
  target: CreationTarget,
) {
  if (target === 'workspace') {
    const col = await store.createCollection(workspaceId, { name: 'New Folder' })
    return store.saveRequest(blankRequest(col.id))
  }
  const collectionId = await ensureTargetCollectionId(store, workspaceId, target)
  return store.saveRequest(blankRequest(collectionId))
}
