export function safeAuthRedirect(value: unknown, fallback = '/workspaces'): string {
  if (typeof value !== 'string') return fallback
  if (!value.startsWith('/') || value.startsWith('//') || value.startsWith('/\\')) return fallback
  if (value === '/login' || value === '/register') return fallback
  return value
}
