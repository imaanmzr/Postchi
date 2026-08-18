export type ResponseBodyLang = 'json' | 'xml' | 'text'

function contentTypeFromHeaders(headers?: Record<string, string>): string {
  if (!headers) return ''
  const entry = Object.entries(headers).find(([k]) => k.toLowerCase() === 'content-type')
  return entry?.[1]?.toLowerCase() ?? ''
}

/** Infer syntax highlighting language from headers and body shape. */
export function detectResponseBodyLang(
  body: unknown,
  headers?: Record<string, string>,
): ResponseBodyLang {
  const contentType = contentTypeFromHeaders(headers)

  if (contentType.includes('json') || contentType.includes('+json')) return 'json'
  if (contentType.includes('xml') || contentType.includes('html')) return 'xml'

  if (body == null) return 'text'
  const text = typeof body === 'string' ? body.trim() : ''
  if (!text) return 'text'

  if (text.startsWith('{') || text.startsWith('[')) {
    try {
      JSON.parse(text)
      return 'json'
    } catch {
      // fall through
    }
  }

  if (text.startsWith('<')) return 'xml'

  return 'text'
}

/** Pretty-print JSON response bodies when the payload is valid JSON text. */
export function formatResponseBody(body: unknown): string {
  if (body == null) return ''
  if (typeof body === 'string') {
    const trimmed = body.trim()
    if (trimmed.startsWith('{') || trimmed.startsWith('[')) {
      try {
        return JSON.stringify(JSON.parse(trimmed), null, 2)
      } catch {
        return body
      }
    }
    return body
  }
  try {
    return JSON.stringify(body, null, 2)
  } catch {
    return String(body)
  }
}
