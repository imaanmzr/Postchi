<template>
  <div class="space-y-6">
    <section>
      <h4 class="text-sm font-medium mb-2">Pre Request</h4>
      <table class="w-full text-sm">
        <thead>
          <tr class="text-left" style="color: var(--text-secondary)">
            <th class="w-8 pb-2" />
            <th class="pb-2 pr-2">Name</th>
            <th class="pb-2 pr-2">Value</th>
            <th class="pb-2 pr-2">Type</th>
            <th class="pb-2 pr-2">Description</th>
            <th class="w-8 pb-2" />
          </tr>
        </thead>
        <tbody>
          <tr v-for="(row, i) in preRows" :key="'pre-' + i" class="border-t" style="border-color: var(--border)">
            <td class="py-1"><input v-model="row.enabled" type="checkbox" /></td>
            <td class="py-1 pr-2"><input v-model="row.name" class="ui-input w-full" placeholder="Name" /></td>
            <td class="py-1 pr-2">
              <input v-model="row.value" :type="row.secret ? 'password' : 'text'" class="ui-input w-full" placeholder="Value" />
            </td>
            <td class="py-1 pr-2">
              <select v-model="row.type" class="ui-input w-full">
                <option value="string">string</option>
                <option value="number">number</option>
                <option value="boolean">boolean</option>
              </select>
            </td>
            <td class="py-1 pr-2"><input v-model="row.description" class="ui-input w-full" placeholder="Description" /></td>
            <td class="py-1 flex gap-1 items-center">
              <button :title="row.secret ? 'Unmask' : 'Secret'" class="text-xs" style="color: var(--accent)" @click="row.secret = !row.secret">🔒</button>
              <button class="text-[var(--method-delete)]" @click="removePre(i)">×</button>
            </td>
          </tr>
          <tr><td colspan="6" class="py-1"><button style="color: var(--accent)" class="text-sm" @click="addPre">+ Add</button></td></tr>
        </tbody>
      </table>
    </section>

    <section>
      <h4 class="text-sm font-medium mb-2 flex items-center gap-1">
        Post Response
        <Tooltip text="Expression subset: res.status, res.body.field">
          <span class="cursor-help text-xs" style="color: var(--text-secondary)">?</span>
        </Tooltip>
      </h4>
      <table class="w-full text-sm">
        <thead>
          <tr class="text-left" style="color: var(--text-secondary)">
            <th class="w-8 pb-2" />
            <th class="pb-2 pr-2">Name</th>
            <th class="pb-2 pr-2">Expr</th>
            <th class="pb-2 pr-2">Description</th>
            <th class="w-8 pb-2" />
          </tr>
        </thead>
        <tbody>
          <tr v-for="(row, i) in postRows" :key="'post-' + i" class="border-t" style="border-color: var(--border)">
            <td class="py-1"><input v-model="row.enabled" type="checkbox" /></td>
            <td class="py-1 pr-2"><input v-model="row.name" class="ui-input w-full" placeholder="Name" /></td>
            <td class="py-1 pr-2"><input v-model="row.expr" class="ui-input w-full font-mono" placeholder="res.body.id" /></td>
            <td class="py-1 pr-2"><input v-model="row.description" class="ui-input w-full" placeholder="Description" /></td>
            <td class="py-1"><button class="text-[var(--method-delete)]" @click="removePost(i)">×</button></td>
          </tr>
          <tr><td colspan="5" class="py-1"><button style="color: var(--accent)" class="text-sm" @click="addPost">+ Add</button></td></tr>
        </tbody>
      </table>
    </section>

    <div class="flex justify-end">
      <Button variant="primary" @click="$emit('save')">Save</Button>
    </div>
  </div>
</template>

<script setup lang="ts">
export interface PreRequestVar {
  enabled: boolean
  name: string
  value: string
  type: string
  description: string
  secret: boolean
}

export interface PostResponseVar {
  enabled: boolean
  name: string
  expr: string
  description: string
}

export interface VariablesSpec {
  pre_request: PreRequestVar[]
  post_response: PostResponseVar[]
}

const model = defineModel<VariablesSpec>({ required: true })
defineEmits<{ save: [] }>()

const preRows = computed({
  get: () => model.value.pre_request,
  set: (v) => { model.value = { ...model.value, pre_request: v } },
})

const postRows = computed({
  get: () => model.value.post_response,
  set: (v) => { model.value = { ...model.value, post_response: v } },
})

function addPre() {
  preRows.value = [...preRows.value, { enabled: true, name: '', value: '', type: 'string', description: '', secret: false }]
}

function removePre(i: number) {
  preRows.value = preRows.value.filter((_, idx) => idx !== i)
}

function addPost() {
  postRows.value = [...postRows.value, { enabled: true, name: '', expr: '', description: '' }]
}

function removePost(i: number) {
  postRows.value = postRows.value.filter((_, idx) => idx !== i)
}
</script>
