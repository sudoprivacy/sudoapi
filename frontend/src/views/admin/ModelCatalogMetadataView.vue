<!-- sudoapi: Model catalog. -->

<template>
  <AppLayout>
    <TablePageLayout>
      <template #filters>
        <div class="flex flex-col justify-between gap-4 lg:flex-row lg:items-center">
          <div class="flex flex-1 flex-wrap items-center gap-3">
            <div class="relative w-full sm:w-80">
              <Icon
                name="search"
                size="md"
                class="absolute left-3 top-1/2 -translate-y-1/2 text-gray-400 dark:text-gray-500"
              />
              <input
                v-model="searchQuery"
                type="text"
                :placeholder="t('admin.modelCatalog.metadata.searchPlaceholder')"
                class="input pl-10"
              />
            </div>
            <label class="inline-flex items-center gap-2 text-sm text-gray-600 dark:text-dark-300">
              <input
                v-model="uncoveredOnly"
                type="checkbox"
                class="h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
              />
              {{ t('admin.modelCatalog.metadata.uncoveredOnly') }}
            </label>
          </div>
          <div class="flex w-full flex-shrink-0 flex-wrap items-center justify-end gap-3 lg:w-auto">
            <EndpointConfigButton />
            <button type="button" class="btn btn-secondary" :disabled="loading" @click="loadItems">
              <Icon name="refresh" size="md" :class="loading ? 'animate-spin' : ''" />
              <span class="ml-2">{{ t('admin.modelCatalog.metadata.refresh') }}</span>
            </button>
          </div>
        </div>
      </template>

      <template #table>
        <DataTable :columns="columns" :data="filteredItems" :loading="loading">
          <template #cell-model_name="{ row }">
            <div class="min-w-0">
              <div class="flex items-center gap-2">
                <span class="truncate font-mono text-sm font-medium text-gray-900 dark:text-white">
                  {{ row.model_name }}
                </span>
                <span
                  v-if="row.override"
                  class="rounded bg-primary-500/10 px-1.5 py-0.5 text-[10px] font-medium text-primary-700 dark:text-primary-300"
                >
                  {{ t('admin.modelCatalog.metadata.overrideActive') }}
                </span>
              </div>
              <div class="mt-1 truncate text-xs text-gray-500 dark:text-dark-400">
                {{ row.metadata.display_name || row.model_name }}
              </div>
            </div>
          </template>

          <template #cell-platforms="{ row }">
            <div class="flex flex-wrap gap-1">
              <span
                v-for="platform in row.platforms"
                :key="platform"
                :class="['rounded border px-1.5 py-0.5 text-[10px] font-medium', platformBadgeClass(platform)]"
              >
                {{ platform }}
              </span>
            </div>
          </template>

          <template #cell-category="{ row }">
            <span class="text-sm text-gray-700 dark:text-dark-200">
              {{ formatCategoryLabel(row.metadata.category, row.platforms) }}
            </span>
          </template>

          <template #cell-context="{ row }">
            <div class="text-sm text-gray-700 dark:text-dark-200">
              <div>{{ formatTokens(row.metadata.context_window) }}</div>
              <div class="text-xs text-gray-400">{{ formatTokens(row.metadata.max_output) }}</div>
            </div>
          </template>

          <template #cell-missing="{ row }">
            <div class="flex max-w-xl flex-wrap gap-1">
              <span
                v-if="row.missing_fields.length === 0"
                class="rounded bg-emerald-500/10 px-1.5 py-0.5 text-[10px] font-medium text-emerald-700 dark:text-emerald-300"
              >
                OK
              </span>
              <template v-else>
                <span
                  v-for="field in row.missing_fields.slice(0, 8)"
                  :key="field"
                  class="rounded bg-amber-500/10 px-1.5 py-0.5 text-[10px] font-medium text-amber-700 dark:text-amber-300"
                >
                  {{ missingFieldLabel(field) }}
                </span>
              </template>
              <span
                v-if="row.missing_fields.length > 8"
                class="rounded bg-gray-100 px-1.5 py-0.5 text-[10px] font-medium text-gray-500 dark:bg-dark-700 dark:text-dark-400"
              >
                +{{ row.missing_fields.length - 8 }}
              </span>
            </div>
          </template>

          <template #cell-actions="{ row }">
            <div class="flex items-center gap-1">
              <button
                type="button"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-gray-100 hover:text-primary-600 dark:hover:bg-dark-700 dark:hover:text-primary-400"
                @click="openEditDialog(row)"
              >
                <Icon name="edit" size="sm" />
                <span class="text-xs">{{ t('common.edit') }}</span>
              </button>
              <button
                type="button"
                class="flex flex-col items-center gap-0.5 rounded-lg p-1.5 text-gray-500 transition-colors hover:bg-red-50 hover:text-red-600 disabled:cursor-not-allowed disabled:opacity-40 dark:hover:bg-red-900/20 dark:hover:text-red-400"
                :disabled="!row.override"
                @click="openClearDialog(row)"
              >
                <Icon name="trash" size="sm" />
                <span class="text-xs">{{ t('common.delete') }}</span>
              </button>
            </div>
          </template>

          <template #empty>
            <EmptyState
              :title="
                items.length === 0
                  ? t('admin.modelCatalog.metadata.noModels')
                  : t('admin.modelCatalog.metadata.noResults')
              "
              :description="items.length === 0 ? t('admin.modelCatalog.metadata.noModelsDesc') : ''"
            />
          </template>
        </DataTable>
      </template>
    </TablePageLayout>

    <BaseDialog :show="showDialog" :title="t('admin.modelCatalog.metadata.edit')" width="wide" @close="closeDialog">
      <form class="space-y-4" @submit.prevent="saveMetadata">
        <div>
          <label class="input-label">{{ t('admin.modelCatalog.metadata.fields.modelName') }}</label>
          <input v-model="form.model_name" type="text" class="input font-mono" readonly />
        </div>

        <div class="grid gap-4 md:grid-cols-2">
          <div>
            <label class="input-label">{{ t('admin.modelCatalog.metadata.fields.displayName') }}</label>
            <input
              v-model="form.display_name"
              type="text"
              class="input"
              :placeholder="t('admin.modelCatalog.metadata.form.displayNamePlaceholder')"
            />
          </div>
          <div>
            <label class="input-label">{{ t('admin.modelCatalog.metadata.fields.category') }}</label>
            <div ref="categoryComboboxRef" class="relative">
              <div class="relative">
                <input
                  v-model="form.category"
                  type="text"
                  class="input pr-10"
                  :placeholder="t('admin.modelCatalog.metadata.form.categoryPlaceholder')"
                  list="model-catalog-metadata-category-options"
                  @focus="categoryDropdownOpen = true"
                  @keydown.escape="categoryDropdownOpen = false"
                  @keydown.down.prevent="categoryDropdownOpen = true"
                />
                <button
                  type="button"
                  class="absolute right-2 top-1/2 -translate-y-1/2 rounded p-1 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-dark-700 dark:hover:text-dark-200"
                  @mousedown.prevent
                  @click="toggleCategoryDropdown"
                >
                  <Icon
                    name="chevronDown"
                    size="sm"
                    class="transition-transform"
                    :class="categoryDropdownOpen ? 'rotate-180' : ''"
                  />
                </button>
              </div>
              <div
                v-if="categoryDropdownOpen"
                class="absolute z-50 mt-1 max-h-56 w-full overflow-y-auto rounded-md border border-gray-200 bg-white py-1 shadow-lg dark:border-dark-700 dark:bg-dark-800"
              >
                <button
                  v-for="opt in categoryOptions"
                  :key="opt.value"
                  type="button"
                  class="flex w-full items-center justify-between gap-3 px-3 py-2 text-left text-sm transition-colors hover:bg-gray-50 dark:hover:bg-dark-700"
                  :class="
                    opt.value === form.category
                      ? 'bg-primary-50 text-primary-700 dark:bg-primary-900/20 dark:text-primary-300'
                      : 'text-gray-700 dark:text-dark-200'
                  "
                  @mousedown.prevent
                  @click="selectCategoryOption(opt.value)"
                >
                  <span class="truncate">{{ opt.label }}</span>
                  <span
                    v-if="opt.label !== opt.value"
                    class="shrink-0 font-mono text-xs text-gray-400 dark:text-dark-400"
                  >
                    {{ opt.value }}
                  </span>
                </button>
                <div v-if="categoryOptions.length === 0" class="px-3 py-2 text-sm text-gray-400 dark:text-dark-400">
                  {{ t('common.noOptionsFound') }}
                </div>
              </div>
            </div>
            <datalist id="model-catalog-metadata-category-options">
              <option v-for="opt in categoryOptions" :key="opt.value" :value="opt.value" :label="opt.label" />
            </datalist>
          </div>
          <div>
            <label class="input-label">{{ t('admin.modelCatalog.metadata.fields.modelType') }}</label>
            <input
              v-model="form.model_type"
              type="text"
              class="input"
              :placeholder="t('admin.modelCatalog.metadata.form.modelTypePlaceholder')"
              list="model-catalog-metadata-type-options"
            />
            <datalist id="model-catalog-metadata-type-options">
              <option v-for="opt in modelTypeOptions" :key="opt" :value="opt" />
            </datalist>
          </div>
        </div>

        <div>
          <label class="input-label">{{ t('admin.modelCatalog.metadata.fields.description') }}</label>
          <textarea
            v-model="form.description"
            rows="3"
            class="input"
            :placeholder="t('admin.modelCatalog.metadata.form.descriptionPlaceholder')"
          ></textarea>
        </div>

        <div class="grid gap-4 md:grid-cols-2">
          <div>
            <label class="input-label">{{ t('admin.modelCatalog.metadata.fields.contextWindow') }}</label>
            <input
              v-model.number="form.context_window"
              type="number"
              min="0"
              step="1"
              class="input"
              :placeholder="t('admin.modelCatalog.metadata.form.contextWindowPlaceholder')"
            />
          </div>
          <div>
            <label class="input-label">{{ t('admin.modelCatalog.metadata.fields.maxOutput') }}</label>
            <input
              v-model.number="form.max_output"
              type="number"
              min="0"
              step="1"
              class="input"
              :placeholder="t('admin.modelCatalog.metadata.form.maxOutputPlaceholder')"
            />
          </div>
        </div>

        <div>
          <label class="input-label">{{ t('admin.modelCatalog.metadata.fields.inputModalities') }}</label>
          <div class="flex flex-wrap gap-2">
            <label
              v-for="modality in modalityOptions"
              :key="`input-${modality.value}`"
              class="inline-flex cursor-pointer items-center gap-1.5 rounded border border-gray-200 px-2 py-1 text-xs text-gray-700 dark:border-dark-700 dark:text-dark-200"
            >
              <input
                type="checkbox"
                class="h-3.5 w-3.5 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                :checked="form.input_modalities.includes(modality.value)"
                @change="toggleListValue('input_modalities', modality.value)"
              />
              {{ modality.label }}
            </label>
          </div>
          <p class="mt-1 text-xs text-gray-400">{{ t('admin.modelCatalog.metadata.form.modalitiesHint') }}</p>
        </div>

        <div>
          <label class="input-label">{{ t('admin.modelCatalog.metadata.fields.outputModalities') }}</label>
          <div class="flex flex-wrap gap-2">
            <label
              v-for="modality in modalityOptions"
              :key="`output-${modality.value}`"
              class="inline-flex cursor-pointer items-center gap-1.5 rounded border border-gray-200 px-2 py-1 text-xs text-gray-700 dark:border-dark-700 dark:text-dark-200"
            >
              <input
                type="checkbox"
                class="h-3.5 w-3.5 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                :checked="form.output_modalities.includes(modality.value)"
                @change="toggleListValue('output_modalities', modality.value)"
              />
              {{ modality.label }}
            </label>
          </div>
          <p class="mt-1 text-xs text-gray-400">{{ t('admin.modelCatalog.metadata.form.modalitiesHint') }}</p>
        </div>

        <div>
          <label class="input-label">{{ t('admin.modelCatalog.metadata.fields.supportFlags') }}</label>
          <div class="flex flex-wrap gap-2">
            <label
              v-for="flag in supportFlagOptions"
              :key="flag.value"
              class="inline-flex cursor-pointer items-center gap-1.5 rounded border border-gray-200 px-2 py-1 text-xs text-gray-700 dark:border-dark-700 dark:text-dark-200"
            >
              <input
                type="checkbox"
                class="h-3.5 w-3.5 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
                :checked="form.support_flags.includes(flag.value)"
                @change="toggleListValue('support_flags', flag.value)"
              />
              {{ flag.label }}
            </label>
          </div>
          <p class="mt-1 text-xs text-gray-400">{{ t('admin.modelCatalog.metadata.form.supportFlagsHint') }}</p>
        </div>

        <div>
          <label class="input-label">{{ t('admin.modelCatalog.metadata.fields.iconUrl') }}</label>
          <input
            v-model="form.icon_url"
            type="url"
            class="input"
            :placeholder="t('admin.modelCatalog.metadata.form.iconUrlPlaceholder')"
          />
        </div>

        <label class="flex cursor-pointer items-start gap-2">
          <input
            v-model="form.featured"
            type="checkbox"
            class="mt-0.5 h-4 w-4 rounded border-gray-300 text-primary-600 focus:ring-primary-500"
          />
          <span>
            <span class="block text-sm font-medium text-gray-700 dark:text-dark-200">
              {{ t('admin.modelCatalog.metadata.fields.featured') }}
            </span>
            <span class="block text-xs text-gray-400">{{ t('admin.modelCatalog.metadata.form.featuredHint') }}</span>
          </span>
        </label>

        <div class="flex justify-end gap-3 border-t border-gray-200 pt-4 dark:border-dark-700">
          <button type="button" class="btn btn-secondary" @click="closeDialog">
            {{ t('common.cancel') }}
          </button>
          <button type="submit" class="btn btn-primary" :disabled="saving">
            {{ saving ? t('common.saving') : t('common.save') }}
          </button>
        </div>
      </form>
    </BaseDialog>

    <ConfirmDialog
      :show="showClearDialog"
      :title="t('admin.modelCatalog.metadata.clear')"
      :message="clearConfirmMessage"
      :confirm-text="t('common.delete')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      @confirm="confirmClear"
      @cancel="showClearDialog = false"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { ModelCatalogMetadataListItem } from '@/api/admin/modelCatalog'
