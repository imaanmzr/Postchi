import { describe, expect, it } from 'vitest'
import { detectResponseBodyLang, formatResponseBody } from './formatResponseBody'

describe('formatResponseBody', () => {
  it('pretty-prints JSON strings', () => {
    const raw = '{"ok":true,"items":[1,2]}'
    expect(formatResponseBody(raw)).toBe(
      JSON.stringify({ ok: true, items: [1, 2] }, null, 2),
    )
  })

  it('returns plain text unchanged', () => {
    expect(formatResponseBody('not json')).toBe('not json')
  })

  it('pretty-prints object bodies', () => {
    expect(formatResponseBody({ a: 1 })).toBe('{\n  "a": 1\n}')
  })
})

describe('detectResponseBodyLang', () => {
  it('detects JSON from content-type', () => {
    expect(detectResponseBodyLang('{}', { 'Content-Type': 'application/json' })).toBe('json')
  })

  it('detects JSON from body shape', () => {
    expect(detectResponseBodyLang('{"ok":true}')).toBe('json')
  })

  it('detects XML from body shape', () => {
    expect(detectResponseBodyLang('<root/>')).toBe('xml')
  })

  it('falls back to plain text', () => {
    expect(detectResponseBodyLang('hello')).toBe('text')
  })
})
