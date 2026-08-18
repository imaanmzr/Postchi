import { describe, expect, it } from 'vitest'
import { detectVariableAutocompleteContext, splitVariableHighlightParts } from './variableHighlight'

describe('variableHighlight', () => {
  it('splits complete and partial variable tokens', () => {
    expect(splitVariableHighlightParts('{{base}}/users')).toEqual([
      { text: '{{base}}', kind: 'variable' },
      { text: '/users', kind: 'text' },
    ])
    expect(splitVariableHighlightParts('{{paymentBase')).toEqual([
      { text: '{{paymentBase', kind: 'variable-partial' },
    ])
  })

  it('detects autocomplete context inside an open variable', () => {
    const value = '{{paymentBaseUrl}}/api'
    const cursor = '{{paymentBase'.length
    expect(detectVariableAutocompleteContext(value, cursor)).toEqual({
      query: 'paymentBase',
      replaceFrom: 0,
      replaceTo: cursor,
    })
  })

  it('returns null outside variable context', () => {
    expect(detectVariableAutocompleteContext('https://example.com', 10)).toBeNull()
    expect(detectVariableAutocompleteContext('{{done}}', '{{done}}'.length)).toBeNull()
  })
})