import type { Column } from '@/components/common/types'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { platformBadgeClass, platformLabel } from '@/utils/platformColors'
import AppLayout from '@/components/layout/AppLayout.vue'
import TablePageLayout from '@/components/layout/TablePageLayout.vue'
import DataTable from '@/components/common/DataTable.vue'
import EmptyState from '@/components/common/EmptyState.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import EndpointConfigButton from '@/components/admin/channel/EndpointConfigButton.vue'

const { t } = useI18n()
const appStore = useAppStore()

const items = ref<ModelCatalogMetadataListItem[]>([])
const loading = ref(false)
const saving = ref(false)
const searchQuery = ref('')
const uncoveredOnly = ref(false)
const showDialog = ref(false)
const showClearDialog = ref(false)
const clearing = ref<ModelCatalogMetadataListItem | null>(null)
const rememberedCategoryOptions = ref<string[]>([])
const categoryDropdownOpen = ref(false)
const categoryComboboxRef = ref<HTMLElement | null>(null)

interface FormState {
  model_name: string
  display_name: string
  description: string
  category: string
  model_type: string
  context_window: number
  max_output: number
  capabilities: string[]
  input_modalities: string[]
  output_modalities: string[]
  support_flags: string[]
  featured: boolean
  icon_url: string
}

