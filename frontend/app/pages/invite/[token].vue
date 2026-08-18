<template>
  <div class="min-h-screen flex items-center justify-center p-4">
    <div class="w-full max-w-md rounded-lg border p-6 space-y-4" style="background: var(--surface); border-color: var(--border)">
      <h1 class="text-xl font-semibold">Join workspace</h1>
      <p v-if="preview" class="text-sm" style="color: var(--text-secondary)">
        You've been invited to <strong>{{ preview.workspace_name }}</strong> as
        <span class="font-mono">{{ preview.email }}</span>
      </p>
      <p v-if="error" class="text-sm" style="color: var(--method-delete)">{{ error }}</p>

      <template v-if="preview && !preview.expired">
        <template v-if="needsPassword">
          <Input v-model="displayName" placeholder="Display name" />
          <Input v-model="password" type="password" placeholder="Set password" />
          <Button variant="primary" class="w-full" :disabled="loading" @click="accept">Create account & join</Button>
        </template>
        <template v-else-if="preview.user_exists">
          <Button variant="primary" class="w-full" :disabled="loading" @click="acceptExisting">Accept invite</Button>
          <NuxtLink to="/login" class="block text-center text-sm" style="color: var(--accent)">Sign in instead</NuxtLink>
        </template>
      </template>
      <p v-else-if="preview?.expired" class="text-sm" style="color: var(--method-delete)">
        This invite has expired. Ask the workspace owner to send a new one.
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
definePageMeta({ layout: false })

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const token = computed(() => route.params.token as string)

const preview = ref<{ email: string; workspace_name: string; expired: boolean; user_exists: boolean } | null>(null)
const needsPassword = ref(false)
const password = ref('')
const displayName = ref('')
const error = ref('')
const loading = ref(false)

onMounted(async () => {
  try {
    const api = useApi()
    preview.value = await api.get(`/api/invites/${token.value}`)
    needsPassword.value = !preview.value?.user_exists
  } catch (e: any) {
    error.value = e.message || 'Invalid invite'
  }
})

async function accept() {
  loading.value = true
  error.value = ''
  try {
    const api = useApi()
    const res = await api.post<any>(`/api/invites/${token.value}/accept`, {
      password: password.value,
      display_name: displayName.value,
    })
    if (res.requires_password) {
      needsPassword.value = true
      return
    }
    if (res.tokens) {
      auth.setSession(res.user, res.tokens.access_token, res.tokens.refresh_token)
    }
    router.push(`/workspaces/${res.workspace_id}`)
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function acceptExisting() {
  loading.value = true
  try {
    const api = useApi()
    const res = await api.post<any>(`/api/invites/${token.value}/accept`, {})
    if (res.requires_login) {
      router.push('/login')
      return
    }
    router.push(`/workspaces/${res.workspace_id}`)
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}
</script>
