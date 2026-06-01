<!-- sudoapi: Account contributor review workflow. -->

<template>
  <div class="min-h-screen bg-gray-50 dark:bg-dark-950">
    <div class="pointer-events-none fixed inset-0 bg-mesh-gradient"></div>
    <div class="relative min-h-screen">
      <ContributorHeader />
      <main :class="contentClass">
        <slot />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import ContributorHeader from './ContributorHeader.vue'

withDefaults(defineProps<{
  contentClass?: string
}>(), {
  contentClass: 'p-4 md:p-6 lg:p-8'
})
</script>
