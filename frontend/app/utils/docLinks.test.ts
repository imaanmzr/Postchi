import { describe, expect, it } from 'vitest'
import { buildCatalogRequestUrl, buildWorkspaceRequestUrl } from './docLinks'

describe('request links', () => {
  it('builds a live API-reference link for one request', () => {
    expect(buildCatalogRequestUrl('workspace-id', 'request-id'))
      .toBe('/workspaces/workspace-id/catalog?request=request-id')
  })

  it('keeps request-editor links separate from API-reference links', () => {
    expect(buildWorkspaceRequestUrl('workspace-id', 'request-id'))
      .toBe('/workspaces/workspace-id?request=request-id')
  })
})
