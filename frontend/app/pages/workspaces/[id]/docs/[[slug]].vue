<template>
  <div class="h-screen flex flex-col overflow-hidden">
    <DocsBrowser :workspace-id="workspaceId" />
  </div>
</template>

<script setup lang="ts">
// Key by workspace only: slug changes must NOT remount the page, otherwise
// the docs browser loses its editor state on every navigation.
definePageMeta({
  key: route => `workspace-docs-${route.params.id}`,
})

const route = useRoute()
const workspaceId = computed(() => route.params.id as string)
</script>
