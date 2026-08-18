import type { VarSuggestion } from '~/utils/variableSuggestions'
import { buildVarSuggestions, filterVarSuggestions } from '~/utils/variableSuggestions'

export function useAvailableVariables(collectionId?: string) {
  const colStore = useCollectionsStore()
  const envStore = useEnvironmentsStore()
  const wsStore = useWorkspaceStore()

  const suggestions = computed(() => {
    const env = envStore.active
    return buildVarSuggestions({
      workspaceVars: wsStore.current?.variables,
      collections: colStore.collections,
      collectionId,
      envVariables: env?.variables,
    })
  })

  function filter(query: string): VarSuggestion[] {
    return filterVarSuggestions(suggestions.value, query)
  }

  return { suggestions, filter }
}
