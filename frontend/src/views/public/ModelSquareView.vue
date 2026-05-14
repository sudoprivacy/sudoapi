<template>
  <div
    class="relative flex min-h-screen flex-col overflow-hidden bg-gradient-to-br from-gray-50 via-primary-50/20 to-gray-100 dark:from-dark-950 dark:via-dark-900 dark:to-dark-950"
  >
    <!-- Header (复用 HomeView 简化样式) -->
    <header class="relative z-20 px-6 py-4">
      <nav class="mx-auto flex max-w-7xl items-center justify-between">
        <router-link to="/home" class="flex items-center gap-2 text-sm text-gray-600 hover:text-gray-900 dark:text-dark-300 dark:hover:text-white">
          <Icon name="arrowLeft" size="md" />
          <span class="hidden sm:inline">{{ siteName }}</span>
        </router-link>
        <div class="flex items-center gap-3">
          <LocaleSwitcher />
          <router-link
            v-if="isAuthenticated"
            :to="dashboardPath"
            class="text-sm font-medium text-primary-600 hover:underline dark:text-primary-400"
          >
            {{ t('home.dashboard') }}
          </router-link>
          <router-link
            v-else
            to="/login"
            class="rounded-full bg-gray-900 px-3 py-1 text-xs font-medium text-white hover:bg-gray-800 dark:bg-gray-800 dark:hover:bg-gray-700"
          >
            {{ t('home.login') }}
          </router-link>
        </div>
      </nav>
    </header>

    <main class="relative z-10 flex-1 px-6 py-8">
      <div class="mx-auto max-w-7xl">
        <div class="mb-8">
          <h1 class="mb-2 text-3xl font-bold text-gray-900 dark:text-white">
            {{ t('modelSquare.title') }}
          </h1>
          <p class="text-sm text-gray-600 dark:text-dark-300">
            {{ t('modelSquare.subtitle') }}
          </p>
        </div>

        <ModelSquareGrid :scope="isAuthenticated ? 'me' : 'public'" />
      </div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import Icon from '@/components/icons/Icon.vue'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import ModelSquareGrid from '@/components/models/ModelSquareGrid.vue'

const { t } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()

const siteName = computed(
  () => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API',
)
const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => (isAdmin.value ? '/admin/dashboard' : '/dashboard'))
</script>
