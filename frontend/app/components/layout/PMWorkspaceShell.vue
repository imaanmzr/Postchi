<template>
  <div class="h-screen flex flex-col overflow-hidden app-bg">
    <WorkspaceChrome
      :workspace-id="workspaceId"
      :workspace-name="workspaceName"
      workspace-type="pm"
      :nav-items="navItems"
    />
    <div class="flex-1 min-h-0 overflow-hidden">
      <slot />
    </div>
  </div>
</template>

<script setup lang="ts">
const props = defineProps<{
  workspaceId: string
  workspaceName: string
}>()

const navItems = computed(() => [
  { label: 'Stories', to: `/workspaces/${props.workspaceId}/diagrams`, match: 'prefix' as const },
  { label: 'Docs', to: `/workspaces/${props.workspaceId}/docs`, match: 'prefix' as const },
])
</script>
