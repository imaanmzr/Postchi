export type DetectedImportFormat = 'bruno' | 'postman' | 'opencollection' | 'openapi' | 'unknown'

export function detectImportFormat(name: string, text: string): DetectedImportFormat {
  const lower = name.toLowerCase()
  if (lower.endsWith('.zip') || lower.endsWith('.bru')) return 'bruno'

  const trimmed = text.trim()
  if (trimmed.startsWith('{')) {
    try {
      const json = JSON.parse(trimmed)
      if (json.opencollection) return 'opencollection'
      const schema = json.info?.schema || ''
      if (typeof schema === 'string' && schema.includes('postman.com/json/collection')) return 'postman'
      if (json.openapi || json.swagger) return 'openapi'
    } catch {
      // fall through
    }
  }

  if (/^opencollection\s*:/m.test(trimmed)) return 'opencollection'
  if (/^openapi\s*:/m.test(trimmed) || /^swagger\s*:/m.test(trimmed)) return 'openapi'

  if (lower.endsWith('.yml') || lower.endsWith('.yaml')) {
    return trimmed ? 'opencollection' : 'opencollection'
  }
  if (lower.endsWith('.json')) return 'postman'
  return 'postman'
}
