import { autocompletion, type CompletionContext } from '@codemirror/autocomplete'
import { defaultKeymap, historyKeymap, indentWithTab } from '@codemirror/commands'
import { markdown, markdownLanguage } from '@codemirror/lang-markdown'
import { javascript } from '@codemirror/lang-javascript'
import { json } from '@codemirror/lang-json'
import { LanguageDescription, indentOnInput } from '@codemirror/language'
import { closeBracketsKeymap } from '@codemirror/autocomplete'
import { Prec, type Extension } from '@codemirror/state'
import { EditorView, keymap, placeholder } from '@codemirror/view'
import { editorBasicSetup } from '~/utils/codemirror/editorSetup'
import { editorLineWrap, editorSelectionTheme, editorSyntax, editorTheme } from '~/utils/codemirror/editorTheme'

export interface MarkdownEditorOptions {
  placeholder?: string
  docCompletions?: { label: string, slug: string }[]
  onTogglePreview?: () => void
  onForceSave?: () => void
}

function toggleWrap(view: EditorView, before: string, after: string) {
  const { from, to } = view.state.selection.main
  const selected = view.state.sliceDoc(from, to)
  if (selected.startsWith(before) && selected.endsWith(after)) {
    view.dispatch({
      changes: { from, to, insert: selected.slice(before.length, selected.length - after.length) },
      selection: { anchor: from, head: to - before.length - after.length },
    })
    return true
  }
  view.dispatch({
    changes: { from, to, insert: `${before}${selected}${after}` },
    selection: { anchor: from + before.length, head: to + before.length },
  })
  return true
}

function wikilinkCompletion(context: CompletionContext, docs: { label: string, slug: string }[]) {
  const before = context.matchBefore(/\[\[[^\]]*/)
  if (!before || (before.from === before.to && !context.explicit)) return null
  const query = before.text.slice(2).toLowerCase()
  const options = docs
    .filter(d => d.label.toLowerCase().includes(query) || d.slug.toLowerCase().includes(query))
    .slice(0, 20)
    .map(d => ({
      label: d.label,
      detail: d.slug,
      apply: `[[${d.slug}|${d.label}]]`,
    }))
  return { from: before.from + 2, options }
}

export function createMarkdownExtensions(options: MarkdownEditorOptions = {}): Extension[] {
  const docList = options.docCompletions ?? []

  const obsidianKeymap = keymap.of([
    {
      key: 'Mod-b',
      run: (view) => toggleWrap(view, '**', '**'),
    },
    {
      key: 'Mod-i',
      run: (view) => toggleWrap(view, '*', '*'),
    },
    {
      key: 'Mod-k',
      run: (view) => {
        const { from, to } = view.state.selection.main
        const selected = view.state.sliceDoc(from, to) || 'link text'
        view.dispatch({
          changes: { from, to, insert: `[${selected}](url)` },
          selection: { anchor: from + selected.length + 3, head: from + selected.length + 6 },
        })
        return true
      },
    },
    {
      key: 'Mod-e',
      run: () => {
        options.onTogglePreview?.()
        return true
      },
    },
    {
      key: 'Mod-s',
      run: () => {
        options.onForceSave?.()
        return true
      },
      preventDefault: true,
    },
  ])

  return [
    editorBasicSetup,
    editorLineWrap,
    indentOnInput(),
    markdown({
      base: markdownLanguage,
      // Must be real LanguageDescription instances: lang-markdown reads
      // `desc.alias` when resolving fence languages and crashes on plain objects.
      codeLanguages: [
        LanguageDescription.of({
          name: 'javascript',
          alias: ['js', 'ts', 'typescript'],
          load: async () => javascript(),
        }),
        LanguageDescription.of({
          name: 'json',
          load: async () => json(),
        }),
      ],
    }),
    editorSelectionTheme,
    editorTheme,
    editorSyntax,
    placeholder(options.placeholder ?? 'Write markdown…'),
    autocompletion({
      override: [
        (ctx) => wikilinkCompletion(ctx, docList),
      ],
    }),
    Prec.highest(obsidianKeymap),
    keymap.of([
      ...closeBracketsKeymap,
      ...defaultKeymap,
      ...historyKeymap,
      indentWithTab,
    ]),
  ]
}
