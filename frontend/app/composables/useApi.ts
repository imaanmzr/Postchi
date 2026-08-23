import { apiUrl } from '~/utils/apiBase'
import { retryOnUnauthorized, shouldLogoutOnUnauthorized } from '~/utils/apiSession'

function isPublicAuthPath(path: string) {
  return path.startsWith('/api/auth/login')
    || path.startsWith('/api/auth/register')
    || path.startsWith('/api/auth/refresh')
}

export function useApi() {
  const config = useRuntimeConfig()
  const auth = useAuthStore()

  function base(path: string) {
    return apiUrl(config.public.apiUrl as string, path)
  }

  async function handleUnauthorized(path: string, res: Response) {
    if (isPublicAuthPath(path)) return res

    auth.clearSession()
    if (import.meta.client) {
      const router = useRouter()
      if (router.currentRoute.value.path !== '/login' && router.currentRoute.value.path !== '/register') {
        await router.push('/login')
      }
    }
    const err = await res.json().catch(() => ({ error: 'Session expired. Please sign in again.' }))
    throw new Error(err.error || 'Session expired. Please sign in again.')
  }

  function applyAuthHeader(headers: Record<string, string>) {
    if (auth.accessToken) {
      headers.Authorization = `Bearer ${auth.accessToken}`
    }
  }

  async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
      ...(options.headers as Record<string, string> || {}),
    }
    if (auth.accessToken) {
      headers.Authorization = `Bearer ${auth.accessToken}`
    }

    if (!isPublicAuthPath(path)) {
      await auth.ensureAccessTokenFresh()
      if (auth.accessToken) {
        headers.Authorization = `Bearer ${auth.accessToken}`
      }
    }

    const { response: res, refreshSucceeded } = await retryOnUnauthorized(
      () => {
        applyAuthHeader(headers)
        return fetch(base(path), { ...options, headers })
      },
      async () => {
        if (!auth.refreshToken || isPublicAuthPath(path)) return false
        return auth.refresh()
      },
    )

    if (res.status === 401 && !isPublicAuthPath(path)) {
      if (!shouldLogoutOnUnauthorized(refreshSucceeded)) {
        const err = await res.json().catch(() => ({ error: 'Request failed' }))
        throw new Error(err.error || 'Request failed')
      }
      return handleUnauthorized(path, res)
    }

    if (!res.ok) {
      const err = await res.json().catch(() => ({ error: res.statusText }))
      throw new Error(err.error || 'Request failed')
    }
    if (res.status === 204) return undefined as T
    return res.json()
  }

  async function upload<T>(path: string, formData: FormData): Promise<T> {
    const headers: Record<string, string> = {}
    if (auth.accessToken) headers.Authorization = `Bearer ${auth.accessToken}`

    const { response: res, refreshSucceeded } = await retryOnUnauthorized(
      () => {
        applyAuthHeader(headers)
        return fetch(base(path), { method: 'POST', headers, body: formData })
      },
      async () => !!auth.refreshToken && auth.refresh(),
    )

    if (res.status === 401) {
      if (!shouldLogoutOnUnauthorized(refreshSucceeded)) {
        const err = await res.json().catch(() => ({ error: 'Upload failed' }))
        throw new Error(err.error || 'Upload failed')
      }
      return handleUnauthorized(path, res)
    }

    if (!res.ok) {
      const err = await res.json().catch(() => ({ error: res.statusText }))
      throw new Error(err.error || 'Upload failed')
    }
    return res.json()
  }

  async function download(path: string, filename: string) {
    const headers: Record<string, string> = {}
    if (auth.accessToken) headers.Authorization = `Bearer ${auth.accessToken}`

    const { response: res, refreshSucceeded } = await retryOnUnauthorized(
      () => {
        applyAuthHeader(headers)
        return fetch(base(path), { headers })
      },
      async () => !!auth.refreshToken && auth.refresh(),
    )

    if (res.status === 401) {
      if (!shouldLogoutOnUnauthorized(refreshSucceeded)) {
        throw new Error('Download failed')
      }
      await handleUnauthorized(path, res)
      return
    }

    if (!res.ok) throw new Error('Download failed')
    const blob = await res.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = filename
    a.click()
    URL.revokeObjectURL(url)
  }

  async function fetchText(path: string): Promise<string> {
    const headers: Record<string, string> = {}
    if (auth.accessToken) headers.Authorization = `Bearer ${auth.accessToken}`

    const { response: res } = await retryOnUnauthorized(
      () => {
        applyAuthHeader(headers)
        return fetch(base(path), { headers })
      },
      async () => !!auth.refreshToken && auth.refresh(),
    )

    if (!res.ok) throw new Error('Request failed')
    return res.text()
  }

  return {
    get: <T>(path: string) => request<T>(path),
    post: <T>(path: string, body?: unknown, init?: RequestInit) =>
      request<T>(path, { method: 'POST', body: body !== undefined ? JSON.stringify(body) : undefined, ...init }),
    patch: <T>(path: string, body?: unknown) => request<T>(path, { method: 'PATCH', body: JSON.stringify(body) }),
    delete: <T>(path: string) => request<T>(path, { method: 'DELETE' }),
    upload,
    download,
    fetchText,
    uploadRaw: async <T>(path: string, body: BodyInit, contentType?: string) => {
      const headers: Record<string, string> = {}
      if (contentType) headers['Content-Type'] = contentType
      if (auth.accessToken) headers.Authorization = `Bearer ${auth.accessToken}`
      await auth.ensureAccessTokenFresh()
      if (auth.accessToken) headers.Authorization = `Bearer ${auth.accessToken}`

      const { response: res, refreshSucceeded } = await retryOnUnauthorized(
        () => {
          applyAuthHeader(headers)
          return fetch(base(path), { method: 'POST', headers, body })
        },
        async () => !!auth.refreshToken && auth.refresh(),
      )

      if (res.status === 401) {
        if (!shouldLogoutOnUnauthorized(refreshSucceeded)) {
          const err = await res.json().catch(() => ({ error: 'Upload failed' }))
          throw new Error(err.error || 'Upload failed')
        }
        return handleUnauthorized(path, res)
      }
      if (!res.ok) {
        const err = await res.json().catch(() => ({ error: res.statusText }))
        throw new Error(err.error || 'Upload failed')
      }
      return res.json() as Promise<T>
    },
  }
}
