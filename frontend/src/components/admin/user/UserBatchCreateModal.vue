<template>
  <BaseDialog
    :show="show"
    :title="t('admin.users.batch.title')"
    width="extra-wide"
    @close="handleClose"
  >
    <div v-if="!resultMode" class="space-y-4">
      <!-- Format hint -->
      <div
        class="rounded-md border border-blue-200 bg-blue-50 p-3 text-sm text-blue-800 dark:border-blue-900/40 dark:bg-blue-950/30 dark:text-blue-200"
      >
        <div class="font-medium">{{ t('admin.users.batch.columnsTitle') }}</div>
        <div class="mt-1 font-mono text-xs">
          email, password, username, balance, concurrency, rpm
        </div>
        <div class="mt-1 text-xs opacity-80">
          {{ t('admin.users.batch.columnsHint') }}
        </div>
      </div>

      <!-- Security notice -->
      <div
        class="rounded-md border border-amber-200 bg-amber-50 p-3 text-xs text-amber-800 dark:border-amber-900/40 dark:bg-amber-950/30 dark:text-amber-200"
      >
        {{ t('admin.users.batch.securityNotice') }}
      </div>

      <!-- File upload -->
      <div class="flex items-center gap-3">
        <label class="btn btn-secondary cursor-pointer">
          <Icon name="upload" size="sm" class="mr-2" />
          {{ t('admin.users.batch.uploadCsv') }}
          <input
            ref="fileInputRef"
            type="file"
            accept=".csv,.txt,text/csv,text/plain"
            class="hidden"
            @change="handleFileChange"
          />
        </label>
        <span v-if="fileName" class="text-xs text-gray-500 dark:text-dark-400">
          {{ fileName }}
        </span>
      </div>

      <!-- Textarea input -->
      <div>
        <label class="input-label">{{ t('admin.users.batch.pasteLabel') }}</label>
        <textarea
          v-model="rawText"
          rows="10"
          class="input font-mono text-xs"
          :placeholder="placeholderSample"
          spellcheck="false"
          @input="schedulePreview"
        />
      </div>

      <!-- Preview -->
      <div v-if="parseResult && parseResult.rows.length > 0">
        <div class="mb-2 flex items-center gap-3 text-xs">
          <span class="font-medium">{{ t('admin.users.batch.previewTitle') }}</span>
          <span class="text-green-600 dark:text-green-400">
            ✓ {{ parseResult.validCount }} {{ t('admin.users.batch.statsValid') }}
          </span>
          <span v-if="parseResult.errorCount > 0" class="text-red-600 dark:text-red-400">
            ✗ {{ parseResult.errorCount }} {{ t('admin.users.batch.statsError') }}
          </span>
          <span
            v-if="parseResult.duplicateCount > 0"
            class="text-amber-600 dark:text-amber-400"
          >
            ⚠ {{ parseResult.duplicateCount }} {{ t('admin.users.batch.statsDup') }}
          </span>
          <span v-if="parseResult.headerSkipped" class="text-gray-500 dark:text-dark-400">
            {{ t('admin.users.batch.headerSkipped') }}
          </span>
        </div>
        <div class="max-h-72 overflow-auto rounded-md border border-gray-200 dark:border-dark-700">
          <table class="w-full text-xs">
            <thead class="sticky top-0 bg-gray-50 dark:bg-dark-800">
              <tr class="text-left">
                <th class="px-2 py-1.5 font-medium">#</th>
                <th class="px-2 py-1.5 font-medium">{{ t('admin.users.batch.status') }}</th>
                <th class="px-2 py-1.5 font-medium">email</th>
                <th class="px-2 py-1.5 font-medium">username</th>
                <th class="px-2 py-1.5 font-medium">balance</th>
                <th class="px-2 py-1.5 font-medium">concurrency</th>
                <th class="px-2 py-1.5 font-medium">rpm</th>
                <th class="px-2 py-1.5 font-medium">{{ t('admin.users.batch.errorMsg') }}</th>
              </tr>
            </thead>
            <tbody>
              <tr
                v-for="row in parseResult.rows"
                :key="row.lineNo"
                :class="
                  row.valid
                    ? 'border-t border-gray-100 dark:border-dark-700'
                    : 'border-t border-red-100 bg-red-50/40 dark:border-red-900/40 dark:bg-red-950/20'
                "
              >
                <td class="px-2 py-1 text-gray-500 dark:text-dark-400">{{ row.lineNo }}</td>
                <td class="px-2 py-1">
                  <span
                    v-if="row.valid"
                    class="rounded bg-green-100 px-1.5 py-0.5 text-green-700 dark:bg-green-900/40 dark:text-green-300"
                  >
                    ✓
                  </span>
                  <span
                    v-else
                    class="rounded bg-red-100 px-1.5 py-0.5 text-red-700 dark:bg-red-900/40 dark:text-red-300"
                  >
                    ✗
                  </span>
                </td>
                <td class="truncate px-2 py-1 font-mono">{{ row.raw[0] || '' }}</td>
                <td class="px-2 py-1">{{ row.raw[2] || '' }}</td>
                <td class="px-2 py-1">{{ row.raw[3] || '' }}</td>
                <td class="px-2 py-1">{{ row.raw[4] || '' }}</td>
                <td class="px-2 py-1">{{ row.raw[5] || '' }}</td>
                <td class="px-2 py-1 text-red-700 dark:text-red-400">
                  {{ row.errorCode ? localizeError(row.errorCode, row.errorMsg) : '' }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- Skip-on-error toggle -->
      <div v-if="parseResult && parseResult.rows.length > 0" class="flex items-center gap-2">
        <input
          id="batch-skip-on-error"
          v-model="skipOnError"
          type="checkbox"
          class="h-4 w-4 rounded border-gray-300 text-primary-600"
        />
        <label for="batch-skip-on-error" class="text-sm">
          {{ t('admin.users.batch.skipOnError') }}
        </label>
      </div>
    </div>

    <!-- Result panel -->
    <div v-else class="space-y-4">
      <div
        class="rounded-md border p-3 text-sm"
        :class="
          result && result.failed === 0
            ? 'border-green-200 bg-green-50 text-green-800 dark:border-green-900/40 dark:bg-green-950/30 dark:text-green-200'
            : 'border-amber-200 bg-amber-50 text-amber-800 dark:border-amber-900/40 dark:bg-amber-950/30 dark:text-amber-200'
        "
      >
        <div class="font-medium">{{ t('admin.users.batch.resultTitle') }}</div>
        <div class="mt-1">
          {{
            t('admin.users.batch.resultSummary', {
              created: result?.created ?? 0,
              failed: result?.failed ?? 0,
              total: result?.total ?? 0
            })
          }}
          <span v-if="result?.aborted"> · {{ t('admin.users.batch.resultAborted') }}</span>
        </div>
      </div>

      <div class="max-h-72 overflow-auto rounded-md border border-gray-200 dark:border-dark-700">
        <table class="w-full text-xs">
          <thead class="sticky top-0 bg-gray-50 dark:bg-dark-800">
            <tr class="text-left">
              <th class="px-2 py-1.5 font-medium">#</th>
              <th class="px-2 py-1.5 font-medium">{{ t('admin.users.batch.status') }}</th>
              <th class="px-2 py-1.5 font-medium">email</th>
              <th class="px-2 py-1.5 font-medium">{{ t('admin.users.batch.errorMsg') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="r in result?.results || []"
              :key="r.index"
              :class="
                r.success
                  ? 'border-t border-gray-100 dark:border-dark-700'
                  : 'border-t border-red-100 bg-red-50/40 dark:border-red-900/40 dark:bg-red-950/20'
              "
            >
              <td class="px-2 py-1 text-gray-500 dark:text-dark-400">{{ r.index }}</td>
              <td class="px-2 py-1">
                <span
                  v-if="r.success"
                  class="rounded bg-green-100 px-1.5 py-0.5 text-green-700 dark:bg-green-900/40 dark:text-green-300"
                >
                  ✓
                </span>
                <span
                  v-else
                  class="rounded bg-red-100 px-1.5 py-0.5 text-red-700 dark:bg-red-900/40 dark:text-red-300"
                >
                  ✗
                </span>
              </td>
              <td class="truncate px-2 py-1 font-mono">{{ r.email }}</td>
              <td class="px-2 py-1 text-red-700 dark:text-red-400">
                {{ r.error_code ? localizeError(r.error_code, r.error_msg) : '' }}
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button
          v-if="!resultMode"
          type="button"
          class="btn btn-secondary"
          @click="handleClose"
        >
          {{ t('common.cancel') }}
        </button>
        <button
          v-if="!resultMode"
          type="button"
          :disabled="!canSubmit || submitting"
          class="btn btn-primary"
          @click="submit"
        >
          <span v-if="submitting">{{ t('admin.users.batch.submitting') }}</span>
          <span v-else>
            {{
              t('admin.users.batch.submitWithCount', { count: parseResult?.validCount ?? 0 })
            }}
          </span>
        </button>
        <button v-else type="button" class="btn btn-primary" @click="handleClose">
          {{ t('common.close') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import BaseDialog from '@/components/common/BaseDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'
import { adminAPI } from '@/api/admin'
import type { BatchCreateUsersResponse } from '@/api/admin/users'
import { parseBatchUserCsv, type BatchUserParseResult } from '@/utils/batchUserCsv'

const props = defineProps<{ show: boolean }>()
const emit = defineEmits(['close', 'success'])

const { t } = useI18n()
const appStore = useAppStore()

const rawText = ref('')
const fileName = ref('')
const fileInputRef = ref<HTMLInputElement | null>(null)
const parseResult = ref<BatchUserParseResult | null>(null)
const skipOnError = ref(true)
const submitting = ref(false)
const result = ref<BatchCreateUsersResponse | null>(null)
const resultMode = computed(() => result.value !== null)

const placeholderSample =
  'a@example.com,passwd1,Alice,10,2,60\nb@example.com,passwd2,Bob,0,1,0'

const canSubmit = computed(
  () => !!parseResult.value && parseResult.value.validCount > 0 && (skipOnError.value || parseResult.value.errorCount === 0)
)

let previewTimer: number | null = null
function schedulePreview() {
  if (previewTimer !== null) window.clearTimeout(previewTimer)
  previewTimer = window.setTimeout(() => {
    parseResult.value = parseBatchUserCsv(rawText.value)
  }, 150)
}

function handleFileChange(e: Event) {
  const target = e.target as HTMLInputElement
  const file = target.files?.[0]
  if (!file) return
  const reader = new FileReader()
  reader.onload = () => {
    rawText.value = String(reader.result || '')
    fileName.value = file.name
    parseResult.value = parseBatchUserCsv(rawText.value)
  }
  reader.readAsText(file)
}

function localizeError(code: string, fallback?: string): string {
  const key = `admin.users.batch.errorCodes.${code}`
  const localized = t(key)
  return localized === key ? fallback || code : localized
}

async function submit() {
  if (!parseResult.value || parseResult.value.validCount === 0) return
  const users = parseResult.value.rows
    .filter((r) => r.valid && r.row)
    .map((r) => r.row!)
  submitting.value = true
  try {
    const resp = await adminAPI.users.batchCreate({ users, skip_on_error: skipOnError.value })
    result.value = resp
    if (resp.created > 0) {
      appStore.showSuccess(
        t('admin.users.batch.resultSummary', {
          created: resp.created,
          failed: resp.failed,
          total: resp.total
        })
      )
      emit('success')
    }
    if (resp.failed > 0 && resp.created === 0) {
      appStore.showError(t('admin.users.batch.allFailed'))
    }
  } catch (e: any) {
    appStore.showError(e?.message || t('admin.users.batch.submitFailed'))
  } finally {
    submitting.value = false
  }
}

function handleClose() {
  emit('close')
}

watch(
  () => props.show,
  (v) => {
    if (v) {
      rawText.value = ''
      fileName.value = ''
      parseResult.value = null
      skipOnError.value = true
      result.value = null
      submitting.value = false
      if (fileInputRef.value) fileInputRef.value.value = ''
    }
  }
)
</script>
