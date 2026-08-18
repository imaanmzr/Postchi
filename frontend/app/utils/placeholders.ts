import type { RequestItem } from '~/stores/collections'

const placeholderRe = /\{\{([^}]+)\}\}/g

export function extractPlaceholders(text: string): string[] {
  const seen = new Set<string>()
  let m: RegExpExecArray | null
  const re = new RegExp(placeholderRe.source, 'g')
  while ((m = re.exec(text)) !== null) {
    const name = m[1].trim()
    if (name) seen.add(name)
  }
  return [...seen].sort()
}

export function extractPlaceholdersFromRequest(req: Partial<RequestItem>): string[] {
  const texts: string[] = [req.url || '']
  for (const h of req.headers || []) {
    if (h.enabled) texts.push(h.value)
  }
  for (const p of req.params || []) {
    if (p.enabled) texts.push(p.value)
  }
  for (const pv of req.path_vars || []) {
    if (pv.enabled) texts.push(pv.value)
  }
  if (req.body?.raw) texts.push(req.body.raw)
  if (req.auth?.config) {
    for (const v of Object.values(req.auth.config)) texts.push(String(v))
  }
  const seen = new Set<string>()
  for (const t of texts) {
    for (const n of extractPlaceholders(t)) seen.add(n)
  }
  return [...seen].sort()
}
