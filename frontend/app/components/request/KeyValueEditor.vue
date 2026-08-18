<template>
  <div class="space-y-2">
    <div v-for="(row, i) in rows" :key="i" class="flex gap-2 items-center">
      <input v-model="row.enabled" type="checkbox" />
      <HeaderNameInput
        v-if="suggestHeaderNames"
        v-model="row.key"
        placeholder="Header"
        input-class="ui-input w-full"
      />
      <input
        v-else
        v-model="row.key"
        placeholder="Key"
        class="flex-1 ui-input"
      />
      <VarAutocompleteInput
        v-model="row.value"
        :collection-id="collectionId"
        placeholder="Value"
        wrapper-class="flex-1"
        input-class="ui-input w-full"
      />
      <button style="color: var(--method-delete)" class="text-xs" @click="removeRow(i)">×</button>
    </div>
    <button class="text-xs" style="color: var(--accent)" @click="addRow">+ Add</button>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  collectionId?: string
  suggestHeaderNames?: boolean
}>()

const model = defineModel<{ key: string; value: string; enabled: boolean }[]>({ required: true })
const rows = computed({
  get: () => model.value || [],
  set: (v) => { model.value = v },
})

function addRow() {
  rows.value = [...rows.value, { key: '', value: '', enabled: true }]
}

function removeRow(i: number) {
  rows.value = rows.value.filter((_, idx) => idx !== i)
}
</script>