const form = reactive<FormState>({
  model_name: '',
  display_name: '',
  description: '',
  category: '',
  model_type: '',
  context_window: 0,
  max_output: 0,
  capabilities: [],
  input_modalities: [],
  output_modalities: [],
  support_flags: [],
  featured: false,
  icon_url: '',
})

const columns = computed<Column[]>(() => [
  { key: 'model_name', label: t('admin.modelCatalog.metadata.fields.modelName'), sortable: true, class: 'w-80 min-w-80' },
  { key: 'platforms', label: t('admin.modelCatalog.metadata.fields.platforms'), sortable: false, class: 'w-24 min-w-24' },
  { key: 'category', label: t('admin.modelCatalog.metadata.fields.category'), sortable: false, class: 'w-24 min-w-24' },
  { key: 'context', label: t('admin.modelCatalog.metadata.fields.contextWindow'), sortable: false, class: 'w-24 min-w-24' },
  { key: 'missing', label: t('admin.modelCatalog.metadata.fields.missing'), sortable: false, class: 'w-[30rem] min-w-[30rem]' },
  { key: 'actions', label: t('admin.modelCatalog.metadata.fields.actions'), sortable: false, class: 'w-24 min-w-24' },
])

const catalogPlatformOptions = ['anthropic', 'openai', 'gemini', 'antigravity', 'grok']

