export type WorkspaceType = 'default' | 'pm' | 'tester'

export const WORKSPACE_TYPES: { id: WorkspaceType, label: string, description: string }[] = [
  {
    id: 'default',
    label: 'Default',
    description: 'API requests, collections, and documentation.',
  },
  {
    id: 'pm',
    label: 'Product Manager',
    description: 'User story diagrams and markdown docs.',
  },
  {
    id: 'tester',
    label: 'Tester',
    description: 'Test cases linked to API requests.',
  },
]

export function workspaceTypeLabel(type?: string): string {
  return WORKSPACE_TYPES.find(t => t.id === type)?.label ?? 'Default'
}

export function isWorkspaceType(value: string): value is WorkspaceType {
  return value === 'default' || value === 'pm' || value === 'tester'
}
