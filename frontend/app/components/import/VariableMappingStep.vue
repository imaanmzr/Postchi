<template>
  <div class="space-y-4">
    <div>
      <label class="text-sm block mb-1">Target environment</label>
      <select
        v-model="selectedEnvId"
        class="w-full px-2 py-1.5 rounded text-sm"
        style="background: var(--color-surface-2); border: 1px solid var(--border)"
      >
        <option value="">Select environment</option>
        <template v-for="group in envGroups" :key="group.stage">
          <optgroup :label="group.label">
            <option v-for="env in group.envs" :key="env.id" :value="env.id">{{ env.name }}</option>
          </optgroup>
        </template>
      </select>
    </div>

    <div v-if="missingVars.length">
      <h4 class="text-sm font-medium mb-2">Missing variables</h4>
      <div class="space-y-2">
        <div v-for="(v, i) in missingVars" :key="v.key" class="flex gap-2 items-center text-sm">
          <span class="font-mono shrink-0" style="color: var(--accent)">{{ v.key }}</span>
          <Input v-model="v.value" placeholder="Value" class="flex-1" />
          <label class="flex items-center gap-1 shrink-0">
            <input v-model="v.is_secret" type="checkbox" />
            Secret
          </label>
          <label class="flex items-center gap-1 shrink-0">
            <input v-model="v.skip" type="checkbox" />
            Skip
          </label>
        </div>
      </div>
    </div>

    <p v-if="error" class="text-sm" style="color: var(--method-delete)">{{ error }}</p>
    <div class="flex gap-2 justify-end">
      <Button @click="$emit('cancel')">Cancel</Button>
      <Button variant="primary" :disabled="!selectedEnvId || saving" @click="submit">
        {{ saving ? 'Saving…' : 'Continue' }}
      </Button>
    </div>
  </div>
</template>

<script setup lang="ts">
const props = defineProps<{
  workspaceId: string
  placeholderNames: string[]
}>()
const emit = defineEmits<{ done: [envId: string]; cancel: [] }>()

const envStore = useEnvironmentsStore()
const selectedEnvId = ref('')
const missingVars = ref<{ key: string; value: string; is_secret: boolean; skip: boolean }[]>([])
const saving = ref(false)
const error = ref('')

const stageLabels: Record<string, string> = {
  local: 'Local', dev: 'Dev', uat: 'UAT', staging: 'Staging', prod: 'Prod', custom: 'Custom',
}

const envGroups = computed(() => {
  const groups: Record<string, typeof envStore.environments> = {}
  for (const env of envStore.environments) {
    const stage = env.stage || 'custom'
    if (!groups[stage]) groups[stage] = []
    groups[stage].push(env)
  }
  return Object.entries(groups).map(([stage, envs]) => ({
    stage,
    label: stageLabels[stage] || stage,
    envs,
  }))
})

onMounted(async () => {
  await envStore.fetch(props.workspaceId)
  if (envStore.activeId) selectedEnvId.value = envStore.activeId
  else if (envStore.environments.length) selectedEnvId.value = envStore.environments[0].id
  await loadMissing()
})

watch(selectedEnvId, loadMissing)

async function loadMissing() {
  if (!selectedEnvId.value || !props.placeholderNames.length) {
    missingVars.value = []
    return
  }
  const { missing } = await envStore.resolveVariables(selectedEnvId.value, props.placeholderNames)
  missingVars.value = missing.map(key => ({ key, value: '', is_secret: false, skip: false }))
}

async function submit() {
  if (!selectedEnvId.value) return
  saving.value = true
  error.value = ''
  try {
    const toSet = missingVars.value.filter(v => !v.skip && v.value)
    if (toSet.length) {
      await envStore.bulkSetVariables(selectedEnvId.value, toSet.map(v => ({
        key: v.key,
        value: v.value,
        is_secret: v.is_secret,
      })))
    }
    emit('done', selectedEnvId.value)
  } catch (e: any) {
    error.value = e.message
  } finally {
    saving.value = false
  }
}
</script>
