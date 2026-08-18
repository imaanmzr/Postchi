<template>
  <Teleport to="body">
    <div class="fixed inset-0 z-50 flex items-center justify-center">
      <div class="absolute inset-0 ui-overlay" @click="$emit('close')" />
      <div class="relative z-10 w-full max-w-2xl rounded-lg p-5 max-h-[85vh] overflow-y-auto" style="background: var(--surface); border: 1px solid var(--border)">
        <h2 class="text-lg font-semibold mb-3">Workspace Settings</h2>

        <div class="flex gap-1 mb-4 p-1 rounded-md flex-wrap" style="background: var(--color-surface-2)">
          <button
            type="button"
            class="flex-1 text-sm py-1.5 rounded transition min-w-[80px]"
            :style="tab === 'general' ? { background: 'var(--surface)', fontWeight: 600 } : { color: 'var(--text-secondary)' }"
            @click="tab = 'general'"
          >General</button>
          <button
            type="button"
            class="flex-1 text-sm py-1.5 rounded transition min-w-[80px]"
            :style="tab === 'team' ? { background: 'var(--surface)', fontWeight: 600 } : { color: 'var(--text-secondary)' }"
            @click="tab = 'team'"
          >Team</button>
          <button
            type="button"
            class="flex-1 text-sm py-1.5 rounded transition min-w-[80px]"
            :style="tab === 'shares' ? { background: 'var(--surface)', fontWeight: 600 } : { color: 'var(--text-secondary)' }"
            @click="tab = 'shares'; loadShares()"
          >Shares</button>
          <button
            type="button"
            class="flex-1 text-sm py-1.5 rounded transition min-w-[80px]"
            :style="tab === 'api-sync' ? { background: 'var(--surface)', fontWeight: 600 } : { color: 'var(--text-secondary)' }"
            @click="tab = 'api-sync'"
          >API Sync</button>
          <button
            type="button"
            class="flex-1 text-sm py-1.5 rounded transition min-w-[80px]"
            :style="tab === 'documentation' ? { background: 'var(--surface)', fontWeight: 600 } : { color: 'var(--text-secondary)' }"
            @click="tab = 'documentation'"
          >Documentation</button>
        </div>

        <p v-if="actionError" class="text-sm mb-3" style="color: var(--method-delete)">{{ actionError }}</p>
        <p v-if="actionSuccess" class="text-sm mb-3" style="color: var(--method-get)">{{ actionSuccess }}</p>

        <template v-if="tab === 'general'">
          <ThemePicker class="mb-5 pb-5 border-b" style="border-color: var(--border)" />

          <label class="text-sm block mb-1">Name</label>
          <Input v-model="name" class="mb-3" />
          <label class="text-sm block mb-1">Description</label>
          <textarea v-model="description" class="ui-input w-full h-20 mb-4" />

          <div class="flex gap-2">
            <Button variant="primary" :disabled="saving" @click="save">{{ saving ? 'Saving…' : 'Save' }}</Button>
            <Button v-if="isOwner" @click="askDelete = true" style="color: var(--method-delete)">Delete workspace</Button>
          </div>
        </template>

        <template v-else-if="tab === 'shares'">
          <h3 class="font-medium mb-2">Active shares</h3>
          <p class="text-xs mb-3 text-muted">
            Share links let teammates view request and response snapshots. Workspace visibility requires sign-in.
          </p>
          <p v-if="!sharesStore.shares.length" class="text-sm mb-4" style="color: var(--text-secondary)">No active shares.</p>
          <div v-for="s in sharesStore.shares" :key="s.id" class="flex items-center gap-2 py-2 border-t text-sm" style="border-color: var(--border)">
            <div class="flex-1 min-w-0">
              <div class="truncate font-medium">{{ s.title || s.kind }}</div>
              <div class="text-xs text-muted truncate">{{ s.kind }} · {{ s.visibility }}</div>
            </div>
            <Button v-if="s.share_url" class="text-xs shrink-0" @click="copyShareUrl(s.share_url!)">Copy</Button>
            <Button v-if="s.share_url" class="text-xs shrink-0" @click="openShareUrl(s.share_url!)">Open</Button>
            <Button class="text-xs shrink-0" @click="revokeShare(s.id)">Revoke</Button>
          </div>
        </template>

        <template v-else-if="tab === 'api-sync'">
          <ApiSpecManager :workspace-id="props.workspaceId" />
        </template>

        <template v-else-if="tab === 'documentation'">
          <DocumentationSourcesPanel :workspace-id="props.workspaceId" />
        </template>

        <template v-else>
          <template v-if="isOwner">
            <h3 class="font-medium mb-1">Invite teammate</h3>
            <p class="text-xs mb-3" style="color: var(--text-secondary)">
              Send an email invite. They'll get a link to join this workspace.
            </p>
            <div class="mb-4 space-y-3">
              <div>
                <label class="text-sm block mb-1">Email address</label>
                <Input
                  v-model="inviteEmail"
                  type="email"
                  placeholder="colleague@company.com"
                  autocomplete="email"
                />
              </div>
              <div class="flex gap-2 items-end">
                <div class="flex-1 min-w-0">
                  <label class="text-sm block mb-1">Role</label>
                  <Select v-model="inviteRole">
                    <option value="viewer">Viewer</option>
                    <option value="editor">Editor</option>
                    <option value="owner">Owner</option>
                  </Select>
                </div>
                <Button
                  variant="primary"
                  class="shrink-0"
                  :disabled="inviting || !inviteEmail.trim()"
                  @click="invite"
                >
                  {{ inviting ? 'Sending…' : 'Send invite' }}
                </Button>
              </div>
            </div>

            <h3 class="font-medium mb-2">Pending invites</h3>
            <p v-if="!wsStore.pendingInvites.length" class="text-sm mb-4" style="color: var(--text-secondary)">No pending invites.</p>
            <div v-for="inv in wsStore.pendingInvites" :key="inv.id" class="flex items-center gap-2 py-2 border-t text-sm" style="border-color: var(--border)">
              <span class="flex-1">{{ inv.email }} · {{ inv.role }}</span>
              <Button @click="revokeInvite(inv.id)">Revoke</Button>
            </div>
          </template>

          <h3 class="font-medium mb-2" :class="isOwner ? 'mt-4' : ''">Members</h3>
          <p v-if="!wsStore.members.length" class="text-sm" style="color: var(--text-secondary)">No members yet.</p>
          <div
            v-for="m in wsStore.members"
            :key="m.user_id"
            class="py-3 border-t space-y-2"
            style="border-color: var(--border)"
          >
            <div class="text-sm">
              <span class="font-medium">{{ memberLabel(m) }}</span>
              <span
                v-if="m.display_name && m.email"
                class="block text-xs mt-0.5"
                style="color: var(--text-secondary)"
              >{{ m.email }}</span>
            </div>
            <template v-if="isOwner">
              <div class="flex gap-2">
                <Select
                  class="flex-1 min-w-0"
                  :model-value="m.role"
                  @update:model-value="updateRole(m.user_id, $event)"
                >
                  <option value="viewer">Viewer</option>
                  <option value="editor">Editor</option>
                  <option value="owner">Owner</option>
                </Select>
                <Button class="shrink-0" @click="askRemoveUser = m.user_id">Remove</Button>
              </div>
            </template>
            <span
              v-else
              class="text-xs px-2 py-0.5 rounded inline-block"
              style="background: var(--color-surface-2); color: var(--text-secondary)"
            >{{ m.role }}</span>
          </div>
        </template>

        <div class="mt-4 flex justify-end">
          <Button @click="$emit('close')">Close</Button>
        </div>
      </div>
    </div>

    <ConfirmDialog
      v-model:open="askDelete"
      title="Delete workspace"
      message="This will permanently delete the workspace and all its data."
      confirm-label="Delete"
      destructive
      @confirm="deleteWs"
    />
    <ConfirmDialog
      v-model:open="removeOpen"
      title="Remove member"
      message="Remove this member from the workspace?"
      confirm-label="Remove"
      destructive
      @confirm="confirmRemove"
    />
  </Teleport>
