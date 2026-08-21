<template>
  <Teleport to="body">
    <div class="fixed inset-0 z-50 flex items-center justify-center">
      <div class="absolute inset-0 ui-overlay" @click="$emit('close')" />
      <div
        class="relative z-10 w-full max-w-md rounded-lg p-5"
        style="background: var(--surface); border: 1px solid var(--border)"
      >
        <h2 class="text-lg font-semibold mb-1">Account</h2>
        <p class="text-sm mb-4 text-muted">
          Signed in as <span class="font-medium">{{ auth.user?.email }}</span>
        </p>

        <h3 class="font-medium text-sm mb-2">Change password</h3>
        <form class="space-y-3" @submit.prevent="submit">
          <div>
            <label class="text-sm block mb-1">Current password</label>
            <Input v-model="currentPassword" type="password" autocomplete="current-password" required />
          </div>
          <div>
            <label class="text-sm block mb-1">New password</label>
            <Input v-model="newPassword" type="password" autocomplete="new-password" required />
          </div>
          <div>
            <label class="text-sm block mb-1">Confirm new password</label>
            <Input v-model="confirmPassword" type="password" autocomplete="new-password" required />
          </div>
          <p v-if="mismatch" class="text-sm" style="color: var(--method-delete)">
            New passwords do not match.
          </p>
          <p v-if="error" class="text-sm" style="color: var(--method-delete)">{{ error }}</p>
          <p v-if="success" class="text-sm" style="color: var(--method-get)">{{ success }}</p>
          <div class="flex gap-2 justify-end pt-2">
            <Button @click="$emit('close')">Close</Button>
            <Button
              type="submit"
              variant="primary"
              :disabled="saving || mismatch || !currentPassword || !newPassword || !confirmPassword"
            >
              {{ saving ? 'Updating…' : 'Update password' }}
            </Button>
          </div>
        </form>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
const emit = defineEmits<{ close: [] }>()

const auth = useAuthStore()
const currentPassword = ref('')
const newPassword = ref('')
const confirmPassword = ref('')
const saving = ref(false)
const error = ref('')
const success = ref('')

const mismatch = computed(() => confirmPassword.value.length > 0 && newPassword.value !== confirmPassword.value)

async function submit() {
  if (mismatch.value) return
  saving.value = true
  error.value = ''
  success.value = ''
  try {
    await auth.changePassword(currentPassword.value, newPassword.value)
    currentPassword.value = ''
    newPassword.value = ''
    confirmPassword.value = ''
    success.value = 'Password updated.'
    emit('close')
  } catch (e: unknown) {
    error.value = e instanceof Error ? e.message : 'Failed to update password'
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}
</style>
