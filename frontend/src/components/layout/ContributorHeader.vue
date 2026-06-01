<!-- sudoapi: Account contributor review workflow. -->

<template>
  <header class="glass sticky top-0 z-30 border-b border-gray-200/50 dark:border-dark-700/50">
    <div class="flex h-16 items-center justify-between px-4 md:px-6">
      <div class="min-w-0">
        <h1 class="truncate text-lg font-semibold text-gray-900 dark:text-white">
          {{ pageTitle }}
        </h1>
        <p v-if="pageDescription" class="truncate text-xs text-gray-500 dark:text-dark-400">
          {{ pageDescription }}
        </p>
      </div>

      <div class="flex items-center gap-3">
        <LocaleSwitcher />

        <div v-if="user" ref="dropdownRef" class="relative">
          <button
            type="button"
            class="flex items-center gap-2 rounded-xl p-1.5 transition-colors hover:bg-gray-100 dark:hover:bg-dark-800"
            aria-label="User Menu"
            @click="toggleDropdown"
          >
            <div class="flex h-8 w-8 items-center justify-center overflow-hidden rounded-xl bg-gradient-to-br from-primary-500 to-primary-600 text-sm font-medium text-white shadow-sm">
              <img
                v-if="avatarUrl"
                :src="avatarUrl"
                :alt="displayName"
                class="h-full w-full object-cover"
              >
              <span v-else>{{ userInitials }}</span>
            </div>
            <div class="hidden text-left md:block">
              <div class="text-sm font-medium text-gray-900 dark:text-white">
                {{ displayName }}
              </div>
              <div class="text-xs text-gray-500 dark:text-dark-400">
                {{ user.email }}
              </div>
            </div>
            <Icon name="chevronDown" size="sm" class="hidden text-gray-400 md:block" />
          </button>

          <transition name="dropdown">
            <div v-if="dropdownOpen" class="dropdown right-0 mt-2 w-56">
              <div class="border-b border-gray-100 px-4 py-3 dark:border-dark-700">
                <div class="truncate text-sm font-medium text-gray-900 dark:text-white">
                  {{ displayName }}
                </div>
                <div class="truncate text-xs text-gray-500 dark:text-dark-400">
                  {{ user.email }}
                </div>
              </div>

              <div class="py-1">
                <button
                  type="button"
                  class="dropdown-item w-full text-red-600 hover:bg-red-50 dark:text-red-400 dark:hover:bg-red-900/20"
                  @click="handleLogout"
                >
                  <svg
                    class="h-4 w-4"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                    stroke-width="1.5"
                  >
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      d="M15.75 9V5.25A2.25 2.25 0 0013.5 3h-6a2.25 2.25 0 00-2.25 2.25v13.5A2.25 2.25 0 007.5 21h6a2.25 2.25 0 002.25-2.25V15M12 9l-3 3m0 0l3 3m-3-3h12.75"
                    />
                  </svg>
                  {{ t('nav.logout') }}
                </button>
              </div>
            </div>
          </transition>
        </div>
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { useAuthStore } from '@/stores/auth'
import Icon from '@/components/icons/Icon.vue'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'

const route = useRoute()
const router = useRouter()
const { t } = useI18n()
const authStore = useAuthStore()

const user = computed(() => authStore.user)
const dropdownOpen = ref(false)
const dropdownRef = ref<HTMLElement | null>(null)
const avatarUrl = computed(() => user.value?.avatar_url?.trim() || '')

const pageTitle = computed(() => {
  const titleKey = route.meta.titleKey as string | undefined
  if (titleKey) return t(titleKey)
  return (route.meta.title as string | undefined) || ''
})

const pageDescription = computed(() => {
  const descriptionKey = route.meta.descriptionKey as string | undefined
  if (descriptionKey) return t(descriptionKey)
  return (route.meta.description as string | undefined) || ''
})

const displayName = computed(() => {
  if (!user.value) return ''
  return user.value.username || user.value.email?.split('@')[0] || ''
})

const userInitials = computed(() => {
  if (!user.value) return ''
  if (user.value.username) return user.value.username.substring(0, 2).toUpperCase()
  const localPart = user.value.email?.split('@')[0] || ''
  return localPart.substring(0, 2).toUpperCase()
})

function toggleDropdown() {
  dropdownOpen.value = !dropdownOpen.value
}

function closeDropdown() {
  dropdownOpen.value = false
}

async function handleLogout() {
  closeDropdown()
  try {
    await authStore.logout()
  } catch (error) {
    console.error('Logout error:', error)
  }
  await router.push('/contributor/login')
}

function handleClickOutside(event: MouseEvent) {
  if (dropdownRef.value && !dropdownRef.value.contains(event.target as Node)) {
    closeDropdown()
  }
}

onMounted(() => {
  document.addEventListener('click', handleClickOutside)
})

onBeforeUnmount(() => {
  document.removeEventListener('click', handleClickOutside)
})
</script>

<style scoped>
.dropdown-enter-active,
.dropdown-leave-active {
  transition: all 0.2s ease;
}

.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: scale(0.95) translateY(-4px);
}
</style>