const categoryOptions = computed(() =>
  sortedUniqueCaseInsensitive([
    ...rememberedCategoryOptions.value,
    ...items.value.map((item) => item.metadata.category),
    ...items.value.map((item) => item.override?.category || ''),
    ...catalogPlatformOptions,
    ...items.value.flatMap((item) => item.platforms || []),
  ]).map((value) => ({
    value,
    label: categoryOptionLabel(value),
  }))
)

const defaultModelTypes = [
  'chat',
  'responses',
  'completion',
  'embedding',
  'image_generation',
  'audio_speech',
  'audio_transcription',
]
const defaultModalities = ['text', 'image', 'audio', 'video']
const legacyModelCategories = ['claude', 'gpt', 'image', 'embedding', 'audio', 'other']
const defaultSupportFlags = [
  'assistant_prefill',
  'audio_input',
  'audio_output',
  'computer_use',
  'function_calling',
  'native_streaming',
  'parallel_function_calling',
  'pdf_input',
  'prompt_caching',
  'reasoning',
  'response_schema',
  'service_tier',
  'system_messages',
  'tool_choice',
  'url_context',
  'video_input',
  'vision',
  'web_search',
]

const modelTypeOptions = computed(() =>
  sortedUnique([
    ...defaultModelTypes,
    ...items.value.map((item) => item.metadata.model_type),
    ...items.value.map((item) => item.override?.model_type || ''),
  ])
)

