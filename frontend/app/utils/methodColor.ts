export function methodColor(method: string): string {
  switch (method.toUpperCase()) {
    case 'GET':
      return 'var(--method-get)'
    case 'POST':
      return 'var(--method-post)'
    case 'PUT':
      return 'var(--method-put)'
    case 'PATCH':
      return 'var(--method-patch)'
    case 'DELETE':
      return 'var(--method-delete)'
    default:
      return 'var(--accent)'
  }
}

export function methodClass(method: string): string {
  return 'font-mono text-xs font-bold'
}
