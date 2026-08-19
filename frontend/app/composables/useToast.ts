export type ToastKind = 'success' | 'error'

export interface ToastItem {
  id: number
  kind: ToastKind
  message: string
}

const toasts = ref<ToastItem[]>([])
let nextToastId = 0

export function useToast() {
  function show(message: string, kind: ToastKind = 'success', duration = 3500) {
    const id = ++nextToastId
    toasts.value.push({ id, kind, message })

    if (import.meta.client) {
      window.setTimeout(() => {
        toasts.value = toasts.value.filter(toast => toast.id !== id)
      }, duration)
    }
  }

  function dismiss(id: number) {
    toasts.value = toasts.value.filter(toast => toast.id !== id)
  }

  return {
    toasts: readonly(toasts),
    show,
    dismiss,
  }
}
