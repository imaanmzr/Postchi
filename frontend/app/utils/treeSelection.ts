import type { Collection, RequestItem } from '~/stores/collections'

export type TreeSelectionKey = `collection:${string}` | `request:${string}`

export function treeSelectionKey(type: 'collection' | 'request', id: string): TreeSelectionKey {
  return `${type}:${id}`
}

export function allTreeSelectionKeys(
  collections: Collection[],
  requests: RequestItem[],
): TreeSelectionKey[] {
  const keys: TreeSelectionKey[] = []
  for (const col of collections) keys.push(treeSelectionKey('collection', col.id))
  for (const req of requests) keys.push(treeSelectionKey('request', req.id))
  return keys
}

/** Collections whose ancestors are not also selected (DB cascade deletes descendants). */
export function topLevelSelectedCollectionIds(
  selectedCollectionIds: Set<string>,
  collections: Collection[],
): string[] {
  const byId = new Map(collections.map(c => [c.id, c]))
  return [...selectedCollectionIds].filter((id) => {
    let cur = byId.get(id)
    while (cur?.parent_id) {
      if (selectedCollectionIds.has(cur.parent_id)) return false
      cur = byId.get(cur.parent_id)
    }
    return true
  })
}

/** All collection ids in the subtrees rooted at the given collection ids. */
export function collectionSubtreeIds(
  rootIds: string[],
  collections: Collection[],
): Set<string> {
  const childrenByParent = new Map<string, string[]>()
  for (const col of collections) {
    if (!col.parent_id) continue
    const list = childrenByParent.get(col.parent_id) ?? []
    list.push(col.id)
    childrenByParent.set(col.parent_id, list)
  }

  const result = new Set<string>()
  const stack = [...rootIds]
  while (stack.length) {
    const id = stack.pop()!
    if (result.has(id)) continue
    result.add(id)
    for (const childId of childrenByParent.get(id) ?? []) stack.push(childId)
  }
  return result
}
