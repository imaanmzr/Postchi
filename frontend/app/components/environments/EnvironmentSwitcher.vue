<template>
  <div class="p-2 text-sm">
    <select
      :value="envStore.activeId || ''"
      class="ui-input w-full text-xs"
      @change="onChange"
    >
      <option value="">No environment</option>
      <optgroup v-for="group in envGroups" :key="group.stage" :label="group.label">
        <option v-for="env in group.envs" :key="env.id" :value="env.id">{{ env.name }}</option>
      </optgroup>
    </select>
  </div>
</template>

<script setup lang="ts">
const props = defineProps<{ workspaceId: string }>()
const envStore = useEnvironmentsStore()

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

onMounted(() => envStore.fetch(props.workspaceId))

async function onChange(e: Event) {
  const val = (e.target as HTMLSelectElement).value
  envStore.setActive(val || null)
  await envStore.hydrateActive()
}
</script>
