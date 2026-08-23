import { describe, expect, it } from 'vitest'
import { buildClientErrorResponse, normalizeExecutionResult } from './executionResponse'

describe('normalizeExecutionResult', () => {
  it('formats backend network errors into a readable body', () => {
    const result = normalizeExecutionResult({
      status_code: 0,
      error: 'dial tcp 127.0.0.1:9: connect: connection refused',
      timing: { total_ms: 42 },
      headers: {},
    })

    expect(result.body).toContain('Could not reach the server')
    expect(result.body_size).toBeGreaterThan(0)
    expect(result.error).toContain('Could not reach the server')
    expect(result.timing.total_ms).toBe(42)
  })

  it('preserves successful responses', () => {
    const result = normalizeExecutionResult({
      status_code: 200,
      body: '{"ok":true}',
      body_size: 11,
      timing: { total_ms: 120 },
      headers: { 'content-type': 'application/json' },
    })

    expect(result.body).toBe('{"ok":true}')
    expect(result.status_code).toBe(200)
    expect(result.error).toBeUndefined()
  })
})

describe('buildClientErrorResponse', () => {
  it('maps fetch failures to a network error response', () => {
    const result = buildClientErrorResponse(new TypeError('Failed to fetch'), 88)

    expect(result.status_code).toBe(0)
    expect(result.error).toContain('Postchi server')
    expect(result.body).toContain('network_error')
    expect(result.timing.total_ms).toBe(88)
  })
})
