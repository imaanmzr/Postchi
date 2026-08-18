<template>
  <div class="flex flex-wrap gap-3 p-3 border-b items-end" style="border-color: var(--color-border); background: var(--color-surface-1)">
    <div>
      <label class="text-xs block mb-1">Search</label>
      <input v-model="local.q" class="ui-input text-sm w-48" placeholder="Name, URL, method…" @input="emitUpdate" />
    </div>
    <div>
      <label class="text-xs block mb-1">Method</label>
      <select v-model="local.method" class="ui-input text-sm" @change="emitUpdate">
        <option value="">All</option>
        <option v-for="m in methods" :key="m" :value="m">{{ m }}</option>
      </select>
    </div>
    <div>
      <label class="text-xs block mb-1">Tag</label>
      <select v-model="local.tag" class="ui-input text-sm" @change="emitUpdate">
        <option value="">All</option>
        <option v-for="t in tags" :key="t" :value="t">{{ t }}</option>
      </select>
    </div>
    <label class="flex items-center gap-2 text-sm pb-1">
      <input v-model="local.undocumented" type="checkbox" @change="emitUpdate" />
      Undocumented only
    </label>
  </div>
</template>

<script setup lang="ts">
const props = defineProps<{
  filters: { q: string; tag: string; method: string; undocumented: boolean; spec_id: string }
  tags: string[]
}>()
const emit = defineEmits<{ update: [filters: Partial<typeof props.filters>] }>()

const methods = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE']
const local = ref({ ...props.filters })

watch(() => props.filters, (f) => { local.value = { ...f } }, { deep: true })

function emitUpdate() {
  emit('update', { ...local.value })
}
</script>
