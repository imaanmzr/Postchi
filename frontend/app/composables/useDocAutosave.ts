export type SaveStatus = 'saved' | 'saving' | 'unsaved' | 'error'

export function useDocAutosave(options: {
  save: () => Promise<void>
  debounceMs?: number
}) {
  const status = ref<SaveStatus>('saved')
  const errorMessage = ref<string | null>(null)
  const delay = options.debounceMs ?? 1200
  let timer: ReturnType<typeof setTimeout> | null = null

  async function runSave() {
    status.value = 'saving'
    errorMessage.value = null
    try {
      await options.save()
      status.value = 'saved'
    } catch (e: unknown) {
      status.value = 'error'
      errorMessage.value = e instanceof Error ? e.message : 'Save failed'
    }
  }

  function cancelPending() {
    if (timer) {
      clearTimeout(timer)
      timer = null
    }
  }

  function debouncedSave() {
    cancelPending()
    timer = setTimeout(() => {
      timer = null
      void runSave()
    }, delay)
  }

  function markUnsaved() {
    if (status.value !== 'saving') status.value = 'unsaved'
  }

  async function flushSave() {
    cancelPending()
    if (status.value === 'saved') return
    await runSave()
  }

  async function forceSave() {
    cancelPending()
    await runSave()
  }

  function markSaved() {
    status.value = 'saved'
    errorMessage.value = null
  }

  return {
    status,
    errorMessage,
    markUnsaved,
    debouncedSave,
    flushSave,
    forceSave,
    cancelPending,
    markSaved,
  }
}
