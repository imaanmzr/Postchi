import { describe, expect, it } from 'vitest'
import { safeAuthRedirect } from './authRedirect'

describe('safeAuthRedirect', () => {
  it('preserves an internal API-reference deep link', () => {
    const path = '/workspaces/workspace-id/catalog?request=request-id'
    expect(safeAuthRedirect(path)).toBe(path)
  })

  it.each([
    'https://example.com',
    '//example.com/path',
    '/\\example.com/path',
    '/login',
    '/register',
    undefined,
  ])('rejects unsafe or recursive redirect %s', (value) => {
    expect(safeAuthRedirect(value)).toBe('/workspaces')
  })
})
