import type { ImportFormat } from '~/utils/importPlaceholders'
import { extractPlaceholders } from './placeholders'

const builtinPlaceholder = (name: string) => name.startsWith('$')

export function collectionVarNamesFromPostman(text: string): Set<string> {
  const names = new Set<string>()
  try {
    const json = JSON.parse(text)
    for (const v of json.variable || []) {
      if (v?.key) names.add(String(v.key))
    }
  } catch { /* ignore */ }
  return names
}

export function collectionVarNamesFromOpenAPI(text: string): Set<string> {
  const names = new Set<string>()
  try {
    const json = JSON.parse(text)
    for (const s of json.servers || []) {
      if (s?.url) {
        for (const n of extractPlaceholders(String(s.url))) names.add(n)
      }
    }
  } catch { /* ignore */ }
  return names
}

export function collectionVarNamesFromImport(format: Exclude<ImportFormat, 'auto'>, text: string): Set<string> {
  if (format === 'postman') return collectionVarNamesFromPostman(text)
  if (format === 'openapi') return collectionVarNamesFromOpenAPI(text)
  return new Set()
}

function walkPostmanItems(items: any[], out: string[]) {
  for (const item of items || []) {
    const req = item?.request
    if (req) {
      if (typeof req.url === 'string') out.push(req.url)
      else if (req.url?.raw) out.push(String(req.url.raw))
      for (const h of req.header || []) {
        if (!h?.disabled) out.push(String(h.value || ''))
      }
      if (req.body?.raw) out.push(String(req.body.raw))
      const auth = req.auth
      if (auth) {
        for (const block of [auth.bearer, auth.basic, auth.apikey]) {
          if (Array.isArray(block)) {
            for (const kv of block) out.push(String(kv?.value || ''))
          } else if (block && typeof block === 'object') {
            for (const v of Object.values(block)) out.push(String(v))
          }
        }
      }
    }
    if (item?.item) walkPostmanItems(item.item, out)
  }
}

export function requestFieldPlaceholdersFromPostman(text: string): string[] {
  try {
    const json = JSON.parse(text)
    const texts: string[] = []
    walkPostmanItems(json.item || [], texts)
    const seen = new Set<string>()
    for (const t of texts) {
      for (const n of extractPlaceholders(t)) {
        if (!builtinPlaceholder(n)) seen.add(n)
      }
    }
    return [...seen].sort()
  } catch {
    return []
  }
}

export function placeholdersNeedingEnvMapping(
  format: Exclude<ImportFormat, 'auto'>,
  text: string,
  existing: Iterable<string>,
): string[] {
  const existingSet = new Set(existing)
  const collectionVars = collectionVarNamesFromImport(format, text)
  const fromRequests = format === 'postman'
    ? requestFieldPlaceholdersFromPostman(text)
    : extractPlaceholders(text)
  const missing: string[] = []
  for (const name of fromRequests) {
    if (builtinPlaceholder(name)) continue
    if (collectionVars.has(name)) continue
    if (existingSet.has(name)) continue
    missing.push(name)
  }
  return missing
}

export type ImportFormat = 'auto' | 'bruno' | 'opencollection' | 'postman' | 'openapi'
