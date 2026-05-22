<template>
  <div class="min-h-full bg-gray-50 px-4 py-6 dark:bg-dark-950 sm:px-6 lg:px-8">
    <div class="mx-auto max-w-5xl space-y-5">
      <header class="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 class="text-2xl font-semibold text-gray-950 dark:text-white">
            {{ t('admin.modelSetting.title') }}
          </h1>
          <p class="mt-1 text-sm text-gray-600 dark:text-dark-300">
            {{ t('admin.modelSetting.description') }}
          </p>
        </div>
        <button
          type="button"
          class="btn btn-secondary h-10 gap-2 px-4 text-sm"
          :disabled="loading"
          @click="loadStatus"
        >
          <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
          <span>{{ t('common.refresh', 'Refresh') }}</span>
        </button>
      </header>

      <section class="rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <div class="border-b border-gray-200 px-4 py-3 dark:border-dark-700">
          <h2 class="text-sm font-semibold text-gray-950 dark:text-white">
            {{ t('admin.modelSetting.currentStatus') }}
          </h2>
        </div>
        <div class="grid gap-3 p-4 sm:grid-cols-2 lg:grid-cols-4">
          <Metric :label="t('admin.modelSetting.modelCount')" :value="formatNumber(status?.model_count)" />
          <Metric :label="t('admin.modelSetting.source')" :value="status?.source || '-'" />
          <Metric :label="t('admin.modelSetting.fileName')" :value="status?.file_name || '-'" />
          <Metric :label="t('admin.modelSetting.updatedAt')" :value="formatTime(status?.updated_at)" />
        </div>
        <div class="border-t border-gray-100 px-4 py-3 text-xs text-gray-500 dark:border-dark-800 dark:text-dark-400">
          <span class="font-medium">{{ t('admin.modelSetting.filePath') }}:</span>
          <span class="ml-1 break-all">{{ status?.file_path || '-' }}</span>
        </div>
      </section>

      <section class="rounded-lg border border-gray-200 bg-white p-4 shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <div class="grid gap-4 lg:grid-cols-[1fr_auto] lg:items-end">
          <label class="block">
            <span class="text-sm font-medium text-gray-700 dark:text-dark-200">
              {{ t('admin.modelSetting.uploadLabel') }}
            </span>
            <input
              ref="fileInput"
              type="file"
              accept=".csv,text/csv"
              class="mt-2 block w-full rounded-lg border border-gray-300 bg-white text-sm text-gray-700 file:mr-4 file:border-0 file:bg-gray-100 file:px-4 file:py-2 file:text-sm file:font-medium file:text-gray-700 hover:file:bg-gray-200 dark:border-dark-700 dark:bg-dark-950 dark:text-dark-200 dark:file:bg-dark-800 dark:file:text-dark-200"
              @change="onFileChange"
            />
          </label>
          <button
            type="button"
            class="btn btn-primary h-10 gap-2 px-4 text-sm"
            :disabled="!selectedFile || uploading"
            @click="submit"
          >
            <Icon name="upload" size="sm" />
            <span>{{ uploading ? t('admin.modelSetting.uploading') : t('admin.modelSetting.upload') }}</span>
          </button>
        </div>
        <p class="mt-3 text-xs text-gray-500 dark:text-dark-400">
          {{ t('admin.modelSetting.uploadHint') }}
        </p>
      </section>

      <div
        v-if="message"
        class="rounded-lg border border-emerald-200 bg-emerald-50 px-4 py-3 text-sm text-emerald-700 dark:border-emerald-900/40 dark:bg-emerald-900/10 dark:text-emerald-300"
      >
        {{ message }}
      </div>
      <div
        v-if="error"
        class="rounded-lg border border-rose-200 bg-rose-50 px-4 py-3 text-sm text-rose-700 dark:border-rose-900/40 dark:bg-rose-900/10 dark:text-rose-300"
      >
        {{ error }}
      </div>

      <section v-if="status" class="rounded-lg border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
        <div class="border-b border-gray-200 px-4 py-3 dark:border-dark-700">
          <h2 class="text-sm font-semibold text-gray-950 dark:text-white">
            {{ t('admin.modelSetting.parseSummary') }}
          </h2>
        </div>
        <div class="grid gap-3 p-4 sm:grid-cols-2 lg:grid-cols-4">
          <Metric :label="t('admin.modelSetting.loadedRows')" :value="formatNumber(status.summary.loaded_rows)" />
          <Metric :label="t('admin.modelSetting.totalRows')" :value="formatNumber(status.summary.total_rows)" />
          <Metric :label="t('admin.modelSetting.duplicateRows')" :value="formatNumber(status.summary.duplicate_rows)" />
          <Metric :label="t('admin.modelSetting.skippedRows')" :value="formatNumber(status.summary.skipped_rows)" />
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, defineComponent, h, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import modelSettingAPI, { type ModelSettingStatus } from '@/api/admin/modelSetting'
import Icon from '@/components/icons/Icon.vue'
import { extractApiErrorMessage } from '@/utils/apiError'

const Metric = defineComponent({
  props: {
    label: { type: String, required: true },
    value: { type: String, required: true },
  },
  setup(props) {
    return () =>
      h('div', { class: 'rounded-md bg-gray-50 p-3 dark:bg-dark-800' }, [
        h('div', { class: 'text-xs text-gray-500 dark:text-dark-400' }, props.label),
        h('div', { class: 'mt-1 break-words text-sm font-semibold text-gray-950 dark:text-white' }, props.value),
      ])
  },
})

const { t } = useI18n()
const status = ref<ModelSettingStatus | null>(null)
const loading = ref(false)
const uploading = ref(false)
const selectedFile = ref<File | null>(null)
const message = ref('')
const error = ref('')
const fileInput = ref<HTMLInputElement | null>(null)

const formatter = computed(() => new Intl.NumberFormat())

onMounted(loadStatus)

async function loadStatus() {
  loading.value = true
  error.value = ''
  try {
    status.value = await modelSettingAPI.getStatus()
  } catch (e) {
    error.value = extractApiErrorMessage(e, t('admin.modelSetting.loadFailed'))
  } finally {
    loading.value = false
  }
}

function onFileChange(event: Event) {
  const input = event.target as HTMLInputElement
  selectedFile.value = input.files?.[0] ?? null
  message.value = ''
  error.value = ''
}

async function submit() {
  if (!selectedFile.value) return
  uploading.value = true
  error.value = ''
  message.value = ''
  try {
    status.value = await modelSettingAPI.upload(selectedFile.value)
    message.value = t('admin.modelSetting.uploadSuccess', { count: status.value.model_count })
    selectedFile.value = null
    if (fileInput.value) fileInput.value.value = ''
  } catch (e) {
    error.value = extractApiErrorMessage(e, t('admin.modelSetting.uploadFailed'))
  } finally {
    uploading.value = false
  }
}

function formatNumber(value: number | undefined): string {
  if (value == null) return '-'
  return formatter.value.format(value)
}

function formatTime(value: string | undefined): string {
  if (!value) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return new Intl.DateTimeFormat(undefined, {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  }).format(date)
}
</script>
