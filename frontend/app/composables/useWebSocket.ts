import { wsUrl } from '~/utils/apiBase'

export function useWebSocket(workspaceId: Ref<string | null>) {
  const config = useRuntimeConfig()
  const auth = useAuthStore()
  const messages = ref<any[]>([])
  let socket: WebSocket | null = null

  function connect() {
    if (!import.meta.client || !workspaceId.value || !auth.accessToken) return
    const path = `/api/ws?workspace_id=${encodeURIComponent(workspaceId.value)}`
    const url = wsUrl(config.public.apiUrl as string, path, auth.accessToken)
    socket = new WebSocket(url)
    socket.onopen = () => {
      socket?.send(JSON.stringify({ type: 'presence.join' }))
    }
    socket.onmessage = (ev) => {
      try {
        messages.value.push(JSON.parse(ev.data))
      } catch { /* ignore */ }
    }
  }

  function disconnect() {
    socket?.close()
    socket = null
  }

  function send(msg: unknown) {
    socket?.send(JSON.stringify(msg))
  }

  watch([workspaceId, () => auth.accessToken], ([id, token]) => {
    disconnect()
    if (id && token) connect()
  }, { immediate: true })

  onUnmounted(disconnect)

  return { messages, send, connect, disconnect }
}
