const varPattern = /\{\{([^}]+)\}\}/g

export function useVariableInterpolation() {
  function interpolate(template: string, vars: Record<string, string>): string {
    return template.replace(varPattern, (_, key: string) => {
      const k = key.trim()
      return vars[k] ?? `{{${k}}}`
    })
  }

  return { interpolate }
}
