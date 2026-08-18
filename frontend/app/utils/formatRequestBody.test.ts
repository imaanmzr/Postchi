import { describe, expect, it } from 'vitest'
import { formatRequestBody } from './formatRequestBody'

describe('formatRequestBody', () => {
  it('pretty-prints raw JSON bodies', () => {
    const body = {
      mode: 'raw',
      raw: '{"isBale":false,"phoneNumber":""}',
      raw_lang: 'json',
    }
    expect(formatRequestBody(body).raw).toBe(
      JSON.stringify({ isBale: false, phoneNumber: '' }, null, 2),
    )
  })

  it('pretty-prints json mode bodies', () => {
    const body = {
      mode: 'json',
      raw: '{"a":1}',
      raw_lang: 'json',
    }
    expect(formatRequestBody(body).raw).toBe('{\n  "a": 1\n}')
  })

  it('leaves non-JSON raw bodies unchanged', () => {
    const body = {
      mode: 'raw',
      raw: 'plain text',
      raw_lang: 'text',
    }
    expect(formatRequestBody(body)).toEqual(body)
  })

  it('pretty-prints GraphQL variables', () => {
    const body = {
      mode: 'graphql',
      raw: '',
      raw_lang: 'json',
      graphql: { query: 'query {}', variables: '{"id":1}' },
    }
    expect(formatRequestBody(body).graphql?.variables).toBe('{\n  "id": 1\n}')
  })
})
