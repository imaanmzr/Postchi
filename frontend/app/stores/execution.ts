import { defineStore } from 'pinia'

export const useExecutionStore = defineStore('execution', {
  state: () => ({
    responses: {} as Record<string, unknown>,
  }),
  actions: {
    set(requestId: string, response: unknown) {
      if (!requestId) return
      this.responses[requestId] = response
    },
    get(requestId: string) {
      return requestId ? this.responses[requestId] ?? null : null
    },
    clear(requestId?: string) {
      if (requestId) {
        delete this.responses[requestId]
        return
      }
      this.responses = {}
    },
  },
})
