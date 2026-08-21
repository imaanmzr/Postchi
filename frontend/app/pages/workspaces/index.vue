<template>
  <div class="min-h-screen app-bg p-6 max-w-4xl mx-auto">
    <div class="flex items-center justify-between mb-8 gap-4">
      <div>
        <PostchiLogo :height="26" class="mb-3" aria-hidden="true" />
        <h1 class="text-2xl font-bold tracking-tight">Workspaces</h1>
        <p class="text-sm mt-1 text-muted">
          Signed in as <span class="font-medium">{{ auth.user?.email }}</span>
          · API health:
          <span :style="{ color: healthOk ? 'var(--method-get)' : 'var(--method-delete)' }">{{ healthStatus }}</span>
        </p>
      </div>
      <div class="flex items-center gap-2 shrink-0">
        <Button @click="showAccount = true">Account</Button>
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
        <div>
          <p class="text-sm font-medium mb-2">Workspace type</p>
          <div class="grid gap-2 sm:grid-cols-3">
            <button
              v-for="option in WORKSPACE_TYPES"
              :key="option.id"
              type="button"
              class="text-left p-3 rounded-lg border transition"
              :style="newType === option.id
                ? { borderColor: 'var(--color-accent)', background: 'var(--color-surface-2)' }
                : { borderColor: 'var(--color-border)' }"
              @click="newType = option.id"
            >
              <div class="text-sm font-medium">{{ option.label }}</div>
              <div class="text-xs mt-1 text-muted">{{ option.description }}</div>
            </button>
          </div>
        </div>
        <Input v-model="newName" placeholder="Workspace name" required />
        <Input v-model="newDescription" placeholder="Description (optional)" />
        <p v-if="nameConflict" class="text-sm" style="color: var(--method-delete)">
          You already have a workspace with this name.
        </p>
        <p v-if="createError" class="text-sm" style="color: var(--method-delete)">{{ createError }}</p>
        <div class="flex gap-2">
          <Button type="submit" variant="primary" :disabled="creating || !newName.trim() || nameConflict">
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
        <div class="flex items-center gap-2">
          <div class="font-medium tracking-tight">{{ ws.name }}</div>
          <span
            class="text-[10px] uppercase tracking-wide px-2 py-0.5 rounded border shrink-0"
            style="border-color: var(--color-border); color: var(--color-text-muted)"
          >{{ workspaceTypeLabel(ws.type) }}</span>
        </div>
        <div v-if="ws.description" class="text-sm mt-1 text-muted">{{ ws.description }}</div>
        <div class="text-xs mt-2 text-muted font-mono">{{ ws.role }}</div>
      </NuxtLink>
      <p v-if="!loadError && !workspaces.length" class="text-center py-8 text-muted">
        No workspaces yet. Create one to get started.
      </p>
    </div>

    <AccountSettingsModal v-if="showAccount" @close="showAccount = false" />
  </div>
</template>

<script setup lang="ts">
import { healthUrl } from '~/utils/apiBase'
import { WORKSPACE_TYPES, workspaceTypeLabel, type WorkspaceType } from '~/utils/workspaceType'

const auth = useAuthStore()
const wsStore = useWorkspaceStore()
const config = useRuntimeConfig()
const router = useRouter()

const showCreate = ref(false)
const showAccount = ref(false)
const newName = ref('')
const newDescription = ref('')
const newType = ref<WorkspaceType>('default')
const creating = ref(false)
const createError = ref('')
const loadError = ref('')
const healthStatus = ref('checking...')
const healthOk = ref(false)

const workspaces = computed(() => wsStore.workspaces)

const nameConflict = computed(() => {
  const name = newName.value.trim().toLowerCase()
  if (!name) return false
  return workspaces.value.some(ws => ws.name.trim().toLowerCase() === name)
})

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
  newType.value = 'default'
}

async function create() {
  const name = newName.value.trim()
  if (!name || nameConflict.value) return
  creating.value = true
  createError.value = ''
  try {
    const ws = await wsStore.create(name, newDescription.value.trim(), newType.value)
    newName.value = ''
    newDescription.value = ''
    newType.value = 'default'
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
