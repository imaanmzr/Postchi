/** Parse a response body into a JSON value when possible. */
export function parseResponseJson(body: unknown): unknown | null {
  if (body == null) return null
  if (typeof body === 'object') return body
  if (typeof body !== 'string') return null

  const trimmed = body.trim()
  if (!trimmed.startsWith('{') && !trimmed.startsWith('[')) return null

  try {
    return JSON.parse(trimmed)
  } catch {
    return null
  }
}

/** Serialize a JSON value for clipboard copy. */
export function serializeJsonValue(value: unknown): string {
  if (value === null) return 'null'
  if (typeof value === 'string') return value
  if (typeof value === 'number' || typeof value === 'boolean') return String(value)
  return JSON.stringify(value, null, 2)
}
