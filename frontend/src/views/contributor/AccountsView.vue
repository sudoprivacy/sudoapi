<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div class="flex flex-wrap items-center gap-2">
            <SearchInput v-model="search" :placeholder="t('contributor.accounts.search')" class="w-64" @search="loadAccounts" />
            <Select v-model="platform" :options="platformOptions" class="w-40" @change="loadAccounts" />
          </div>
          <button class="btn btn-primary" @click="showCreate = true">{{ t('contributor.accounts.add') }}</button>
        </div>
      </template>

      <template #table>
        <DataTable :columns="columns" :data="accounts" :loading="loading" row-key="id">
          <template #cell-platform="{ row }">
            <PlatformTypeBadge :platform="row.platform" :type="row.type" />
          </template>
          <template #cell-review_status="{ value }">
            <span :class="reviewBadgeClass(value)">{{ reviewLabel(value) }}</span>
          </template>
          <template #cell-status="{ row }">
            <AccountStatusIndicator :account="row" />
          </template>
          <template #cell-created_at="{ value }">
            <span class="text-sm text-gray-500 dark:text-dark-400">{{ formatDateTime(value) }}</span>
          </template>
          <template #cell-actions="{ row }">
            <div class="flex items-center gap-1">
              <button class="btn btn-secondary px-2 py-1 text-xs" @click="openEdit(row)">{{ t('common.edit') }}</button>
              <button class="btn btn-secondary px-2 py-1 text-xs" @click="testAccount(row)" :disabled="testingId === row.id">
                {{ testingId === row.id ? t('common.loading') : t('admin.accounts.test') }}
              </button>
            </div>
          </template>
        </DataTable>
      </template>

      <template #pagination>
        <Pagination v-if="pagination.total > 0" :page="pagination.page" :total="pagination.total" :page-size="pagination.page_size" @update:page="handlePageChange" @update:pageSize="handlePageSizeChange" />
      </template>
    </TablePageLayout>

    <BaseDialog :show="showForm" :title="editing ? t('contributor.accounts.edit') : t('contributor.accounts.add')" width="normal" @close="closeForm">
      <form id="contributor-account-form" class="space-y-4" @submit.prevent="submitForm">
        <div>
          <label class="input-label">{{ t('admin.accounts.accountName') }}</label>
          <input v-model="form.name" class="input" required />
        </div>
        <div>
          <label class="input-label">{{ t('admin.accounts.notes') }}</label>
          <textarea v-model="form.notes" rows="3" class="input" />
        </div>
        <div class="grid grid-cols-1 gap-4 sm:grid-cols-2">
          <div>
            <label class="input-label">{{ t('admin.accounts.platform') }}</label>
            <Select v-model="form.platform" :options="platformCreateOptions" :disabled="!!editing" />
          </div>
          <div>
            <label class="input-label">{{ t('admin.accounts.accountType') }}</label>
            <Select v-model="form.type" :options="typeOptions" :disabled="!!editing" />
          </div>
        </div>
        <div>
          <label class="input-label">{{ baseUrlLabel }}</label>
          <input v-model="form.base_url" class="input" :placeholder="baseUrlPlaceholder" />
        </div>
        <div>
          <label class="input-label">{{ apiKeyLabel }}</label>
          <input v-model="form.api_key" type="password" class="input font-mono" :required="!editing" autocomplete="new-password" />
        </div>
      </form>
      <template #footer>
        <div class="flex justify-end gap-3">
          <button class="btn btn-secondary" @click="closeForm">{{ t('common.cancel') }}</button>
          <button class="btn btn-primary" type="submit" form="contributor-account-form" :disabled="submitting">
            {{ submitting ? t('common.loading') : t('common.save') }}
          </button>
        </div>
      </template>
    </BaseDialog>

    <CreateAccountModal
      :show="showCreate"
      :proxies="proxies"
      :groups="groups"
      api-scope="contributor"
      :show-groups="false"
      :enable-codex-session-import="false"
      @close="showCreate = false"
      @created="handleCreated"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import SearchInput from '@/components/common/SearchInput.vue'
import Select from '@/components/common/Select.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import PlatformTypeBadge from '@/components/common/PlatformTypeBadge.vue'
import AccountStatusIndicator from '@/components/account/AccountStatusIndicator.vue'
import CreateAccountModal from '@/components/account/CreateAccountModal.vue'
import { contributorAPI } from '@/api/contributor'
import { useAppStore } from '@/stores/app'
import { formatDateTime } from '@/utils/format'
import type { Account, AccountPlatform, AccountType, AdminGroup, Proxy } from '@/types'

const { t } = useI18n()
const appStore = useAppStore()

const accounts = ref<Account[]>([])
const loading = ref(false)
const submitting = ref(false)
const testingId = ref<number | null>(null)
const showCreate = ref(false)
const showForm = ref(false)
const editing = ref<Account | null>(null)
const proxies = ref<Proxy[]>([])
const groups = ref<AdminGroup[]>([])
const search = ref('')
const platform = ref('')
const pagination = reactive({ page: 1, page_size: 20, total: 0 })

