<template>
  <div class="min-h-screen app-bg flex items-center justify-center p-4">
    <div
      class="w-full max-w-md space-y-6 rounded-lg border p-8 shadow-md"
      style="background: var(--color-surface-1); border-color: var(--color-border)"
    >
      <div class="text-center">
        <h1 class="text-2xl font-bold tracking-tight">Postchi</h1>
        <p class="text-sm mt-1 text-muted">Self-hosted API client</p>
      </div>

      <div class="text-center">
        <h2 class="text-lg font-semibold">{{ mode === 'register' ? 'Create your account' : 'Welcome back' }}</h2>
        <p class="text-sm mt-1 text-muted">
          {{ mode === 'register' ? 'Set up a new account to get started.' : 'Sign in to your workspaces.' }}
        </p>
      </div>

      <form class="space-y-4" @submit.prevent="handleSubmit">
        <div v-if="mode === 'register'">
          <label class="ui-label block mb-1">Display name</label>
          <Input v-model="displayName" type="text" placeholder="How should we call you?" />
        </div>
        <div>
          <label class="ui-label block mb-1">Email</label>
          <Input v-model="email" type="email" required autocomplete="email" />
        </div>
        <div>
          <label class="ui-label block mb-1">Password</label>
          <Input
            v-model="password"
            type="password"
            required
            :autocomplete="mode === 'register' ? 'new-password' : 'current-password'"
          />
        </div>
        <p v-if="error" class="text-sm" style="color: var(--method-delete)">{{ error }}</p>
        <Button type="submit" variant="primary" class="w-full" :disabled="loading">
          {{ loading ? 'Please wait…' : (mode === 'register' ? 'Create account' : 'Sign in') }}
        </Button>
      </form>

      <p class="text-sm text-center text-muted">
        <template v-if="mode === 'register'">
          Already have an account?
          <NuxtLink to="/login" class="accent-link">Sign in</NuxtLink>
        </template>
        <template v-else>
          New here?
          <NuxtLink to="/register" class="accent-link">Create an account</NuxtLink>
        </template>
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
const props = defineProps<{
  mode: 'login' | 'register'
}>()

const auth = useAuthStore()
const router = useRouter()
const email = ref('')
const password = ref('')
const displayName = ref('')
const loading = ref(false)
const error = ref('')

async function handleSubmit() {
  loading.value = true
  error.value = ''
  try {
    if (props.mode === 'register') {
      await auth.register(email.value, password.value, displayName.value || email.value)
    } else {
      await auth.login(email.value, password.value)
    }
    await router.push('/workspaces')
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Something went wrong'
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}
.accent-link {
  color: var(--color-accent);
}
</style>
