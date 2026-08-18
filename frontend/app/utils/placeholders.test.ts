import { describe, expect, it } from 'vitest'
import { extractPlaceholders, extractPlaceholdersFromRequest } from './placeholders'

describe('extractPlaceholders', () => {
  it('extracts unique sorted names', () => {
    expect(extractPlaceholders('{{a}} {{b}} {{a}}')).toEqual(['a', 'b'])
  })
})

describe('extractPlaceholdersFromRequest', () => {
  it('skips disabled headers', () => {
    const names = extractPlaceholdersFromRequest({
      url: '',
      headers: [{ key: 'X', value: '{{skip}}', enabled: false }],
      params: [],
      body: { mode: 'raw', raw: '', raw_lang: 'json' },
    })
    expect(names).not.toContain('skip')
  })
})
