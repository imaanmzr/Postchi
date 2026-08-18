import { EditorView } from '@codemirror/view'
import { HighlightStyle, syntaxHighlighting } from '@codemirror/language'
import { tags as t } from '@lezer/highlight'

export const editorTheme = EditorView.theme({
  '&': {
    backgroundColor: 'var(--color-editor-bg)',
    color: 'var(--color-editor-text)',
    minHeight: '150px',
    width: '100%',
    fontFamily: 'var(--font-editor)',
  },
  '.cm-scroller': {
    fontFamily: 'var(--font-editor)',
  },
  '.cm-line': {
    fontFamily: 'var(--font-editor)',
  },
  '.cm-content': {
    caretColor: 'var(--color-editor-caret)',
    fontFamily: 'var(--font-editor) !important',
    fontSize: 'var(--font-editor-size)',
    lineHeight: 'var(--font-editor-line-height)',
    fontVariantLigatures: 'none',
    wordBreak: 'break-word',
    overflowWrap: 'anywhere',
  },
  '.cm-gutters': {
    backgroundColor: 'var(--color-editor-gutter)',
    color: 'var(--color-text-muted)',
    border: 'none',
  },
  '.cm-activeLine': { backgroundColor: 'var(--color-editor-active-line)' },
  '.cm-activeLineGutter': { backgroundColor: 'var(--color-editor-active-line)' },
  '.cm-selectionMatch': {
    backgroundColor: 'color-mix(in srgb, var(--color-syntax-string) 22%, transparent)',
  },
  '.cm-selectionMatch-main': {
    backgroundColor: 'color-mix(in srgb, var(--color-syntax-string) 32%, transparent)',
  },
  '&.cm-focused .cm-cursor': { borderLeftColor: 'var(--color-editor-caret)' },
  '&.cm-focused .cm-selectionBackground, .cm-selectionBackground': {
    backgroundColor: 'var(--color-editor-selection) !important',
  },
}, { dark: true })

/** Fills a flex parent and enables internal scrolling (docs editor, response viewer). */
export const editorFillLayout = EditorView.theme({
  '&': {
    height: '100%',
    width: '100%',
    minHeight: '0',
  },
  '.cm-scroller': {
    overflow: 'auto',
    fontFamily: 'var(--font-editor) !important',
    fontSize: 'var(--font-editor-size)',
    lineHeight: 'var(--font-editor-line-height)',
    fontVariantLigatures: 'none',
  },
}, { dark: true })

export const editorHighlightStyle = HighlightStyle.define([
  { tag: [t.heading1, t.heading2, t.heading3, t.heading4, t.heading5, t.heading6], color: 'var(--color-syntax-md-heading)', fontWeight: 'bold' },
  { tag: t.strong, color: 'var(--color-syntax-md-strong)', fontWeight: 'bold' },
  { tag: t.emphasis, color: 'var(--color-syntax-md-emphasis)', fontStyle: 'italic' },
  { tag: [t.link, t.url], color: 'var(--color-syntax-md-link)', textDecoration: 'underline' },
  { tag: t.monospace, color: 'var(--color-syntax-md-code)' },
  { tag: t.quote, color: 'var(--color-syntax-md-quote)', fontStyle: 'italic' },
  { tag: t.meta, color: 'var(--color-syntax-md-meta)' },
  { tag: t.processingInstruction, color: 'var(--color-syntax-md-marker)' },
  { tag: t.propertyName, color: 'var(--color-syntax-property)' },
  { tag: [t.string, t.special(t.string), t.literal, t.character], color: 'var(--color-syntax-string)' },
  { tag: [t.number, t.bool, t.null], color: 'var(--color-syntax-number)' },
  { tag: [t.punctuation, t.bracket, t.separator], color: 'var(--color-syntax-punctuation)' },
  { tag: [t.comment, t.lineComment, t.blockComment], color: 'var(--color-syntax-comment)' },
  { tag: [t.keyword, t.operatorKeyword], color: 'var(--color-syntax-keyword)' },
  { tag: [t.variableName, t.name], color: 'var(--color-syntax-variable)' },
  { tag: t.className, color: 'var(--color-syntax-keyword)' },
])

export const editorSyntax = syntaxHighlighting(editorHighlightStyle)
export const editorLineWrap = EditorView.lineWrapping
