import { defineStore } from 'pinia'

export interface TabEntry {
  key: string
  type: 'request' | 'collection' | 'openapi'
  entityId: string
  label: string
  method?: string
  dirty: boolean
  /** Collection settings: initial sub-tab (e.g. vars). */
  collectionTab?: string
}

export const useTabsStore = defineStore('tabs', {
  state: () => ({
    openTabs: [] as TabEntry[],
    activeKey: null as string | null,
  }),
  getters: {
    activeTab: (s) => s.openTabs.find(t => t.key === s.activeKey) || null,
  },
  actions: {
    openRequest(req: { id?: string; name: string; method: string }) {
      const key = `request:${req.id || 'new'}`
      const existing = this.openTabs.find(t => t.key === key)
      if (existing) {
        existing.label = req.name
        existing.method = req.method
      } else {
        this.openTabs.push({
          key,
          type: 'request',
          entityId: req.id || '',
          label: req.name,
          method: req.method,
          dirty: !req.id,
        })
      }
      this.activeKey = key
    },
    updateRequestTabMeta(key: string | null, meta: { name?: string; method?: string }) {
      const tabKey = key ?? this.activeKey
      if (!tabKey) return
      const tab = this.openTabs.find(t => t.key === tabKey)
      if (!tab || tab.type !== 'request') return
      if (meta.name !== undefined) tab.label = meta.name
      if (meta.method !== undefined) tab.method = meta.method
    },
    syncSavedRequest(previousKey: string | null, req: { id: string; name: string; method: string }) {
      const key = previousKey ?? this.activeKey
      if (!key) return
      const idx = this.openTabs.findIndex(t => t.key === key)
      if (idx < 0) return
      const tab = this.openTabs[idx]
      if (tab.type !== 'request') return

      const newKey = `request:${req.id}`
      this.openTabs[idx] = {
        ...tab,
        key: newKey,
        entityId: req.id,
        label: req.name,
        method: req.method,
        dirty: false,
      }
      if (this.activeKey === key) this.activeKey = newKey
    },
    openCollection(col: { id: string; name: string }, options?: { tab?: string }) {
      const key = `collection:${col.id}`
      const existing = this.openTabs.find(t => t.key === key)
      if (existing) {
        if (options?.tab) existing.collectionTab = options.tab
      } else {
        this.openTabs.push({
          key,
          type: 'collection',
          entityId: col.id,
          label: col.name,
          dirty: false,
          collectionTab: options?.tab,
        })
      }
      this.activeKey = key
    },
    openOpenApi() {
      const key = 'openapi:connect'
      if (!this.openTabs.find(t => t.key === key)) {
        this.openTabs.push({
          key,
          type: 'openapi',
          entityId: '',
          label: 'OpenAPI',
          dirty: false,
        })
      }
      this.activeKey = key
    },
    setActive(key: string) {
      this.activeKey = key
    },
    closeTab(key: string) {
      this._closeTabs([key])
    },
    closeOthers(key: string) {
      const keys = this.openTabs.filter(t => t.key !== key).map(t => t.key)
      this._closeTabs(keys)
    },
    closeToLeft(key: string) {
      const idx = this.openTabs.findIndex(t => t.key === key)
      if (idx <= 0) return
      this._closeTabs(this.openTabs.slice(0, idx).map(t => t.key))
    },
    closeToRight(key: string) {
      const idx = this.openTabs.findIndex(t => t.key === key)
      if (idx < 0 || idx >= this.openTabs.length - 1) return
      this._closeTabs(this.openTabs.slice(idx + 1).map(t => t.key))
    },
    closeSaved() {
      const keys = this.openTabs.filter(t => !t.dirty).map(t => t.key)
      this._closeTabs(keys, { skipDirtyConfirm: true })
    },
    closeAll() {
      this._closeTabs(this.openTabs.map(t => t.key))
    },
    closeTabsForEntities(
      type: 'request' | 'collection',
      entityIds: string[],
      options?: { skipDirtyConfirm?: boolean },
    ) {
      const keys = entityIds.map(id => `${type}:${id}`)
      this._closeTabs(keys, options)
    },
    _closeTabs(keys: string[], options?: { skipDirtyConfirm?: boolean }) {
      const uniqueKeys = [...new Set(keys)]
      for (const key of uniqueKeys) {
        const tab = this.openTabs.find(t => t.key === key)
        if (!tab) continue
        if (tab.dirty && !options?.skipDirtyConfirm) {
          if (!confirm('Discard unsaved changes?')) return
        }
      }
      const remove = new Set(uniqueKeys)
      const remaining = this.openTabs.filter(t => !remove.has(t.key))
      this.openTabs = remaining
      if (this.activeKey && remove.has(this.activeKey)) {
        this.activeKey = remaining[remaining.length - 1]?.key ?? null
      }
    },
    markDirty(key: string) {
      const tab = this.openTabs.find(t => t.key === key)
      if (tab) tab.dirty = true
    },
    markClean(key: string) {
      const tab = this.openTabs.find(t => t.key === key)
      if (tab) tab.dirty = false
    },
    clear() {
      this.openTabs = []
      this.activeKey = null
    },
  },
})
