import { describe, expect, it } from 'vitest'
import { getAccessTokenExpiry, isAccessTokenExpired } from './jwt'

function makeToken(payload: Record<string, unknown>) {
  const header = btoa(JSON.stringify({ alg: 'HS256', typ: 'JWT' }))
  const body = btoa(JSON.stringify(payload))
  return `${header}.${body}.signature`
}

describe('jwt utils', () => {
  it('reads exp from access tokens', () => {
    const exp = Math.floor(Date.now() / 1000) + 300
    expect(getAccessTokenExpiry(makeToken({ exp }))).toBe(exp)
  })

  it('detects expired access tokens', () => {
    const exp = Math.floor(Date.now() / 1000) - 10
    expect(isAccessTokenExpired(makeToken({ exp }))).toBe(true)
  })

  it('treats valid tokens as fresh', () => {
    const exp = Math.floor(Date.now() / 1000) + 300
    expect(isAccessTokenExpired(makeToken({ exp }))).toBe(false)
  })
})
