import { describe, expect, it, beforeEach } from 'vitest'
import { setActivePinia, createPinia } from 'pinia'
import { useExecutionStore } from './execution'

describe('useExecutionStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('stores and retrieves responses per request id', () => {
    const store = useExecutionStore()
    const body = { status_code: 200, body: '{"ok":true}' }
    store.set('req-1', body)
    expect(store.get('req-1')).toEqual(body)
    expect(store.get('req-2')).toBeNull()
  })

  it('clears a single response', () => {
    const store = useExecutionStore()
    store.set('req-1', { status_code: 200 })
    store.clear('req-1')
    expect(store.get('req-1')).toBeNull()
  })
})
