import type { Collection } from '~/stores/collections'
import type { EnvVariable } from '~/stores/environments'

export type VarSource = 'builtin' | 'workspace' | 'collection' | 'environment'

export interface VarSuggestion {
  name: string
  source: VarSource
  value?: string
  description?: string
}

const BUILT_IN_VARS: VarSuggestion[] = [
  { name: '$timestamp', source: 'builtin', description: 'Unix timestamp (seconds)' },
  { name: '$isoTimestamp', source: 'builtin', description: 'ISO 8601 timestamp' },
]

export function collectionAncestorChain(collections: Collection[], collectionId?: string): Collection[] {
  if (!collectionId) return []
  const chain: Collection[] = []
  let cur = collections.find(c => c.id === collectionId)
  while (cur) {
    chain.unshift(cur)
    if (!cur.parent_id) break
    cur = collections.find(c => c.id === cur!.parent_id)
  }
  return chain
}

export function buildVarSuggestions(options: {
  workspaceVars?: Record<string, unknown> | null
  collections: Collection[]
  collectionId?: string
  envVariables?: EnvVariable[]
}): VarSuggestion[] {
  const out: VarSuggestion[] = []
  const seen = new Set<string>()

  const add = (item: VarSuggestion) => {
    if (!item.name || seen.has(item.name)) return
    seen.add(item.name)
    out.push(item)
  }

  for (const v of BUILT_IN_VARS) add(v)

  if (options.workspaceVars) {
    for (const [key, val] of Object.entries(options.workspaceVars)) {
      add({ name: key, source: 'workspace', value: String(val) })
    }
  }

  for (const col of collectionAncestorChain(options.collections, options.collectionId)) {
    for (const row of col.variables?.pre_request || []) {
      if (!row.enabled || !row.name) continue
      add({
        name: row.name,
        source: 'collection',
        value: row.secret ? undefined : row.value,
        description: row.description,
      })
    }
  }

  for (const v of options.envVariables || []) {
    if (!v.enabled || v.phase !== 'pre_request' || !v.key) continue
    add({
      name: v.key,
      source: 'environment',
      value: v.is_secret ? undefined : v.value,
      description: v.description,
    })
  }

  return out
}

export function filterVarSuggestions(suggestions: VarSuggestion[], query: string): VarSuggestion[] {
  const q = query.trim().toLowerCase()
  if (!q) return suggestions
  return suggestions.filter(s =>
    s.name.toLowerCase().includes(q) || s.description?.toLowerCase().includes(q),
  )
}
