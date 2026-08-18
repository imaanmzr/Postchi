<template>
  <Teleport to="body">
    <div class="fixed inset-0 z-50 flex items-center justify-center">
      <div class="absolute inset-0 ui-overlay" @click="$emit('close')" />
      <div class="relative z-10 w-full max-w-2xl max-h-[80vh] overflow-y-auto rounded-lg p-5" style="background: var(--surface); border: 1px solid var(--border)">
        <div class="flex items-center justify-between mb-4">
          <h2 class="text-lg font-semibold">Environments</h2>
          <button @click="$emit('close')">×</button>
        </div>

        <div class="flex gap-2 mb-4">
          <Input v-model="newName" placeholder="New environment name" class="flex-1" />
          <Button variant="primary" @click="create">Create</Button>
        </div>

        <div class="space-y-4">
          <div v-for="env in envStore.environments" :key="env.id" class="rounded border p-3" style="border-color: var(--border)">
            <div class="flex items-center gap-2 mb-3">
              <Input v-model="env.name" class="flex-1" />
              <Select v-model="env.stage" class="w-28">
                <option value="local">Local</option>
                <option value="dev">Dev</option>
                <option value="uat">UAT</option>
                <option value="staging">Staging</option>
                <option value="prod">Prod</option>
                <option value="custom">Custom</option>
              </Select>
              <Button variant="primary" @click="saveEnv(env)">Save</Button>
              <Button @click="deleteEnv(env.id)">Delete</Button>
            </div>
            <VarsTableEditor
              v-if="envVars[env.id]"
              v-model="envVars[env.id]"
              @save="saveEnvVars(env)"
            />
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import type { Environment } from '~/stores/environments'
import type { VariablesSpec } from '~/components/shared/VarsTableEditor'

const props = defineProps<{ workspaceId: string }>()
defineEmits<{ close: [] }>()

const envStore = useEnvironmentsStore()
const newName = ref('')
const envVars = ref<Record<string, VariablesSpec>>({})

onMounted(async () => {
  await envStore.fetch(props.workspaceId)
  for (const env of envStore.environments) {
    await loadEnv(env)
  }
})

async function loadEnv(env: Environment) {
  const full = await envStore.fetchOne(env.id)
  const i = envStore.environments.findIndex(e => e.id === env.id)
  if (i >= 0) {
    envStore.environments[i] = { ...envStore.environments[i], ...full }
    env.stage = full.stage || 'custom'
    env.name = full.name
  }
  envVars.value[env.id] = envStore.envVarsToSpec(full.variables || [])
}

async function create() {
  if (!newName.value) return
  const env = await envStore.create(props.workspaceId, newName.value)
  envVars.value[env.id] = { pre_request: [], post_response: [] }
  newName.value = ''
}

async function saveEnv(env: Environment) {
  await envStore.update(env.id, { name: env.name, stage: env.stage })
}

async function saveEnvVars(env: Environment) {
  const full = await envStore.fetchOne(env.id)
  const vars = envStore.specToEnvVars(envVars.value[env.id], full.variables || [])
  await envStore.update(env.id, { variables: vars })
}

async function deleteEnv(id: string) {
  if (!confirm('Delete environment?')) return
  await envStore.delete(id)
  delete envVars.value[id]
}
</script>
