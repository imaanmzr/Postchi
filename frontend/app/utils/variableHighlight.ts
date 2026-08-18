export type VariableTextPartKind = 'text' | 'variable' | 'variable-partial'

export interface VariableTextPart {
  text: string
  kind: VariableTextPartKind
}

/** Split text into plain segments and {{variable}} tokens for display. */
export function splitVariableHighlightParts(text: string): VariableTextPart[] {
  if (!text) return []

  const parts: VariableTextPart[] = []
  const re = /\{\{[^}]*\}\}|\{\{[^}]*$/g
  let lastIndex = 0
  let match: RegExpExecArray | null

  while ((match = re.exec(text)) !== null) {
    if (match.index > lastIndex) {
      parts.push({ text: text.slice(lastIndex, match.index), kind: 'text' })
    }
    const token = match[0]
    parts.push({
      text: token,
      kind: token.endsWith('}}') ? 'variable' : 'variable-partial',
    })
    lastIndex = match.index + token.length
  }

  if (lastIndex < text.length) {
    parts.push({ text: text.slice(lastIndex), kind: 'text' })
  }

  return parts
}

export interface VariableAutocompleteContext {
  query: string
  replaceFrom: number
  replaceTo: number
}

/** Return autocomplete context when the cursor is inside an open `{{` token. */
export function detectVariableAutocompleteContext(
  value: string,
  cursor: number,
): VariableAutocompleteContext | null {
  const before = value.slice(0, cursor)
  const match = /\{\{([^}]*)$/.exec(before)
  if (!match) return null
  return {
    query: match[1],
    replaceFrom: cursor - match[0].length,
    replaceTo: cursor,
  }
}
