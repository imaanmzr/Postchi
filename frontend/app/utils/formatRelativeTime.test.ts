import { describe, expect, it } from 'vitest'
import { formatRelativeTime } from './formatRelativeTime'

describe('formatRelativeTime', () => {
  it('returns empty string for falsy input', () => {
    expect(formatRelativeTime(null)).toBe('')
    expect(formatRelativeTime(undefined)).toBe('')
  })

  it('formats recent times', () => {
    const now = new Date()
    expect(formatRelativeTime(now)).toBe('just now')

    const hourAgo = new Date(now.getTime() - 2 * 60 * 60 * 1000)
    expect(formatRelativeTime(hourAgo)).toBe('2 hours ago')

    const dayAgo = new Date(now.getTime() - 24 * 60 * 60 * 1000)
    expect(formatRelativeTime(dayAgo)).toBe('1 day ago')
  })
})
