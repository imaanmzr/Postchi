<template>
  <Teleport to="body">
    <div v-if="open" class="fixed inset-0 z-50 flex items-center justify-center">
      <div class="absolute inset-0 ui-overlay" @click="close" />
      <div class="relative z-10 w-full max-w-md rounded-lg p-5" style="background: var(--surface); border: 1px solid var(--border)">
        <h2 class="text-lg font-semibold mb-1">{{ dialogTitle }}</h2>
        <p class="text-xs mb-4 text-muted">{{ dialogHint }}</p>

        <template v-if="!createdShare">
          <div class="space-y-3 mb-4">
            <div>
              <label class="text-sm block mb-1">Visibility</label>
              <Select v-model="visibility">
                <option value="workspace">Workspace members (team)</option>
                <option value="link">Anyone with link</option>
              </Select>
              <p v-if="visibility === 'workspace'" class="text-[10px] mt-1 text-muted">
                Teammates in this workspace can view this share when signed in.
              </p>
            </div>
            <div>
              <label class="text-sm block mb-1">Expires after (hours, optional)</label>
              <Input v-model.number="ttlHours" type="number" placeholder="Never" />
            </div>
            <div>
              <label class="text-sm block mb-1">Title (optional)</label>
              <Input v-model="title" />
            </div>
          </div>
          <p v-if="error" class="text-sm mb-3" style="color: var(--method-delete)">{{ error }}</p>
          <div class="flex gap-2 justify-end">
            <Button @click="close">Cancel</Button>
            <Button variant="primary" :disabled="loading" @click="create">{{ loading ? 'Creating…' : 'Create link' }}</Button>
          </div>
        </template>

        <template v-else>
          <p class="text-sm mb-2" style="color: var(--text-secondary)">Share link created. Send this to your teammates:</p>
          <div class="flex gap-2 mb-4">
            <Input :model-value="createdShare.share_url || ''" readonly class="flex-1 font-mono text-xs" />
            <Button @click="copyUrl">{{ copied ? 'Copied' : 'Copy' }}</Button>
          </div>
          <div class="flex justify-end gap-2">
            <Button v-if="createdShare.share_url" @click="openInBrowser">Open</Button>
            <Button @click="close">Done</Button>
          </div>
        </template>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { copyToClipboard } from '~/utils/copyToClipboard'
import type { Share } from '~/stores/shares'

const props = defineProps<{
  open: boolean
  workspaceId: string
  kind: 'request' | 'history' | 'catalog'
  sourceId: string
  defaultTitle?: string
}>()
const emit = defineEmits<{ close: [] }>()

const sharesStore = useSharesStore()
const visibility = ref('workspace')
const ttlHours = ref<number | undefined>()
const title = ref(props.defaultTitle || '')
const loading = ref(false)
const error = ref('')
const createdShare = ref<Share | null>(null)
const copied = ref(false)

const dialogTitle = computed(() => {
  if (props.kind === 'catalog') return 'Share API catalog'
  if (props.kind === 'history') return 'Share request & response'
  return 'Share request'
})

const dialogHint = computed(() => {
  if (props.kind === 'catalog') {
    return 'Creates a read-only snapshot of collection API documentation for frontend, backend, and QA.'
  }
  if (props.kind === 'history') {
    return 'Creates a snapshot of the request and response that teammates can view or import.'
  }
  return 'Creates a snapshot of the request that teammates can view or import.'
})

watch(() => props.open, (v) => {
  if (v) {
    title.value = props.defaultTitle || ''
    visibility.value = 'workspace'
    createdShare.value = null
    error.value = ''
    copied.value = false
  }
})

async function create() {
  loading.value = true
  error.value = ''
  try {
    createdShare.value = await sharesStore.create({
      kind: props.kind,
      source_id: props.sourceId,
      workspace_id: props.workspaceId,
      title: title.value || undefined,
      visibility: visibility.value,
      ttl_hours: ttlHours.value || undefined,
    })
  } catch (e: any) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function copyUrl() {
  if (!createdShare.value?.share_url) return
  const ok = await copyToClipboard(createdShare.value.share_url)
  if (ok) {
    copied.value = true
    setTimeout(() => { copied.value = false }, 1500)
  }
}

function openInBrowser() {
  if (!createdShare.value?.share_url) return
  window.open(createdShare.value.share_url, '_blank')
}

function close() {
  emit('close')
}
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}
</style>