const modalityOptions = computed(() =>
  sortedUnique([
    ...defaultModalities,
    ...items.value.flatMap((item) => item.metadata.input_modalities || []),
    ...items.value.flatMap((item) => item.metadata.output_modalities || []),
    ...items.value.flatMap((item) => item.override?.input_modalities || []),
    ...items.value.flatMap((item) => item.override?.output_modalities || []),
  ]).map((value) => ({
    value,
    label: t(`modelCatalog.modalities.${value}`, humanizeKey(value)),
  }))
)

const supportFlagOptions = computed(() =>
  sortedUnique([
    ...defaultSupportFlags,
    ...items.value.flatMap((item) => item.metadata.support_flags || []),
    ...items.value.flatMap((item) => item.override?.support_flags || []),
  ]).map((value) => ({
    value,
    label: t(`modelCatalog.capabilities.${value}`, humanizeKey(value)),
  }))
)

const filteredItems = computed(() => {
  const q = searchQuery.value.trim().toLowerCase()
  return items.value.filter((item) => {
    if (uncoveredOnly.value && item.override) return false
    if (!q) return true
    return modelCatalogMetadataSearchText(item).includes(q)
  })
})

function modelCatalogMetadataSearchText(item: ModelCatalogMetadataListItem): string {
  const metadata = item.metadata
  const override = item.override
  return [
    item.model_name,
    metadata?.display_name,
    metadata?.description,
    metadata?.category,
    metadata?.model_type,
    override?.display_name,
    override?.description,
    override?.category,
    override?.model_type,
    ...(item.platforms || []),
    ...(item.missing_fields || []),
    ...(metadata?.capabilities || []),
    ...(metadata?.input_modalities || []),
    ...(metadata?.output_modalities || []),
    ...(metadata?.support_flags || []),
    ...(override?.capabilities || []),
    ...(override?.input_modalities || []),
    ...(override?.output_modalities || []),
    ...(override?.support_flags || []),
  ]
    .filter(Boolean)
    .join(' ')
    .toLowerCase()
}

const clearConfirmMessage = computed(() =>
  t('admin.modelCatalog.metadata.clearConfirm', { name: clearing.value?.model_name || '' })
)

async function loadItems() {
  loading.value = true
  try {
    const res = await adminAPI.modelCatalog.listMetadata()
    items.value = res.items || []
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('admin.modelCatalog.metadata.loadError')))
  } finally {
    loading.value = false
  }
}

function openEditDialog(row: ModelCatalogMetadataListItem) {
  const src = row.metadata
  form.model_name = row.model_name
  form.display_name = src.display_name || ''
  form.description = src.description || ''
  form.category = src.category || ''
  form.model_type = src.model_type || ''
  form.context_window = src.context_window || 0
  form.max_output = src.max_output || 0
  form.capabilities = [...(src.capabilities || [])]
  form.input_modalities = [...(src.input_modalities || [])]
  form.output_modalities = [...(src.output_modalities || [])]
  form.support_flags = [...(src.support_flags || [])]
  form.featured = !!src.featured
  form.icon_url = src.icon_url || ''
  showDialog.value = true
}

function closeDialog() {
  showDialog.value = false
  categoryDropdownOpen.value = false
}

