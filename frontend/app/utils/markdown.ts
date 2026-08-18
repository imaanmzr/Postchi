import { marked, type Tokens } from 'marked'
import DOMPurify from 'dompurify'

marked.setOptions({ gfm: true, breaks: true })

export type WikilinkResolver = (target: string) => string | null

function lineOf(text: string, raw: string): number {
  const idx = text.indexOf(raw)
  if (idx < 0) return 1
  return text.slice(0, idx).split('\n').length
}

function preprocessWikilinks(text: string, resolveLink?: WikilinkResolver): string {
  return text.replace(/\[\[([^\]|]+)(?:\|([^\]]+))?\]\]/g, (_match, target: string, label?: string) => {
    const display = (label || target).trim()
    const slug = resolveLink?.(target.trim()) ?? target.trim()
    return `[${display}](wikilink:${encodeURIComponent(slug)})`
  })
}

function createRenderer(source: string) {
  const renderer = new marked.Renderer()
  const baseLink = renderer.link.bind(renderer)
  const withLine = (tag: string, line: number, inner: string, attrs = '') =>
    `<${tag} data-line="${line}"${attrs}>${inner}</${tag}>`

  renderer.heading = function heading({ tokens, depth, raw }: Tokens.Heading) {
    const line = lineOf(source, raw)
    const inner = this.parser.parseInline(tokens)
    return withLine(`h${depth}`, line, inner) + '\n'
  }
  renderer.paragraph = function paragraph({ tokens, raw }: Tokens.Paragraph) {
    const line = lineOf(source, raw)
    const inner = this.parser.parseInline(tokens)
    return withLine('p', line, inner) + '\n'
  }
  renderer.code = function code({ text, raw, lang }: Tokens.Code) {
    const line = lineOf(source, raw)
    const langClass = lang ? ` class="language-${lang}"` : ''
    return `<pre data-line="${line}"><code${langClass}>${text}</code></pre>\n`
  }
  renderer.blockquote = function blockquote({ tokens, raw }: Tokens.Blockquote) {
    const line = lineOf(source, raw)
    const inner = this.parser.parse(tokens)
    return withLine('blockquote', line, inner) + '\n'
  }
  renderer.list = function list(token: Tokens.List) {
    const line = lineOf(source, token.raw)
    const tag = token.ordered ? 'ol' : 'ul'
    const body = token.items.map(item => this.listitem(item)).join('')
    return withLine(tag, line, body) + '\n'
  }
  renderer.link = function link({ href, title, tokens }) {
    if (href?.startsWith('wikilink:')) {
      const slug = decodeURIComponent(href.slice('wikilink:'.length))
      const text = this.parser.parseInline(tokens)
      return `<a href="#" class="wikilink" data-doc-slug="${slug}">${text}</a>`
    }
    return baseLink({ href, title, tokens })
  }
  return renderer
}

export function renderMarkdown(text: string, options?: { resolveLink?: WikilinkResolver }): string {
  if (!text?.trim()) return ''
  const processed = preprocessWikilinks(text, options?.resolveLink)
  const html = marked.parse(processed, { renderer: createRenderer(text) }) as string
  if (import.meta.client) {
    return DOMPurify.sanitize(html, {
      ADD_ATTR: ['data-doc-slug', 'data-line'],
    })
  }
  return html
}

export function resolveDocSlug(target: string, slugs: Set<string>, titles: Map<string, string>): string | null {
  const trimmed = target.trim()
  if (!trimmed) return null
  if (slugs.has(trimmed)) return trimmed

  const fromPath = trimmed
    .replace(/^\.\//, '')
    .replace(/^\//, '')
    .replace(/#.*$/, '')
    .replace(/\.md$/, '')
    .replace(/\//g, '-')
  if (slugs.has(fromPath)) return fromPath

  const slugified = trimmed.toLowerCase().replace(/\s+/g, '-')
  if (slugs.has(slugified)) return slugified

  const byTitle = titles.get(trimmed.toLowerCase())
  if (byTitle) return byTitle

  return null
}
