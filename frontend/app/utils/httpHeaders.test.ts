import { describe, expect, it } from 'vitest'
import { filterHttpHeaders, HTTP_HEADER_NAMES } from './httpHeaders'

describe('filterHttpHeaders', () => {
  it('returns first batch when query empty', () => {
    const result = filterHttpHeaders('', 5)
    expect(result).toHaveLength(5)
    expect(result[0]).toBe(HTTP_HEADER_NAMES[0])
  })

  it('matches prefix case-insensitively', () => {
    const result = filterHttpHeaders('accept')
    expect(result[0]).toBe('Accept')
    expect(result).toContain('Accept-Encoding')
    expect(result).toContain('Accept-Language')
  })

  it('matches D-prefix headers', () => {
    const result = filterHttpHeaders('D')
    expect(result).toContain('DASL')
    expect(result).toContain('DAV')
    expect(result).toContain('Date')
    expect(result).toContain('Destination')
  })

  it('includes substring matches after prefix matches', () => {
    const result = filterHttpHeaders('Match')
    expect(result[0]).toBe('If-Match')
    expect(result).toContain('If-None-Match')
  })
})
