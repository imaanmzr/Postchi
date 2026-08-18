import type { Collection, RequestItem, TreeNode } from '~/stores/collections'

export type TreeEntry =
  | { kind: 'folder'; sort_order: number; node: TreeNode }
  | { kind: 'request'; sort_order: number; request: RequestItem }

/** Merge child folders and requests into a single list sorted by sort_order. */
export function mergeTreeEntries(node: TreeNode, requests: RequestItem[]): TreeEntry[] {
  const entries: TreeEntry[] = [
    ...node.children.map(child => ({
      kind: 'folder' as const,
      sort_order: child.sort_order ?? 0,
      node: child,
    })),
    ...requests.map(request => ({
      kind: 'request' as const,
      sort_order: request.sort_order ?? 0,
      request,
    })),
  ]
  entries.sort((a, b) => a.sort_order - b.sort_order)
  return entries
}