async function saveMetadata() {
  saving.value = true
  try {
    const saved = await adminAPI.modelCatalog.upsertMetadata({
      model_name: form.model_name,
      display_name: form.display_name,
      description: form.description,
      category: form.category,
      model_type: form.model_type,
      context_window: Number(form.context_window || 0),
      max_output: Number(form.max_output || 0),
      capabilities: form.capabilities,
      input_modalities: form.input_modalities,
      output_modalities: form.output_modalities,
      support_flags: form.support_flags,
      featured: !!form.featured,
      icon_url: form.icon_url,
    })
    rememberCategoryOption(saved.category || form.category)
    appStore.showSuccess(t('admin.modelCatalog.metadata.saveSuccess'))
    closeDialog()
    await loadItems()
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('admin.modelCatalog.metadata.saveError')))
  } finally {
    saving.value = false
  }
}

function openClearDialog(row: ModelCatalogMetadataListItem) {
  if (!row.override) return
  clearing.value = row
  showClearDialog.value = true
}

async function confirmClear() {
  if (!clearing.value) return
  try {
    await adminAPI.modelCatalog.removeMetadata(clearing.value.model_name)
    appStore.showSuccess(t('admin.modelCatalog.metadata.deleteSuccess'))
    showClearDialog.value = false
    clearing.value = null
    await loadItems()
  } catch (err: unknown) {
    appStore.showError(extractApiErrorMessage(err, t('admin.modelCatalog.metadata.deleteError')))
  }
}

function toggleListValue(field: 'input_modalities' | 'output_modalities' | 'support_flags', value: string) {
  const list = form[field] || []
  form[field] = list.includes(value) ? list.filter((item) => item !== value) : [...list, value]
}

function toggleCategoryDropdown() {
  categoryDropdownOpen.value = !categoryDropdownOpen.value
}

function selectCategoryOption(value: string) {
  form.category = value
  categoryDropdownOpen.value = false
}

function rememberCategoryOption(value: string) {
  const category = value.trim()
  if (!category) return
  rememberedCategoryOptions.value = sortedUniqueCaseInsensitive([...rememberedCategoryOptions.value, category])
}

function handleCategoryOutsideClick(event: PointerEvent) {
  const root = categoryComboboxRef.value
  if (!root || root.contains(event.target as Node)) return
  categoryDropdownOpen.value = false
}

function missingFieldLabel(field: string) {
  return t(`admin.modelCatalog.metadata.missingFields.${field}`, field)
}

function platformCategoryLabel(value: string): string {
  return t(`admin.groups.platforms.${value}`, platformLabel(value))
}

function categoryOptionLabel(value: string): string {
  return catalogPlatformOptions.includes(value)
    ? platformCategoryLabel(value)
    : t(`modelCatalog.categories.${value}`, value)
}

function formatCategoryLabel(category: string, rowPlatforms: string[]): string {
  const platforms = sortedUnique([...catalogPlatformOptions, ...(rowPlatforms || [])])
  const effective = platforms.includes(category)
    ? category
    : legacyModelCategories.includes(category)
      ? rowPlatforms?.[0] || category
      : category
  if (!effective) return '-'
  return platforms.includes(effective)
    ? platformCategoryLabel(effective)
    : t(`modelCatalog.categories.${effective}`, effective)
}

function formatTokens(n: number): string {
  if (!n || n <= 0) return '-'
  if (n >= 1_000_000) return `${(n / 1_000_000).toFixed(1)}M`
  if (n >= 1000) return `${Math.round(n / 1000)}K`
  return String(n)
}

function sortedUnique(values: string[]): string[] {
  return Array.from(new Set(values.map((v) => v.trim().toLowerCase()).filter(Boolean))).sort()
}

function sortedUniqueCaseInsensitive(values: string[]): string[] {
  const seen = new Set<string>()
  const out: string[] = []
  for (const raw of values) {
    const value = raw.trim()
    if (!value) continue
    const key = value.toLowerCase()
    if (seen.has(key)) continue
    seen.add(key)
    out.push(value)
  }
  return out.sort((a, b) => a.localeCompare(b, undefined, { sensitivity: 'base' }))
}

function humanizeKey(key: string): string {
  return key
    .split('_')
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ')
}

onMounted(() => {
  document.addEventListener('pointerdown', handleCategoryOutsideClick)
  loadItems()
})

onBeforeUnmount(() => {
  document.removeEventListener('pointerdown', handleCategoryOutsideClick)
})
</script>