const columns = computed(() => [
  { key: 'name', label: t('admin.accounts.columns.name') },
  { key: 'platform', label: t('admin.accounts.columns.platformType') },
  { key: 'review_status', label: t('contributor.accounts.reviewStatus') },
  { key: 'status', label: t('admin.accounts.columns.status') },
  { key: 'created_at', label: t('common.created') },
  { key: 'actions', label: t('admin.accounts.columns.actions') },
])

const platformOptions = computed(() => [
  { value: '', label: t('admin.accounts.allPlatforms') },
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'openai', label: 'OpenAI' },
  { value: 'gemini', label: 'Gemini' },
  { value: 'antigravity', label: 'Antigravity' },
])
const platformCreateOptions = computed(() => platformOptions.value.filter(o => o.value !== ''))
const typeOptions = computed(() => [
  { value: 'apikey', label: 'API Key' },
  { value: 'upstream', label: 'Upstream' },
  { value: 'bedrock', label: 'Bedrock' },
  { value: 'service_account', label: 'Service Account' },
])

const form = reactive({
  name: '',
  notes: '',
  platform: 'anthropic' as AccountPlatform,
  type: 'apikey' as AccountType,
  base_url: '',
  api_key: '',
})

const baseUrlLabel = computed(() => form.type === 'bedrock' ? 'Region / Base URL' : t('admin.accounts.baseUrl'))
const baseUrlPlaceholder = computed(() => form.platform === 'gemini' ? 'https://generativelanguage.googleapis.com' : 'https://api.anthropic.com')
const apiKeyLabel = computed(() => form.type === 'service_account' ? 'Service Account JSON' : t('admin.accounts.apiKey'))

async function loadAccounts() {
  loading.value = true
  try {
    const result = await contributorAPI.accounts.list(pagination.page, pagination.page_size, {
      platform: platform.value || undefined,
      search: search.value || undefined,
      sort_by: 'created_at',
      sort_order: 'desc',
    })
    accounts.value = result.items
    pagination.total = result.total
    pagination.page = result.page
    pagination.page_size = result.page_size
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('contributor.accounts.loadFailed'))
  } finally {
    loading.value = false
  }
}

async function loadProxies() {
  try {
    proxies.value = await contributorAPI.accounts.getProxies()
  } catch {
    proxies.value = []
  }
}

async function handleCreated() {
  showCreate.value = false
  await loadAccounts()
}

function openEdit(account: Account) {
  editing.value = account
  Object.assign(form, {
    name: account.name,
    notes: account.notes || '',
    platform: account.platform,
    type: account.type,
    base_url: String(account.credentials?.base_url || ''),
    api_key: '',
  })
  showForm.value = true
}

function closeForm() {
  showForm.value = false
  editing.value = null
}

function buildCredentials() {
  const credentials: Record<string, unknown> = {}
  if (form.base_url.trim()) credentials.base_url = form.base_url.trim()
  if (form.api_key.trim()) {
    if (form.type === 'service_account') {
      try {
        credentials.service_account = JSON.parse(form.api_key)
      } catch {
        credentials.service_account = form.api_key.trim()
      }
    } else {
      credentials.api_key = form.api_key.trim()
    }
  }
  return credentials
}

async function submitForm() {
  submitting.value = true
  try {
    const payload = {
      name: form.name.trim(),
      notes: form.notes.trim() || null,
      platform: form.platform,
      type: form.type,
      credentials: buildCredentials(),
      extra: {},
    }
    if (editing.value) {
      await contributorAPI.accounts.update(editing.value.id, payload)
      appStore.showSuccess(t('contributor.accounts.updated'))
    } else {
      await contributorAPI.accounts.create(payload)
      appStore.showSuccess(t('contributor.accounts.created'))
    }
    closeForm()
    await loadAccounts()
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('contributor.accounts.saveFailed'))
  } finally {
    submitting.value = false
  }
}

async function testAccount(account: Account) {
  testingId.value = account.id
  try {
    await contributorAPI.accounts.testAccount(account.id)
    appStore.showSuccess(t('admin.accounts.testSuccess'))
  } catch (error: any) {
    appStore.showError(error.response?.data?.detail || t('admin.accounts.testFailed'))
  } finally {
    testingId.value = null
  }
}

function handlePageChange(page: number) {
  pagination.page = page
  loadAccounts()
}

function handlePageSizeChange(pageSize: number) {
  pagination.page = 1
  pagination.page_size = pageSize
  loadAccounts()
}

function reviewLabel(value: Account['review_status']) {
  return t(`contributor.accounts.review.${value || 'pending'}`)
}

function reviewBadgeClass(value: Account['review_status']) {
  const base = 'inline-flex rounded px-2 py-0.5 text-xs font-medium'
  if (value === 'approved') return `${base} bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300`
  if (value === 'rejected') return `${base} bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300`
  return `${base} bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300`
}

onMounted(() => {
  loadAccounts()
  loadProxies()
})
</script>
