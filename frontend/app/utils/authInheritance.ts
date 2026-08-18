import type { Collection } from '~/stores/collections'

export interface AuthSpec {
  type: string
  config?: Record<string, string>
}

export interface InheritSource {
  name: string
  kind: 'collection' | 'folder'
  auth: AuthSpec
}

function authLabel(auth: AuthSpec): string {
  switch (auth.type) {
    case 'basic':
      return 'Basic Auth'
    case 'bearer':
      return 'Bearer Token'
    case 'apikey':
      return 'API Key'
    case 'none':
      return 'No Auth'
    default:
      return auth.type
  }
}

export function inheritSourceLabel(source: InheritSource): string {
  const kind = source.kind === 'folder' ? 'folder' : 'collection'
  return `Inheriting ${authLabel(source.auth)} from ${kind} "${source.name}"`
}

function resolveAuthUpChain(
  startCollectionId: string | null | undefined,
  collections: Collection[],
): { auth: AuthSpec; source?: InheritSource } {
  const visited = new Set<string>()
  let cur = startCollectionId
  while (cur && !visited.has(cur)) {
    visited.add(cur)
    const col = collections.find(c => c.id === cur)
    if (!col) break
    const colAuth = col.auth || { type: 'inherit' }
    if (colAuth.type && colAuth.type !== 'inherit') {
      return {
        auth: colAuth,
        source: {
          name: col.name,
          kind: col.parent_id ? 'folder' : 'collection',
          auth: colAuth,
        },
      }
    }
    cur = col.parent_id ?? undefined
  }
  return { auth: { type: 'none' } }
}

export function resolveRequestInheritedAuth(
  requestCollectionId: string,
  requestAuth: AuthSpec,
  collections: Collection[],
): { auth: AuthSpec; source?: InheritSource } {
  if (requestAuth.type && requestAuth.type !== 'inherit') {
    return { auth: requestAuth }
  }
  return resolveAuthUpChain(requestCollectionId, collections)
}

export function resolveCollectionInheritedAuth(
  collection: Collection,
  collections: Collection[],
): { auth: AuthSpec; source?: InheritSource } {
  if (collection.auth?.type && collection.auth.type !== 'inherit') {
    return { auth: collection.auth }
  }
  return resolveAuthUpChain(collection.parent_id, collections)
}
