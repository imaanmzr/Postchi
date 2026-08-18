<template>
  <div class="flex flex-col h-full">
    <div
      class="flex flex-col gap-2 p-4 border-b"
      style="border-color: var(--color-border); background: var(--color-surface-1)"
    >
      <input
        v-model="local.name"
        type="text"
        class="ui-input w-full max-w-md text-sm font-medium"
        placeholder="Request name"
        @input="emitDirty"
      />
      <div class="flex gap-2 items-center">
        <select
          v-model="local.method"
          class="ui-input font-mono text-xs font-bold shrink-0 w-auto min-w-[6.25rem]"
        >
          <option v-for="m in methods" :key="m">{{ m }}</option>
        </select>
        <div class="flex-1 min-w-0">
          <VarAutocompleteInput
            v-model="local.url"
            :collection-id="local.collection_id"
            placeholder="https://api.example.com/{{path}}"
            wrapper-class="w-full"
            input-class="ui-input ui-input-editor w-full text-sm"
            @update:model-value="emitDirty"
          />
        </div>
        <Button
          variant="primary"
          class="text-xs shrink-0 inline-flex items-center gap-1.5"
          :disabled="!local.id"
          @click="$emit('execute', local)"
        >
          <Send :size="14" :stroke-width="2" aria-hidden="true" />
          <span>Send</span>
        </Button>
      </div>
      <div class="flex flex-wrap gap-2 items-center">
        <Button class="text-xs inline-flex items-center gap-1.5" @click="$emit('save', local)">
          <Save :size="14" :stroke-width="2" aria-hidden="true" />
          <span>Save</span>
        </Button>
        <Button v-if="local.id" class="text-xs inline-flex items-center gap-1.5" @click="codeOpen = true">
          <Code2 :size="14" :stroke-width="2" aria-hidden="true" />
          <span>Code</span>
        </Button>
        <ShareButton
          v-if="local.id && workspaceId"
          :workspace-id="workspaceId"
          kind="request"
          :source-id="local.id"
          :default-title="local.name"
          button-class="text-xs inline-flex items-center gap-1.5"
        >
          <Share2 :size="14" :stroke-width="2" aria-hidden="true" />
          <span>Share</span>
        </ShareButton>
        <Button
          v-if="local.id && !local.template_id"
          class="text-xs inline-flex items-center gap-1.5"
          @click="createVariant"
        >
          <GitBranch :size="14" :stroke-width="2" aria-hidden="true" />
          <span>Variant</span>
        </Button>
      </div>
    </div>

    <div
      v-if="local.is_template && childCount > 0"
      class="px-4 py-1.5 text-xs border-b text-muted font-mono"
      style="border-color: var(--color-border); background: var(--color-surface-2)"
    >
      Used by {{ childCount }} variant{{ childCount === 1 ? '' : 's' }}
    </div>

    <CodeSnippetDialog
      :open="codeOpen"
      :request-id="local.id"
      @close="codeOpen = false"
    />

    <SubTabBar v-model="activeTab" :tabs="tabItems" class="px-2 bg-bg" />

    <div class="flex-1 ui-scroll-y p-4">
      <div v-if="activeTab === 'Params'">
        <FieldInheritancePill field="params" :request="local" @reset="resetField" />
        <KeyValueEditor v-model="local.params" :collection-id="local.collection_id" />
      </div>
      <div v-else-if="activeTab === 'Headers'">
        <FieldInheritancePill field="headers" :request="local" @reset="resetField" />
        <KeyValueEditor v-model="local.headers" :collection-id="local.collection_id" suggest-header-names />
      </div>
      <div v-else-if="activeTab === 'Body'">
        <FieldInheritancePill field="body" :request="local" @reset="resetField" />
        <BodyEditor v-model="local.body" :collection-id="local.collection_id" />
      </div>
      <div v-else-if="activeTab === 'Docs'">
        <RequestDocsPanel
          :request="local"
          :workspace-id="workspaceId"
          @save="onSaveDocs"
        />
      </div>
      <div v-else-if="activeTab === 'Auth'">
        <FieldInheritancePill field="auth" :request="local" @reset="resetField" />
        <AuthEditor
          v-model="local.auth"
          :collection-id="local.collection_id"
          :inherit-label="authInheritLabel"
        />
      </div>
      <div v-else-if="activeTab === 'Scripts'">
        <div class="space-y-6">
          <div>
            <FieldInheritancePill field="pre_request_script" :request="local" @reset="resetField" />
            <label class="ui-label block mb-2">Pre-request</label>
            <ScriptEditor v-model="local.pre_request_script" />
          </div>
          <div>
            <FieldInheritancePill field="test_script" :request="local" @reset="resetField" />
            <label class="ui-label block mb-2">Tests</label>
            <ScriptEditor v-model="local.test_script" />
          </div>
        </div>
      </div>
      <div v-else-if="activeTab === 'Settings'" class="space-y-4">
        <label class="flex items-center gap-2 text-sm">
          <input v-model="local.settings.follow_redirects" type="checkbox" />
          Follow redirects
        </label>
        <label class="flex items-center gap-2 text-sm">
          <input v-model="local.settings.verify_ssl" type="checkbox" />
          Verify SSL
        </label>
        <div>
          <label class="ui-label block mb-1">Timeout (ms)</label>
          <input v-model.number="local.settings.timeout_ms" type="number" class="ui-input w-40" />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { Code2, GitBranch, Save, Send, Share2 } from 'lucide-vue-next'
