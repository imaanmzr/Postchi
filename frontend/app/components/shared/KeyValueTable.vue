<template>
  <table class="w-full text-sm">
    <thead>
      <tr class="text-left" style="color: var(--text-secondary)">
        <th class="w-8 pb-2" />
        <th v-for="col in columns" :key="col.key" class="pb-2 pr-2 font-medium">{{ col.label }}</th>
        <th class="w-8 pb-2" />
      </tr>
    </thead>
    <tbody>
      <tr v-for="(row, i) in rows" :key="i" class="border-t" style="border-color: var(--border)">
        <td class="py-1">
          <input v-model="row.enabled" type="checkbox" />
        </td>
        <td v-for="col in columns" :key="col.key" class="py-1 pr-2">
          <input
            v-model="row[col.key]"
            :type="col.type || 'text'"
            class="ui-input w-full"
            :placeholder="col.placeholder"
          />
        </td>
        <td class="py-1">
          <button class="text-[var(--method-delete)] hover:opacity-80" @click="removeRow(i)">×</button>
        </td>
      </tr>
      <tr class="border-t" style="border-color: var(--border)">
        <td />
        <td :colspan="columns.length + 1" class="py-1">
          <button class="text-sm" style="color: var(--accent)" @click="addRow">+ Add row</button>
        </td>
      </tr>
    </tbody>
  </table>
</template>

<script setup lang="ts">
export interface KVRow {
  key: string
  value: string
  enabled: boolean
}

export interface KVColumn {
  key: keyof KVRow
  label: string
  placeholder?: string
  type?: string
}

const model = defineModel<KVRow[]>({ required: true })

const columns: KVColumn[] = [
  { key: 'key', label: 'Key', placeholder: 'Key' },
  { key: 'value', label: 'Value', placeholder: 'Value' },
]

const rows = computed({
  get: () => model.value,
  set: (v) => { model.value = v },
})

function addRow() {
  model.value = [...model.value, { key: '', value: '', enabled: true }]
}

function removeRow(i: number) {
  model.value = model.value.filter((_, idx) => idx !== i)
}
</script>
