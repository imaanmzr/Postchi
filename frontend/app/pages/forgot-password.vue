<template>
  <div class="min-h-screen app-bg flex items-center justify-center p-4">
    <div
      class="w-full max-w-md space-y-6 rounded-lg border p-8 shadow-md"
      style="background: var(--color-surface-1); border-color: var(--color-border)"
    >
      <div class="flex flex-col items-center text-center">
        <PostchiLogo :height="40" aria-hidden="true" />
        <h1 class="text-lg font-semibold mt-4">Reset your password</h1>
        <p class="text-sm mt-1 text-muted">
          Enter your email and we'll send you a reset link if an account exists.
        </p>
      </div>

      <p v-if="!smtpConfigured" class="text-sm text-center" style="color: var(--method-delete)">
        Password reset email is not configured on this server. Contact your administrator.
      </p>

      <form v-if="!sent" class="space-y-4" @submit.prevent="handleSubmit">
        <div>
          <label class="ui-label block mb-1">Email</label>
          <Input v-model="email" type="email" required autocomplete="email" :disabled="!smtpConfigured" />
        </div>
        <p v-if="error" class="text-sm" style="color: var(--method-delete)">{{ error }}</p>
        <Button type="submit" variant="primary" class="w-full" :disabled="loading || !smtpConfigured">
          {{ loading ? 'Please wait…' : 'Send reset link' }}
        </Button>
      </form>

      <div v-else class="text-center space-y-4">
        <p class="text-sm text-muted">
          If an account exists for <strong>{{ email }}</strong>, we've sent a password reset link.
          Check your inbox and spam folder.
        </p>
        <NuxtLink to="/login" class="accent-link text-sm">Back to sign in</NuxtLink>
      </div>

      <p v-if="!sent" class="text-sm text-center text-muted">
        Remember your password?
        <NuxtLink to="/login" class="accent-link">Sign in</NuxtLink>
      </p>
    </div>
  </div>
</template>

<script setup lang="ts">
definePageMeta({ layout: false })

const auth = useAuthStore()
const email = ref('')
const loading = ref(false)
const error = ref('')
const sent = ref(false)
const smtpConfigured = ref(true)

onMounted(async () => {
  try {
    const api = useApi()
    const cfg = await api.get<{ smtp_configured?: boolean }>('/api/config/public')
    smtpConfigured.value = cfg.smtp_configured ?? false
  } catch {
    smtpConfigured.value = false
  }
})

async function handleSubmit() {
  loading.value = true
  error.value = ''
  try {
    await auth.forgotPassword(email.value)
    sent.value = true
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
