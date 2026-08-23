/** Returns JWT `exp` claim in seconds since epoch, or null if unavailable. */
export function getAccessTokenExpiry(token: string): number | null {
  try {
    const payload = token.split('.')[1]
    if (!payload) return null
    const json = JSON.parse(atob(payload.replace(/-/g, '+').replace(/_/g, '/')))
    return typeof json.exp === 'number' ? json.exp : null
  } catch {
    return null
  }
}

/** True when the access token is missing or within `skewSeconds` of expiry. */
export function isAccessTokenExpired(token: string, skewSeconds = 30): boolean {
  if (!token) return true
  const exp = getAccessTokenExpiry(token)
  if (!exp) return false
  return Date.now() >= (exp - skewSeconds) * 1000
}
