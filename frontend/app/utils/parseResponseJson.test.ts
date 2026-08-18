import { describe, expect, it } from 'vitest'
import { parseResponseJson, serializeJsonValue } from './parseResponseJson'

describe('parseResponseJson', () => {
  it('parses JSON strings', () => {
    expect(parseResponseJson('{"ok":true}')).toEqual({ ok: true })
  })

  it('returns objects as-is', () => {
    expect(parseResponseJson({ a: 1 })).toEqual({ a: 1 })
  })

  it('returns null for non-JSON text', () => {
    expect(parseResponseJson('hello')).toBeNull()
  })
})

describe('serializeJsonValue', () => {
  it('copies strings without quotes', () => {
    expect(serializeJsonValue('token')).toBe('token')
  })

  it('stringifies nested values', () => {
    expect(serializeJsonValue({ a: 1 })).toBe('{\n  "a": 1\n}')
  })
})
