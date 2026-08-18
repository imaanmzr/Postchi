<template>
  <div class="space-y-3">
    <select v-model="local.type" class="ui-input text-sm">
      <option v-if="showInherit" value="inherit">Inherit from parent</option>
      <option value="none">No Auth</option>
      <option value="basic">Basic</option>
      <option value="bearer">Bearer Token</option>
      <option value="apikey">API Key</option>
    </select>
    <p
      v-if="local.type === 'inherit' && inheritLabel"
      class="text-xs rounded px-2 py-1.5"
      style="color: var(--text-secondary); background: var(--color-surface-2)"
    >
      {{ inheritLabel }}
    </p>
    <template v-if="local.type === 'basic'">
      <VarAutocompleteInput v-model="config.username" :collection-id="collectionId" placeholder="Username" />
      <VarAutocompleteInput v-model="config.password" :collection-id="collectionId" type="password" placeholder="Password" />
    </template>
    <template v-else-if="local.type === 'bearer'">
      <VarAutocompleteInput v-model="config.token" :collection-id="collectionId" placeholder="Token" />
    </template>
    <template v-else-if="local.type === 'apikey'">
      <VarAutocompleteInput v-model="config.key" :collection-id="collectionId" placeholder="Key" />
      <VarAutocompleteInput v-model="config.value" :collection-id="collectionId" placeholder="Value" />
      <select v-model="config.in" class="ui-input text-sm">
        <option value="header">Header</option>
        <option value="query">Query</option>
      </select>
    </template>
  </div>
</template>

<script setup lang="ts">
const props = withDefaults(defineProps<{
  collectionId?: string
  inheritLabel?: string
  showInherit?: boolean
}>(), {
  showInherit: true,
})

const model = defineModel<{ type: string; config?: Record<string, string> }>({ required: true })
const local = computed({
  get: () => model.value,
  set: (v) => { model.value = v },
})
const config = computed({
  get: () => {
    if (!model.value.config) model.value.config = {}
    return model.value.config
  },
  set: (v) => { model.value.config = v },
})
</script>
