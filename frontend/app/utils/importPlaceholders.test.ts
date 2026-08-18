import { describe, expect, it } from 'vitest'
import {
  collectionVarNamesFromPostman,
  placeholdersNeedingEnvMapping,
  requestFieldPlaceholdersFromPostman,
} from './importPlaceholders'

const showcaseSnippet = `{
  "variable": [
    { "key": "jsonApi", "value": "https://example.com" },
    { "key": "postId", "value": "1" }
  ],
  "item": [
    {
      "name": "req",
      "request": {
        "method": "GET",
        "header": [{ "key": "X-Request-Id", "value": "req-{{$timestamp}}" }],
        "url": "{{jsonApi}}/posts/{{postId}}"
      },
      "event": [{
        "listen": "prerequest",
        "script": { "exec": ["pm.variables.set('runAt', 'now');"] }
      }]
    },
    {
      "name": "script-only",
      "request": {
        "method": "GET",
        "url": "{{jsonApi}}/users/{{demoUserId}}/posts"
      },
      "event": [{
        "listen": "prerequest",
        "script": { "exec": ["pm.variables.set('demoUserId', '1');"] }
      }]
    }
  ]
}`

describe('importPlaceholders', () => {
  it('reads collection variable names from postman', () => {
    const names = collectionVarNamesFromPostman(showcaseSnippet)
    expect([...names].sort()).toEqual(['jsonApi', 'postId'])
  })

  it('ignores script-only placeholders when scanning requests', () => {
    const names = requestFieldPlaceholdersFromPostman(showcaseSnippet)
    expect(names).toEqual(['demoUserId', 'jsonApi', 'postId'])
    expect(names).not.toContain('runAt')
    expect(names).not.toContain('$timestamp')
  })

  it('does not require env mapping for collection vars and builtins', () => {
    const missing = placeholdersNeedingEnvMapping('postman', showcaseSnippet, [])
    expect(missing).toEqual(['demoUserId'])
  })

  it('clears missing when env provides values', () => {
    const missing = placeholdersNeedingEnvMapping('postman', showcaseSnippet, ['demoUserId'])
    expect(missing).toEqual([])
  })
})
