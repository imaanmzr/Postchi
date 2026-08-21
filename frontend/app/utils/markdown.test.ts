import { describe, expect, it } from 'vitest'
import { renderMarkdown } from './markdown'

describe('renderMarkdown diagram links', () => {
  it('renders [[diagram:slug|Label]] as preview card', () => {
    const html = renderMarkdown('See [[diagram:checkout|Checkout flow]] here.', {
      diagramTitles: new Map([['checkout', 'Checkout flow']]),
    })
    expect(html).toContain('class="diagram-link"')
    expect(html).toContain('data-diagram-slug="checkout"')
    expect(html).toContain('Checkout flow')
  })

  it('does not treat diagram wikilinks as doc wikilinks', () => {
    const html = renderMarkdown('[[diagram:story-1|Story 1]]', {
      resolveLink: () => 'wrong-slug',
      diagramTitles: new Map([['story-1', 'Story 1']]),
    })
    expect(html).toContain('data-diagram-slug="story-1"')
    expect(html).not.toContain('data-doc-slug')
  })

  it('still renders doc wikilinks', () => {
    const html = renderMarkdown('[[my-doc|My Doc]]', {
      resolveLink: target => target,
    })
    expect(html).toContain('class="wikilink"')
    expect(html).toContain('data-doc-slug="my-doc"')
  })
})
