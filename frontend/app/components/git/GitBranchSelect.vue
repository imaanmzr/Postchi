<template>
  <div class="space-y-1">
    <div class="flex flex-wrap gap-2 items-center">
      <Select
        v-if="mode === 'list'"
        :model-value="selectValue"
        class="flex-1 min-w-[12rem]"
        :disabled="disabled"
        @update:model-value="onSelectChange"
      >
        <option value="" disabled>Select branch</option>
        <option v-for="branch in branches" :key="branch.name" :value="branch.name">
          {{ branch.name }}{{ branch.is_default ? ' (default)' : '' }}
        </option>
        <option value="__custom__">Custom branch…</option>
      </Select>
      <Input
        v-else
        :model-value="modelValue"
        class="flex-1 min-w-[12rem]"
        placeholder="Branch name"
        :disabled="disabled"
        @update:model-value="$emit('update:modelValue', $event)"
      />
      <Button
        class="text-xs shrink-0"
        type="button"
        :disabled="disabled || !canLoad || loading"
        @click="loadBranches(true)"
      >
        {{ loading ? 'Loading…' : 'Load branches' }}
      </Button>
      <Button
        v-if="mode === 'custom'"
        class="text-xs shrink-0"
        type="button"
        :disabled="disabled"
        @click="switchToList"
      >
        Pick from list
      </Button>
    </div>
    <p v-if="provider === 'GitLab' && !accessToken?.trim()" class="text-xs text-muted">
      GitLab requires a personal access token to list branches. You can still enter a custom branch name.
    </p>
    <p v-if="error" class="text-xs text-red-400">{{ error }}</p>
    <p v-else-if="cached && fetchedAt" class="text-xs text-muted">
      Branch list cached from {{ formatFetchedAt(fetchedAt) }}
    </p>
  </div>
</template>

<script setup lang="ts">
import { detectedGitProvider } from '~/utils/gitRepoForm'
import { useGitBranches, type GitSourceKind } from '~/composables/useGitBranches'

const props = withDefaults(defineProps<{
  modelValue: string
  workspaceId: string
  repoUrl: string
  accessToken?: string
  sourceId?: string
  sourceKind?: GitSourceKind
  disabled?: boolean
}>(), {
  accessToken: '',
  disabled: false,
})

const emit = defineEmits<{ 'update:modelValue': [value: string] }>()

const {
  branches,
  loading,
  error,
  cached,
  fetchedAt,
  fetchForSource,
  previewBranches,
  schedulePreview,
  scheduleForSource,
} = useGitBranches()

const mode = ref<'list' | 'custom'>('list')

const provider = computed(() => detectedGitProvider(props.repoUrl))
const canLoad = computed(() => !!props.repoUrl.trim())

const selectValue = computed(() => {
  if (mode.value === 'custom') return '__custom__'
  if (props.modelValue && branches.value.some(b => b.name === props.modelValue)) {
    return props.modelValue
  }
  if (props.modelValue) return '__custom__'
  return props.modelValue || ''
})

watch(
  () => [props.repoUrl, props.accessToken, props.sourceId, props.sourceKind] as const,
  () => {
    if (props.sourceId && props.sourceKind) {
      scheduleForSource(props.workspaceId, props.sourceKind, props.sourceId)
    } else {
      schedulePreview(props.workspaceId, props.repoUrl, props.accessToken)
    }
  },
)

watch(
  () => props.modelValue,
  (value) => {
    if (!value) return
    if (branches.value.some(b => b.name === value)) {
      mode.value = 'list'
    } else if (value !== '__custom__') {
      mode.value = 'custom'
    }
  },
  { immediate: true },
)

function onSelectChange(value: string) {
  if (value === '__custom__') {
    mode.value = 'custom'
    if (!props.modelValue) {
      emit('update:modelValue', 'main')
    }
    return
  }
  mode.value = 'list'
  emit('update:modelValue', value)
}

function switchToList() {
  mode.value = 'list'
  if (!branches.value.some(b => b.name === props.modelValue) && branches.value.length > 0) {
    const preferred = branches.value.find(b => b.is_default)?.name ?? branches.value[0]?.name
    if (preferred) emit('update:modelValue', preferred)
  }
}

async function loadBranches(refresh = false) {
  if (!canLoad.value) return
  if (props.sourceId && props.sourceKind) {
    await fetchForSource(props.workspaceId, props.sourceKind, props.sourceId, '', refresh)
  } else {
    await previewBranches(props.workspaceId, props.repoUrl, props.accessToken, '', refresh)
  }
  if (props.modelValue && !branches.value.some(b => b.name === props.modelValue)) {
    mode.value = 'custom'
  }
}

function formatFetchedAt(value: string) {
  try {
    return new Date(value).toLocaleString()
  } catch {
    return value
  }
}
</script>
