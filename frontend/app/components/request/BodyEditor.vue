<template>
  <div class="space-y-3">
    <select v-model="local.mode" class="ui-input text-sm">
      <option value="none">None</option>
      <option value="raw">Raw</option>
      <option value="json">JSON</option>
      <option value="graphql">GraphQL</option>
      <option value="urlencoded">x-www-form-urlencoded</option>
      <option value="form-data">Form data</option>
    </select>
    <div v-if="local.mode === 'raw' || local.mode === 'json'">
      <select v-model="local.raw_lang" class="mb-2 ui-input text-sm">
        <option value="json">JSON</option>
        <option value="xml">XML</option>
        <option value="text">Text</option>
        <option value="html">HTML</option>
      </select>
      <ScriptEditor v-model="local.raw" :lang="editorLang" />
    </div>
    <div v-else-if="local.mode === 'graphql'">
      <label class="text-xs" style="color: var(--text-secondary)">Query</label>
      <ScriptEditor v-model="graphqlQuery" lang="javascript" />
      <label class="text-xs mt-2 block" style="color: var(--text-secondary)">Variables</label>
      <ScriptEditor v-model="graphqlVars" lang="json" />
    </div>
    <div v-else-if="local.mode === 'form-data'" class="space-y-2">
      <table class="w-full text-sm">
        <thead>
          <tr class="text-left" style="color: var(--text-secondary)">
            <th class="pb-2">Key</th>
            <th class="pb-2">Type</th>
            <th class="pb-2">Value / File</th>
            <th class="w-8" />
          </tr>
        </thead>
        <tbody>
          <tr v-for="(row, i) in formRows" :key="i" class="border-t" style="border-color: var(--border)">
            <td class="py-1 pr-2"><input v-model="row.key" class="ui-input w-full" /></td>
            <td class="py-1 pr-2">
              <select v-model="row.type" class="ui-input w-full">
                <option value="text">text</option>
                <option value="file">file</option>
              </select>
            </td>
            <td class="py-1 pr-2">
              <VarAutocompleteInput
                v-if="row.type === 'text'"
                v-model="row.value"
                :collection-id="collectionId"
                wrapper-class="w-full"
                input-class="ui-input w-full"
              />
              <input v-else type="file" class="text-xs" @change="onFile(i, $event)" />
            </td>
            <td class="py-1"><button style="color: var(--method-delete)" @click="removeRow(i)">×</button></td>
          </tr>
        </tbody>
      </table>
      <Button @click="addRow">+ Add field</Button>
    </div>
    <div v-else-if="local.mode === 'urlencoded'">
      <KeyValueEditor v-model="urlEncodedRows" :collection-id="collectionId" />
    </div>
  </div>
</template>

<script setup lang="ts">
export interface FormFieldRow {
  key: string
  value: string
  enabled: boolean
  type: 'text' | 'file'
  file_name?: string
  content_type?: string
  file_data?: string
}

const props = defineProps<{ collectionId?: string }>()

const model = defineModel<{
  mode: string
  raw: string
  raw_lang: string
  form_data?: FormFieldRow[]
  urlencoded?: { key: string; value: string; enabled: boolean }[]
  graphql?: { query: string; variables: string }
}>({ required: true })

const collectionId = computed(() => props.collectionId)

const local = computed({
  get: () => model.value,
  set: (v) => { model.value = v },
})

const editorLang = computed(() => {
  if (local.value.raw_lang === 'json') return 'json'
  if (local.value.raw_lang === 'xml' || local.value.raw_lang === 'html') return 'xml'
  return 'javascript'
})

const formRows = computed({
  get: () => {
    if (!model.value.form_data?.length) {
      return [{ key: '', value: '', enabled: true, type: 'text' as const }]
    }
    return model.value.form_data
  },
  set: (v) => { model.value = { ...model.value, form_data: v } },
})

const urlEncodedRows = computed({
  get: () => model.value.urlencoded || [],
  set: (v) => { model.value = { ...model.value, urlencoded: v } },
})

const graphqlQuery = computed({
  get: () => model.value.graphql?.query || '',
  set: (v) => {
    if (!model.value.graphql) model.value.graphql = { query: '', variables: '{}' }
    model.value.graphql.query = v
  },
})

const graphqlVars = computed({
  get: () => model.value.graphql?.variables || '{}',
  set: (v) => {
    if (!model.value.graphql) model.value.graphql = { query: '', variables: '{}' }
    model.value.graphql.variables = v
  },
})

function addRow() {
  formRows.value = [...formRows.value, { key: '', value: '', enabled: true, type: 'text' }]
}

function removeRow(i: number) {
  formRows.value = formRows.value.filter((_, idx) => idx !== i)
}

async function onFile(i: number, e: Event) {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  const buf = await file.arrayBuffer()
  const b64 = btoa(String.fromCharCode(...new Uint8Array(buf)))
  const rows = [...formRows.value]
  rows[i] = {
    ...rows[i],
    type: 'file',
    file_name: file.name,
    content_type: file.type || 'application/octet-stream',
    file_data: b64,
    value: '',
  }
  formRows.value = rows
}
</script>
