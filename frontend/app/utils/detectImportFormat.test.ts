import { describe, expect, it } from 'vitest'
import { detectImportFormat } from './detectImportFormat'

describe('detectImportFormat', () => {
  it('detects Bruno by extension', () => {
    expect(detectImportFormat('api.bru', '')).toBe('bruno')
    expect(detectImportFormat('bundle.zip', '')).toBe('bruno')
  })

  it('prefers OpenAPI YAML content over extension', () => {
    const yaml = 'openapi: 3.0.0\ninfo:\n  title: Pets\n'
    expect(detectImportFormat('spec.yaml', yaml)).toBe('openapi')
  })

  it('detects OpenCollection YAML content', () => {
    const yaml = 'opencollection: 1.0.0\ninfo:\n  name: API\n'
    expect(detectImportFormat('collection.yml', yaml)).toBe('opencollection')
  })

  it('detects Postman JSON', () => {
    const json = JSON.stringify({
      info: { schema: 'https://schema.getpostman.com/json/collection/v2.1.0/collection.json' },
    })
    expect(detectImportFormat('api.json', json)).toBe('postman')
  })
})
