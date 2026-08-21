export type ExcalidrawScene = {
  type?: string
  version?: number
  source?: string
  elements: unknown[]
  appState: Record<string, unknown>
  files: Record<string, unknown>
}

/** Only these appState fields are safe to persist — full appState breaks UI on reload. */
const PERSISTED_APP_STATE_KEYS = [
  'viewBackgroundColor',
  'gridSize',
  'gridModeEnabled',
  'currentItemStrokeColor',
  'currentItemBackgroundColor',
  'currentItemFillStyle',
  'currentItemStrokeWidth',
  'currentItemRoughness',
  'currentItemOpacity',
  'currentItemFontFamily',
  'currentItemFontSize',
  'currentItemTextAlign',
  'currentItemStartArrowhead',
  'currentItemEndArrowhead',
  'currentItemRoundness',
] as const

export function emptyExcalidrawScene(): ExcalidrawScene {
  return {
    type: 'excalidraw',
    version: 2,
    source: 'postchi',
    elements: [],
    appState: { viewBackgroundColor: '#ffffff' },
    files: {},
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === 'object' && !Array.isArray(value)
}

/** Drop deleted elements and non-objects before saving. */
export function persistElements(elements: unknown[]): unknown[] {
  if (!Array.isArray(elements)) return []
  return elements.filter((el) => {
    if (!isRecord(el)) return false
    return el.isDeleted !== true
  })
}

/** Keep only UI-safe appState fields (scroll/zoom/selection break reload). */
export function pickPersistedAppState(appState: Record<string, unknown>): Record<string, unknown> {
  const picked: Record<string, unknown> = {}
  for (const key of PERSISTED_APP_STATE_KEYS) {
    if (key in appState) picked[key] = appState[key]
  }
  if (!('viewBackgroundColor' in picked)) {
    picked.viewBackgroundColor = '#ffffff'
  }
  return picked
}

/** Normalize DB/file JSON into the shape Excalidraw initialData expects. */
export function toExcalidrawInitialData(raw?: Record<string, unknown> | null): {
  elements: unknown[]
  appState: Record<string, unknown>
  files: Record<string, unknown>
} {
  if (!raw) {
    const empty = emptyExcalidrawScene()
    return { elements: empty.elements, appState: empty.appState, files: empty.files }
  }
  const elements = persistElements(Array.isArray(raw.elements) ? raw.elements : [])
  const rawAppState = isRecord(raw.appState) ? raw.appState : {}
  const appState = pickPersistedAppState(rawAppState)
  const files = isRecord(raw.files) ? raw.files : {}
  return { elements, appState, files }
}

/** Normalize onChange payload into storable scene JSON. */
export function toStoredExcalidrawScene(
  elements: unknown[],
  appState: Record<string, unknown>,
  files: Record<string, unknown>,
): ExcalidrawScene {
  return {
    type: 'excalidraw',
    version: 2,
    source: 'postchi',
    elements: persistElements(elements),
    appState: pickPersistedAppState(appState),
    files: isRecord(files) ? files : {},
  }
}

export function diagramWikilink(slug: string, title?: string): string {
  const safeSlug = slug.trim()
  const label = (title || safeSlug).trim()
  return `[[diagram:${safeSlug}|${label}]]`
}

export function parseDiagramWikilink(target: string): { slug: string, label?: string } | null {
  const match = target.trim().match(/^diagram:([^|]+)(?:\|(.+))?$/i)
  if (!match) return null
  return { slug: match[1].trim(), label: match[2]?.trim() }
}
