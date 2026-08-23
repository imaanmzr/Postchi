export interface ExecutionResponse {
  status_code: number
  body: string
  headers: Record<string, string>
  timing: Record<string, number>
  body_size: number
  error?: string
  test_results?: Array<{ name: string; passed: boolean; message?: string }>
  console?: string[]
  history_id?: string
}

function isNetworkErrorMessage(message: string) {
  const lower = message.toLowerCase()
  return lower.includes('failed to fetch')
    || lower.includes('networkerror')
    || lower.includes('network request failed')
    || lower.includes('load failed')
    || lower.includes('dial tcp')
    || lower.includes('connection refused')
    || lower.includes('no such host')
    || lower.includes('i/o timeout')
    || lower.includes('context deadline exceeded')
    || lower.includes('tls handshake')
    || lower.includes('connection reset')
}

function formatErrorBody(message: string, detail: string, type: string) {
  return JSON.stringify({ message, error: detail, type }, null, 2)
}

export function normalizeExecutionResult(
  result: Record<string, unknown>,
  clientElapsedMs?: number,
): ExecutionResponse {
  const timing = (result.timing as Record<string, number> | undefined) ?? {}
  const totalMs = timing.total_ms ?? clientElapsedMs ?? 0
  const error = typeof result.error === 'string' ? result.error : ''
  const existingBody = typeof result.body === 'string' ? result.body : ''
  const statusCode = typeof result.status_code === 'number' ? result.status_code : 0

  if (error && !existingBody) {
    const message = isNetworkErrorMessage(error)
      ? 'Could not reach the server. Check your network connection and try again.'
      : 'Request could not be completed.'
    const body = formatErrorBody(message, error, isNetworkErrorMessage(error) ? 'network_error' : 'request_error')
    return {
      status_code: statusCode,
      body,
      headers: (result.headers as Record<string, string>) ?? { 'Content-Type': 'application/json' },
      timing: { ...timing, total_ms: totalMs },
      body_size: body.length,
      error: message,
      test_results: (result.test_results as ExecutionResponse['test_results']) ?? [],
      console: (result.console as string[]) ?? [],
      history_id: typeof result.history_id === 'string' ? result.history_id : undefined,
    }
  }

  return {
    status_code: statusCode,
    body: existingBody,
    headers: (result.headers as Record<string, string>) ?? {},
    timing: { ...timing, total_ms: totalMs },
    body_size: typeof result.body_size === 'number'
      ? result.body_size
      : (existingBody ? existingBody.length : 0),
    error: error || undefined,
    test_results: (result.test_results as ExecutionResponse['test_results']) ?? [],
    console: (result.console as string[]) ?? [],
    history_id: typeof result.history_id === 'string' ? result.history_id : undefined,
  }
}

export function buildClientErrorResponse(err: unknown, elapsedMs: number): ExecutionResponse {
  const detail = err instanceof Error ? err.message : 'Request failed'
  const isNetwork = isNetworkErrorMessage(detail)
  const message = isNetwork
    ? 'Could not reach the Postchi server. Check your network connection and try again.'
    : detail
  const body = formatErrorBody(message, detail, isNetwork ? 'network_error' : 'client_error')

  return {
    status_code: 0,
    body,
    headers: { 'Content-Type': 'application/json' },
    timing: { total_ms: elapsedMs },
    body_size: body.length,
    error: message,
    test_results: [],
    console: [],
  }
}
