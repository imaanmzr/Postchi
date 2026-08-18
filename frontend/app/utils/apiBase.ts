/** Browser-reachable API base. Empty string = same-origin (relative /api/...). */
export function apiBaseUrl(configured?: string): string {
  const v = (configured ?? '').trim()
  return v.replace(/\/$/, '')
}

export function apiUrl(configured: string | undefined, path: string): string {
  const base = apiBaseUrl(configured)
  return base ? `${base}${path}` : path
}

/** Health lives at API root (/health), not under /api — proxied in dev when apiUrl is empty. */
export function healthUrl(configured?: string | undefined): string {
  return apiUrl(configured, '/health')
}

export function wsUrl(configured: string | undefined, path: string, accessToken?: string): string {
  const base = apiBaseUrl(configured)
  let url: URL
  if (base) {
    const httpBase = base.startsWith('http') ? base : `${window.location.protocol}//${window.location.host}${base}`
    const wsBase = httpBase.replace(/^http/, 'ws')
    url = new URL(path, wsBase.endsWith('/') ? wsBase : `${wsBase}/`)
  } else {
    url = new URL(path, window.location.origin)
    url.protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  }
  if (accessToken) {
    url.searchParams.set('access_token', accessToken)
  }
  return url.toString()
}