</template>

<script setup lang="ts">
import { copyToClipboard } from '~/utils/copyToClipboard'

const props = withDefaults(defineProps<{
  workspaceId: string
  initialTab?: 'general' | 'team' | 'shares' | 'api-sync' | 'documentation'
}>(), {
  initialTab: 'general',
})
const emit = defineEmits<{ close: [] }>()

const wsStore = useWorkspaceStore()
const sharesStore = useSharesStore()
const router = useRouter()
const tab = ref(props.initialTab)
const name = ref('')
const description = ref('')
const inviteEmail = ref('')
const inviteRole = ref('editor')
const askDelete = ref(false)
const askRemoveUser = ref<string | null>(null)
const saving = ref(false)
const inviting = ref(false)
const actionError = ref('')
const actionSuccess = ref('')

const isOwner = computed(() => wsStore.current?.role === 'owner')

function memberLabel(m: { email: string; display_name: string }) {
  return m.display_name || m.email || 'Unknown user'
}
const removeOpen = computed({
  get: () => !!askRemoveUser.value,
  set: (v) => { if (!v) askRemoveUser.value = null },
})

onMounted(async () => {
  const ws = await wsStore.fetchWorkspace(props.workspaceId)
  wsStore.setCurrent(ws)
  name.value = ws.name
  description.value = ws.description || ''
  await wsStore.fetchMembers(props.workspaceId)
  if (isOwner.value) await wsStore.fetchPendingInvites(props.workspaceId)
})

