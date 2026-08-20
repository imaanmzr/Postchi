import type { Collection } from '~/stores/collections'
import { buildTree, type TreeNode } from '~/stores/collections'

export type ImportParentMode = 'root' | 'existing' | 'new'

export interface ImportParentChoice {
  mode: ImportParentMode
  parentId?: string
  newParentName?: string
}

const storageKey = (workspaceId: string) => `postchi:git-import-parent:${workspaceId}`

export function loadImportParentChoice(workspaceId: string): ImportParentChoice | null {
  if (!import.meta.client) return null
  try {
    const raw = localStorage.getItem(storageKey(workspaceId))
    if (!raw) return null
    const parsed = JSON.parse(raw) as ImportParentChoice
    if (parsed.mode !== 'root' && parsed.mode !== 'existing' && parsed.mode !== 'new') return null
    return parsed
  } catch {
    return null
  }
}

export function saveImportParentChoice(workspaceId: string, choice: ImportParentChoice) {
  if (!import.meta.client) return
  localStorage.setItem(storageKey(workspaceId), JSON.stringify(choice))
}

export function importParentPayload(choice: ImportParentChoice): Record<string, unknown> {
  if (choice.mode === 'existing' && choice.parentId) {
    return { parent_id: choice.parentId }
  }
  if (choice.mode === 'new') {
    const name = choice.newParentName?.trim()
    if (name) return { create_parent: { name } }
  }
  return {}
}

export function isImportParentValid(choice: ImportParentChoice): boolean {
  if (choice.mode === 'existing') return !!choice.parentId
  if (choice.mode === 'new') return !!choice.newParentName?.trim()
  return true
}

export interface ImportParentOption {
  id: string
  label: string
  depth: number
}

export function flattenCollectionOptions(collections: Collection[]): ImportParentOption[] {
  const tree = buildTree(collections)
  const out: ImportParentOption[] = []

  function walk(nodes: TreeNode[], depth: number) {
    for (const node of nodes) {
      out.push({
        id: node.id,
        label: node.name,
        depth,
      })
      if (node.children.length) walk(node.children, depth + 1)
    }
  }

  walk(tree, 0)
  return out
}

export function indentLabel(label: string, depth: number): string {
  return `${'  '.repeat(depth)}${label}`
}
