import { describe, expect, it } from 'vitest'
import {
  diagramWikilink,
  emptyExcalidrawScene,
  parseDiagramWikilink,
  persistElements,
  pickPersistedAppState,
  toExcalidrawInitialData,
  toStoredExcalidrawScene,
} from './diagramContent'

describe('diagramContent', () => {
  it('normalizes stored scene for Excalidraw initialData', () => {
    const stored = {
      type: 'excalidraw',
      version: 2,
      elements: [{ id: 'a', type: 'rectangle' }],
      appState: { viewBackgroundColor: '#eee', zoom: { value: 0.5 }, scrollX: 100 },
      files: { img1: { id: 'img1' } },
    }
    const initial = toExcalidrawInitialData(stored)
    expect(initial.elements).toHaveLength(1)
    expect(initial.appState).toEqual({ viewBackgroundColor: '#eee' })
    expect(initial.appState.zoom).toBeUndefined()
    expect(initial.files).toEqual({ img1: { id: 'img1' } })
  })

  it('returns empty scene when data is missing', () => {
    const initial = toExcalidrawInitialData(null)
    expect(initial.elements).toEqual([])
    expect(initial.appState.viewBackgroundColor).toBe('#ffffff')
  })

  it('stores onChange payload without unsafe appState fields', () => {
    const scene = toStoredExcalidrawScene(
      [{ id: 'x' }, { id: 'y', isDeleted: true }],
      { viewBackgroundColor: '#fff', scrollX: 50, selectedElementIds: ['x'] },
      {},
    )
    expect(scene.type).toBe('excalidraw')
    expect(scene.elements).toEqual([{ id: 'x' }])
    expect(scene.appState).toEqual({ viewBackgroundColor: '#fff' })
  })

  it('filters deleted elements and picks safe appState keys', () => {
    expect(persistElements([{ id: 'a' }, { id: 'b', isDeleted: true }])).toEqual([{ id: 'a' }])
    expect(pickPersistedAppState({ viewBackgroundColor: '#000', scrollX: 1 })).toEqual({
      viewBackgroundColor: '#000',
    })
  })

  it('builds and parses diagram wikilinks', () => {
    expect(diagramWikilink('checkout-flow', 'Checkout')).toBe('[[diagram:checkout-flow|Checkout]]')
    expect(parseDiagramWikilink('diagram:checkout-flow|Checkout')).toEqual({
      slug: 'checkout-flow',
      label: 'Checkout',
    })
  })

  it('emptyExcalidrawScene has expected defaults', () => {
    const scene = emptyExcalidrawScene()
    expect(scene.elements).toEqual([])
    expect(scene.version).toBe(2)
  })
})
