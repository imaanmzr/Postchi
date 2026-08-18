<template>
  <div class="min-h-screen app-bg p-6 max-w-4xl mx-auto">
    <div class="flex items-center justify-between mb-8 gap-4">
      <div>
        <h1 class="text-2xl font-bold tracking-tight">Workspaces</h1>
        <p class="text-sm mt-1 text-muted">
          Signed in as <span class="font-medium">{{ auth.user?.email }}</span>
          · API health:
          <span :style="{ color: healthOk ? 'var(--method-get)' : 'var(--method-delete)' }">{{ healthStatus }}</span>
        </p>
      </div>
      <div class="flex items-center gap-2 shrink-0">
        <Button variant="primary" @click="showCreate = true">New workspace</Button>
        <Button @click="signOut">Sign out</Button>
      </div>
    </div>

    <p
      v-if="loadError"
      class="mb-6 p-4 rounded-lg border text-sm"
      style="border-color: var(--method-delete); color: var(--method-delete); background: var(--color-surface-1)"
    >
      {{ loadError }}
    </p>

    <div
      v-if="showCreate"
      class="mb-6 p-4 rounded-lg border flex flex-col gap-3"
      style="border-color: var(--color-border); background: var(--color-surface-1)"
    >
      <form class="flex flex-col gap-3" @submit.prevent="create">
        <Input v-model="newName" placeholder="Workspace name" required />
        <Input v-model="newDescription" placeholder="Description (optional)" />
        <p v-if="createError" class="text-sm" style="color: var(--method-delete)">{{ createError }}</p>
        <div class="flex gap-2">
          <Button type="submit" variant="primary" :disabled="creating || !newName.trim()">
            {{ creating ? 'Creating…' : 'Create' }}
          </Button>
          <Button type="button" :disabled="creating" @click="closeCreate">Cancel</Button>
        </div>
      </form>
    </div>

    <div class="grid gap-3">
      <NuxtLink
        v-for="ws in workspaces"
        :key="ws.id"
        :to="`/workspaces/${ws.id}`"
        class="block p-4 rounded-lg border transition hover:border-accent card-link"
        style="border-color: var(--color-border); background: var(--color-surface-1)"
      >
        <div class="font-medium tracking-tight">{{ ws.name }}</div>
        <div v-if="ws.description" class="text-sm mt-1 text-muted">{{ ws.description }}</div>
        <div class="text-xs mt-2 text-muted font-mono">{{ ws.role }}</div>
      </NuxtLink>
      <p v-if="!loadError && !workspaces.length" class="text-center py-8 text-muted">
        No workspaces yet. Create one to get started.
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { healthUrl } from '~/utils/apiBase'

const auth = useAuthStore()
const wsStore = useWorkspaceStore()
const config = useRuntimeConfig()
const router = useRouter()

const showCreate = ref(false)
const newName = ref('')
const newDescription = ref('')
const creating = ref(false)
const createError = ref('')
const loadError = ref('')
const healthStatus = ref('checking...')
const healthOk = ref(false)

const workspaces = computed(() => wsStore.workspaces)

onMounted(async () => {
  try {
    const res = await fetch(healthUrl(config.public.apiUrl as string))
    const data = await res.json()
    healthOk.value = data.status === 'ok'
    healthStatus.value = healthOk.value ? 'ok' : 'degraded'
  } catch {
    healthStatus.value = 'unreachable'
  }

  try {
    await wsStore.fetchWorkspaces()
  } catch (err: unknown) {
    loadError.value = err instanceof Error ? err.message : 'Failed to load workspaces'
  }
})

function closeCreate() {
  showCreate.value = false
  createError.value = ''
}

async function create() {
  if (!newName.value.trim()) return
  creating.value = true
  createError.value = ''
  try {
    const ws = await wsStore.create(newName.value.trim(), newDescription.value.trim())
    newName.value = ''
    newDescription.value = ''
    showCreate.value = false
    await router.push(`/workspaces/${ws.id}`)
  } catch (err: unknown) {
    createError.value = err instanceof Error ? err.message : 'Failed to create workspace'
  } finally {
    creating.value = false
  }
}

async function signOut() {
  await auth.logout()
  await router.push('/login')
}
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}
.card-link:hover {
  border-color: var(--color-accent);
}
</style>
