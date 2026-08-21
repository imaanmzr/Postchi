<template>
  <div class="min-h-screen app-bg flex items-center justify-center p-4">
    <div
      class="w-full max-w-md space-y-6 rounded-lg border p-8 shadow-md"
      style="background: var(--color-surface-1); border-color: var(--color-border)"
    >
      <div class="text-center">
        <h1 class="text-lg font-semibold">Choose a new password</h1>
        <p v-if="preview" class="text-sm mt-1 text-muted">
          Reset password for <span class="font-mono">{{ preview.email }}</span>
        </p>
      </div>

      <p v-if="error" class="text-sm" style="color: var(--method-delete)">{{ error }}</p>

      <template v-if="preview && !preview.expired">
        <form class="space-y-4" @submit.prevent="handleSubmit">
          <div>
            <label class="ui-label block mb-1">New password</label>
            <Input v-model="password" type="password" required autocomplete="new-password" />
          </div>
          <div>
            <label class="ui-label block mb-1">Confirm password</label>
            <Input v-model="confirmPassword" type="password" required autocomplete="new-password" />
          </div>
          <Button type="submit" variant="primary" class="w-full" :disabled="loading">
            {{ loading ? 'Please wait…' : 'Reset password' }}
          </Button>
        </form>
      </template>

      <template v-else-if="preview?.expired">
        <p class="text-sm text-center text-muted">
          This reset link has expired. Request a new one.
        </p>
        <NuxtLink to="/forgot-password" class="block text-center accent-link text-sm">Request new link</NuxtLink>
      </template>

      <NuxtLink v-if="!preview && error" to="/forgot-password" class="block text-center accent-link text-sm">
        Request a new link
      </NuxtLink>
    </div>
  </div>
</template>

<script setup lang="ts">
definePageMeta({ layout: false })

const route = useRoute()
const router = useRouter()
const auth = useAuthStore()
const token = computed(() => route.params.token as string)

const preview = ref<{ email: string; expired: boolean } | null>(null)
const password = ref('')
const confirmPassword = ref('')
const error = ref('')
const loading = ref(false)

onMounted(async () => {
  try {
    preview.value = await auth.previewResetPassword(token.value)
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Invalid reset link'
  }
})

async function handleSubmit() {
  if (password.value !== confirmPassword.value) {
    error.value = 'Passwords do not match'
    return
  }
  loading.value = true
  error.value = ''
  try {
    await auth.resetPassword(token.value, password.value)
    await router.push('/login')
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