import type { RequestItem } from '~/stores/collections'
import { inheritSourceLabel, resolveRequestInheritedAuth } from '~/utils/authInheritance'

const props = defineProps<{ request: RequestItem; workspaceId?: string }>()
const emit = defineEmits<{ save: [req: RequestItem]; execute: [req: RequestItem]; dirty: [] }>()

const colStore = useCollectionsStore()
const tabsStore = useTabsStore()
const codeOpen = ref(false)
const childCount = ref(0)

const methods = ['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS']
const tabItems = [
  { id: 'Params', label: 'Params' },
  { id: 'Headers', label: 'Headers' },
  { id: 'Body', label: 'Body' },
  { id: 'Docs', label: 'Docs' },
  { id: 'Auth', label: 'Auth' },
  { id: 'Scripts', label: 'Scripts' },
  { id: 'Settings', label: 'Settings' },
]
const activeTab = ref('Body')
const local = ref({ ...props.request })

const authInheritLabel = computed(() => {
  if (local.value.auth?.type !== 'inherit') return ''
  const resolved = resolveRequestInheritedAuth(
    local.value.collection_id,
    local.value.auth,
    colStore.collections,
  )
  if (!resolved.source) return 'No auth configured in parent collection or folders.'
  return inheritSourceLabel(resolved.source)
})

watch(() => props.request.id, async () => {
  local.value = { ...props.request }
  activeTab.value = 'Body'
  if (props.request.is_template && props.request.id) {
    const children = await colStore.listChildren(props.request.id)
    childCount.value = children.length
  } else {
    childCount.value = 0
  }
})

async function createVariant() {
  if (!local.value.id) return
  const child = await colStore.createChild(local.value.id)
  childCount.value++
  const tabsStore = useTabsStore()
  tabsStore.openRequest(child)
  colStore.setActiveRequest(child)
}

async function resetField(field: string) {
  if (!local.value.id) return
  await colStore.resetField(local.value.id, field)
  local.value.overridden_fields = (local.value.overridden_fields || []).filter(f => f !== field)
}

function emitDirty() {
  emit('dirty')
}

function onSaveDocs(req: RequestItem) {
  local.value = { ...req }
  emit('save', req)
}

watch(() => props.request, (r) => { local.value = { ...r } }, { deep: true })
watch(local, () => emitDirty(), { deep: true })

watch(() => local.value.name, (name) => {
  tabsStore.updateRequestTabMeta(tabsStore.activeKey, { name })
})
watch(() => local.value.method, (method) => {
  tabsStore.updateRequestTabMeta(tabsStore.activeKey, { method })
})
</script>

<style scoped>
.text-muted {
  color: var(--color-text-muted);
}
.bg-bg {
  background: var(--color-bg);
}
</style>
