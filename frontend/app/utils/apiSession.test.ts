import { describe, expect, it, vi } from 'vitest'
import { retryOnUnauthorized, shouldLogoutOnUnauthorized } from './apiSession'

describe('shouldLogoutOnUnauthorized', () => {
  it('logs out when refresh did not succeed', () => {
    expect(shouldLogoutOnUnauthorized(false)).toBe(true)
  })

  it('does not log out when refresh succeeded but the endpoint still returned 401', () => {
    expect(shouldLogoutOnUnauthorized(true)).toBe(false)
  })
})

describe('retryOnUnauthorized', () => {
  it('retries once after a successful refresh', async () => {
    const calls: string[] = []
    const refresh = vi.fn(async () => {
      calls.push('refresh')
      return true
    })
    const execute = vi.fn(async () => {
      if (calls.length === 0) {
        calls.push('first-401')
        return new Response(JSON.stringify({ error: 'git auth failed' }), { status: 401 })
      }
      calls.push('retry-200')
      return new Response(JSON.stringify({ ok: true }), { status: 200 })
    })

    const { response, refreshSucceeded } = await retryOnUnauthorized(execute, refresh)

    expect(refreshSucceeded).toBe(true)
    expect(response.status).toBe(200)
    expect(refresh).toHaveBeenCalledTimes(1)
    expect(execute).toHaveBeenCalledTimes(2)
    expect(shouldLogoutOnUnauthorized(refreshSucceeded)).toBe(false)
  })

  it('does not retry when refresh fails', async () => {
    const refresh = vi.fn(async () => false)
    const execute = vi.fn(async () => new Response(null, { status: 401 }))

    const { response, refreshSucceeded } = await retryOnUnauthorized(execute, refresh)

    expect(refreshSucceeded).toBe(false)
    expect(response.status).toBe(401)
    expect(execute).toHaveBeenCalledTimes(1)
    expect(shouldLogoutOnUnauthorized(refreshSucceeded)).toBe(true)
  })
})
