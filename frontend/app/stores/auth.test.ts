import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useAuthStore } from './auth'
import { isAccessTokenExpired } from '~/utils/jwt'

function makeToken(expOffsetSeconds: number) {
  const exp = Math.floor(Date.now() / 1000) + expOffsetSeconds
  const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }))
  const body = btoa(JSON.stringify({ exp }))
  return `${header}.${body}.signature`
}

describe('useAuthStore refresh', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.stubGlobal('useRuntimeConfig', () => ({ public: { apiUrl: 'http://api.test' } }))
    localStorage.clear()
  })

  it('deduplicates concurrent refresh calls', async () => {
    let refreshCalls = 0
    vi.stubGlobal('fetch', vi.fn(async () => {
      refreshCalls += 1
      await new Promise(resolve => setTimeout(resolve, 20))
      return new Response(JSON.stringify({
        access_token: 'new-access',
        refresh_token: 'new-refresh',
      }), { status: 200 })
    }))

    const auth = useAuthStore()
    auth.refreshToken = 'old-refresh'
    auth.accessToken = 'old-access'

    const results = await Promise.all([auth.refresh(), auth.refresh(), auth.refresh()])

    expect(results).toEqual([true, true, true])
    expect(refreshCalls).toBe(1)
    expect(auth.accessToken).toBe('new-access')
    expect(auth.refreshToken).toBe('new-refresh')
  })

  it('refreshes expired access tokens before API calls', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => new Response(JSON.stringify({
      access_token: 'fresh-access',
      refresh_token: 'fresh-refresh',
    }), { status: 200 })))

    const auth = useAuthStore()
    auth.refreshToken = 'refresh-token'
    auth.accessToken = makeToken(-60)

    expect(isAccessTokenExpired(auth.accessToken)).toBe(true)

    const refreshed = await auth.ensureAccessTokenFresh()

    expect(refreshed).toBe(true)
    expect(auth.accessToken).toBe('fresh-access')
    expect(isAccessTokenExpired(auth.accessToken)).toBe(false)
  })
})