async function save() {
  saving.value = true
  actionError.value = ''
  actionSuccess.value = ''
  try {
    await wsStore.update(props.workspaceId, { name: name.value, description: description.value })
    actionSuccess.value = 'Workspace saved.'
  } catch (e: any) {
    actionError.value = e.message
  } finally {
    saving.value = false
  }
}

async function invite() {
  const email = inviteEmail.value.trim()
  if (!email) return
  inviting.value = true
  actionError.value = ''
  actionSuccess.value = ''
  try {
    await wsStore.addMember(props.workspaceId, email, inviteRole.value)
    inviteEmail.value = ''
    actionSuccess.value = `Invite sent to ${email}.`
  } catch (e: any) {
    actionError.value = e.message
  } finally {
    inviting.value = false
  }
}

async function updateRole(userId: string, role: string) {
  actionError.value = ''
  actionSuccess.value = ''
  try {
    await wsStore.updateMember(props.workspaceId, userId, role)
    actionSuccess.value = 'Member role updated.'
  } catch (e: any) {
    actionError.value = e.message
    await wsStore.fetchMembers(props.workspaceId)
  }
}

async function confirmRemove() {
  if (!askRemoveUser.value) return
  actionError.value = ''
  actionSuccess.value = ''
  try {
    await wsStore.removeMember(props.workspaceId, askRemoveUser.value)
    askRemoveUser.value = null
    actionSuccess.value = 'Member removed.'
  } catch (e: any) {
    actionError.value = e.message
  }
}

async function revokeInvite(id: string) {
  actionError.value = ''
  actionSuccess.value = ''
  try {
    await wsStore.revokeInvite(props.workspaceId, id)
    actionSuccess.value = 'Invite revoked.'
  } catch (e: any) {
    actionError.value = e.message
  }
}

async function revokeShare(id: string) {
  actionError.value = ''
  try {
    await sharesStore.revoke(id)
    actionSuccess.value = 'Share revoked.'
  } catch (e: any) {
    actionError.value = e.message
  }
}

async function loadShares() {
  try {
    await sharesStore.list(props.workspaceId)
  } catch (e: any) {
    actionError.value = e.message
  }
}

async function copyShareUrl(url: string) {
  await copyToClipboard(url)
  actionSuccess.value = 'Share link copied.'
}

function openShareUrl(url: string) {
  window.open(url, '_blank')
}

async function deleteWs() {
  await wsStore.delete(props.workspaceId)
  askDelete.value = false
  emit('close')
  router.push('/workspaces')
}
</script>
