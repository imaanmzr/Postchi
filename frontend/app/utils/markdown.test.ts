import { describe, it, expect } from 'vitest'
import { renderMarkdown, resolveDocSlug } from './markdown'

describe('renderMarkdown', () => {
  it('renders basic markdown', () => {
    const html = renderMarkdown('**bold** and `code`')
    expect(html).toContain('<strong>bold</strong>')
    expect(html).toContain('<code>code</code>')
  })

  it('returns empty for blank input', () => {
    expect(renderMarkdown('')).toBe('')
    expect(renderMarkdown('   ')).toBe('')
  })

  it('renders wikilinks as navigable anchors', () => {
    const slugs = new Set(['api-auth'])
    const titles = new Map<string, string>()
    const html = renderMarkdown('See [[api-auth]] for details.', {
      resolveLink: target => resolveDocSlug(target, slugs, titles),
    })
    expect(html).toContain('class="wikilink"')
    expect(html).toContain('data-doc-slug="api-auth"')
  })
})

describe('resolveDocSlug', () => {
  it('resolves by slug, path, and title', () => {
    const slugs = new Set(['getting-started', 'api-auth'])
    const titles = new Map([['api auth', 'api-auth']])
    expect(resolveDocSlug('getting-started', slugs, titles)).toBe('getting-started')
    expect(resolveDocSlug('./api-auth.md', slugs, titles)).toBe('api-auth')
    expect(resolveDocSlug('API Auth', slugs, titles)).toBe('api-auth')
  })
})
