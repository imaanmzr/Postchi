<template>
  <div
    class="fixed right-4 bottom-20 z-[110] flex w-[min(24rem,calc(100vw-2rem))] flex-col gap-2"
    aria-live="polite"
  >
    <TransitionGroup name="pwa-prompt">
      <div
        v-if="offline"
        key="offline"
        class="flex items-start gap-2 rounded-md border px-3 py-2.5 text-sm shadow-lg pwa-prompt-offline"
        role="status"
      >
        <WifiOff :size="16" class="mt-0.5 shrink-0" aria-hidden="true" />
        <span class="min-w-0 flex-1">
          You're offline. Postchi needs a connection to sync and run requests.
        </span>
      </div>

      <div
        v-if="pwa?.needRefresh"
        key="update"
        class="flex items-start gap-2 rounded-md border px-3 py-2.5 text-sm shadow-lg pwa-prompt-update"
        role="status"
      >
        <RefreshCw :size="16" class="mt-0.5 shrink-0" aria-hidden="true" />
        <span class="min-w-0 flex-1">A new version is available.</span>
        <div class="flex shrink-0 gap-2">
          <button
            type="button"
            class="font-medium underline-offset-2 hover:underline"
            @click="reloadApp"
          >
            Reload
          </button>
          <button
            type="button"
            class="opacity-70 transition hover:opacity-100"
            aria-label="Dismiss update notification"
            @click="dismissUpdate"
          >
            Later
          </button>
        </div>
      </div>

      <div
        v-if="pwa?.showInstallPrompt"
        key="install"
        class="flex items-start gap-2 rounded-md border px-3 py-2.5 text-sm shadow-lg pwa-prompt-install"
        role="status"
      >
        <Download :size="16" class="mt-0.5 shrink-0" aria-hidden="true" />
        <span class="min-w-0 flex-1">Install Postchi for quick access from your desktop.</span>
        <div class="flex shrink-0 gap-2">
          <button
            type="button"
            class="font-medium underline-offset-2 hover:underline"
            @click="installApp"
          >
            Install
          </button>
          <button
            type="button"
            class="opacity-70 transition hover:opacity-100"
            aria-label="Dismiss install prompt"
            @click="dismissInstall"
          >
            Not now
          </button>
        </div>
      </div>
    </TransitionGroup>
  </div>
</template>

<script setup lang="ts">
import { Download, RefreshCw, WifiOff } from 'lucide-vue-next'

const { $pwa } = useNuxtApp()
const pwa = computed(() => $pwa)
const offline = ref(false)

function setOfflineState() {
  offline.value = !navigator.onLine
}

async function installApp() {
  await pwa.value?.install()
}

function dismissInstall() {
  pwa.value?.cancelInstall()
}

async function reloadApp() {
  await pwa.value?.updateServiceWorker(true)
}

async function dismissUpdate() {
  await pwa.value?.cancelPrompt()
}

onMounted(() => {
  setOfflineState()
  window.addEventListener('online', setOfflineState)
  window.addEventListener('offline', setOfflineState)
})

onUnmounted(() => {
  window.removeEventListener('online', setOfflineState)
  window.removeEventListener('offline', setOfflineState)
})
</script>

<style scoped>
.pwa-prompt-offline {
  background: color-mix(in srgb, var(--color-warning, #d97706) 14%, var(--color-surface-1));
  border-color: color-mix(in srgb, var(--color-warning, #d97706) 45%, var(--color-border));
  color: var(--color-text);
}

.pwa-prompt-update,
.pwa-prompt-install {
  background: color-mix(in srgb, var(--color-accent, #7aa2f7) 14%, var(--color-surface-1));
  border-color: color-mix(in srgb, var(--color-accent, #7aa2f7) 45%, var(--color-border));
  color: var(--color-text);
}

.pwa-prompt-enter-active,
.pwa-prompt-leave-active {
  transition: opacity 160ms ease, transform 160ms ease;
}

.pwa-prompt-enter-from,
.pwa-prompt-leave-to {
  opacity: 0;
  transform: translateY(0.5rem);
}
</style>
