<!-- sudoapi: Model catalog. -->

<template>
  <button type="button" class="btn btn-secondary" @click="openDialog">
    <Icon name="cog" size="md" class="mr-2" />
    {{ t('admin.modelCatalog.endpointConfig.title') }}
  </button>

  <BaseDialog
    :show="showDialog"
    :title="t('admin.modelCatalog.endpointConfig.title')"
    width="extra-wide"
    @close="closeDialog"
  >
    <div class="endpoint-config-dialog-body">
      <div
        class="-mx-4 -mt-3 flex flex-shrink-0 items-center border-b border-gray-200 px-4 dark:border-dark-700 sm:-mx-6 sm:-mt-4 sm:px-6"
      >
        <button
          v-for="platform in endpointConfigPlatforms"
          :key="platform"
          type="button"
          class="endpoint-config-tab group"
          :class="
            endpointConfigActivePlatform === platform ? 'endpoint-config-tab-active' : 'endpoint-config-tab-inactive'
          "
          @click="endpointConfigActivePlatform = platform"
        >
          <PlatformIcon :platform="platform as GroupPlatform" size="xs" :class="platformTextClass(platform)" />
          <span :class="platformTextClass(platform)">{{ t('admin.groups.platforms.' + platform, platform) }}</span>
        </button>
      </div>

      <div class="flex-1 overflow-y-auto pt-4">
        <div v-if="endpointConfigLoading" class="py-10 text-center text-sm text-gray-500 dark:text-dark-300">
          {{ t('common.loading') }}
        </div>
        <div v-else-if="endpointConfigActivePlatform" class="space-y-4">
          <div class="flex items-center justify-between gap-3">
            <div>
              <p class="text-sm font-medium text-gray-900 dark:text-white">
                {{ t('admin.modelCatalog.endpointConfig.platformRules') }}
              </p>
              <p class="mt-0.5 text-xs text-gray-500 dark:text-dark-400">
                {{ t('admin.modelCatalog.endpointConfig.description') }}
              </p>
            </div>
            <button
              type="button"
              class="btn btn-secondary btn-sm"
              @click="addEndpointRule(endpointConfigActivePlatform)"
            >
              + {{ t('admin.modelCatalog.endpointConfig.addModelType') }}
            </button>
          </div>

          <datalist id="channel-endpoint-model-type-options">
            <option v-for="type in endpointModelTypeOptions" :key="type" :value="type" />
          </datalist>

          <div
            v-if="(endpointConfigForm[endpointConfigActivePlatform] || []).length === 0"
            class="rounded border border-dashed border-gray-300 p-5 text-center text-sm text-gray-400 dark:border-dark-500"
          >
            {{ t('admin.modelCatalog.endpointConfig.noRules') }}
          </div>

          <div v-else class="space-y-3">
            <div
              v-for="(rule, ruleIndex) in endpointConfigForm[endpointConfigActivePlatform]"
              :key="`${endpointConfigActivePlatform}-${ruleIndex}`"
              class="rounded-lg border border-gray-200 p-3 dark:border-dark-700"
            >
              <div class="mb-3 flex items-center gap-2">
                <input
                  v-model="rule.model_type"
                  type="text"
                  list="channel-endpoint-model-type-options"
                  class="input flex-1 text-sm"
                  :placeholder="t('admin.modelCatalog.endpointConfig.modelTypePlaceholder')"
                />
                <button
                  type="button"
                  class="rounded p-1 text-gray-400 hover:text-red-500"
                  @click="removeEndpointRule(endpointConfigActivePlatform, ruleIndex)"
                >
                  <Icon name="trash" size="sm" />
                </button>
              </div>

              <div class="space-y-2">
                <div
                  v-for="(endpoint, endpointIndex) in rule.endpoints"
                  :key="`${ruleIndex}-${endpointIndex}`"
                  class="flex items-center gap-2"
                >
                  <select v-model="endpoint.method" class="input w-28 text-sm">
                    <option value="POST">POST</option>
                    <option value="GET">GET</option>
                  </select>
                  <input
                    v-model="endpoint.path"
                    type="text"
                    class="input flex-1 font-mono text-sm"
                    placeholder="/v1/messages"
                  />
                  <button
                    type="button"
                    class="rounded p-1 text-gray-400 hover:text-red-500"
                    @click="removeEndpoint(endpointConfigActivePlatform, ruleIndex, endpointIndex)"
                  >
                    <Icon name="x" size="sm" />
                  </button>
                </div>
              </div>

              <button
                type="button"
                class="mt-2 text-xs text-primary-600 hover:text-primary-700"
                @click="addEndpoint(endpointConfigActivePlatform, ruleIndex)"
              >
                + {{ t('admin.modelCatalog.endpointConfig.addEndpoint') }}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="flex justify-end gap-3">
        <button type="button" class="btn btn-secondary" @click="closeDialog">
          {{ t('common.cancel') }}
        </button>
        <button
          type="button"
          class="btn btn-primary"
          :disabled="endpointConfigSaving || endpointConfigLoading"
          @click="saveEndpointConfig"
        >
          {{ endpointConfigSaving ? t('common.saving') : t('common.save') }}
        </button>
      </div>
    </template>
  </BaseDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type { ModelCatalogEndpoint, ModelCatalogEndpointConfig } from '@/api/admin/modelCatalog'
import type { GroupPlatform } from '@/types'
import { useAppStore } from '@/stores/app'
import { extractApiErrorMessage } from '@/utils/apiError'
import { platformTextClass } from '@/utils/platformColors'
import BaseDialog from '@/components/common/BaseDialog.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import Icon from '@/components/icons/Icon.vue'

interface EndpointConfigRule {
  model_type: string
  endpoints: ModelCatalogEndpoint[]
}

const { t } = useI18n()
const appStore = useAppStore()

const platformOrder = ['anthropic', 'openai', 'gemini', 'antigravity', 'grok']
const endpointModelTypeOptions = [
  'chat',
  'responses',
  'completion',
  'embedding',
  'image_generation',
  'audio_speech',
  'audio_transcription',
]
const endpointConfigKeyPattern = /^[a-z0-9_-]+$/
const showDialog = ref(false)
const endpointConfigLoading = ref(false)
const endpointConfigSaving = ref(false)
const endpointConfigActivePlatform = ref('')
const endpointConfigForm = reactive<Record<string, EndpointConfigRule[]>>({})

const endpointConfigPlatforms = computed(() => {
  return sortedUniquePlatforms([...platformOrder, ...Object.keys(endpointConfigForm)])
})

function sortedUniquePlatforms(values: string[]): string[] {
  return Array.from(new Set(values.map((v) => v.trim().toLowerCase()).filter(Boolean))).sort()
}

function clearEndpointConfigForm() {
  for (const key of Object.keys(endpointConfigForm)) {
    delete endpointConfigForm[key]
  }
}

function ensureEndpointConfigPlatform(platform: string) {
  const key = platform.trim().toLowerCase()
  if (!key) return
  if (!endpointConfigForm[key]) {
    endpointConfigForm[key] = []
  }
}

function endpointConfigToForm(config: ModelCatalogEndpointConfig) {
  clearEndpointConfigForm()
  const platforms = config.platforms || {}
  for (const platform of sortedUniquePlatforms(Object.keys(platforms))) {
    const rules = platforms[platform] || {}
    endpointConfigForm[platform] = Object.keys(rules)
      .sort()
      .map((modelType) => ({
        model_type: modelType,
        endpoints: (rules[modelType] || []).map((ep) => ({
          method: ep.method,
          path: ep.path,
        })),
      }))
  }
  for (const platform of platformOrder) {
    ensureEndpointConfigPlatform(platform)
  }
}

function addEndpointRule(platform: string) {
  ensureEndpointConfigPlatform(platform)
  const used = new Set((endpointConfigForm[platform] || []).map((rule) => rule.model_type.trim().toLowerCase()))
  const candidate = endpointModelTypeOptions.find((type) => !used.has(type)) || ''
  endpointConfigForm[platform].push({
    model_type: candidate,
    endpoints: [{ method: 'POST', path: '' }],
  })
}

function removeEndpointRule(platform: string, ruleIndex: number) {
  endpointConfigForm[platform]?.splice(ruleIndex, 1)
}

function addEndpoint(platform: string, ruleIndex: number) {
  endpointConfigForm[platform]?.[ruleIndex]?.endpoints.push({ method: 'POST', path: '' })
}

function removeEndpoint(platform: string, ruleIndex: number, endpointIndex: number) {
  endpointConfigForm[platform]?.[ruleIndex]?.endpoints.splice(endpointIndex, 1)
}

function endpointConfigFormToAPI(): ModelCatalogEndpointConfig | null {
  const config: ModelCatalogEndpointConfig = { platforms: {} }
  for (const platform of endpointConfigPlatforms.value) {
    const platformKey = platform.trim().toLowerCase()
    const rules = endpointConfigForm[platform] || []
    if (rules.length === 0) continue
    if (!endpointConfigKeyPattern.test(platformKey)) {
      appStore.showError(t('admin.modelCatalog.endpointConfig.invalidKey'))
      return null
    }
    for (const rule of rules) {
      const modelType = rule.model_type.trim().toLowerCase()
      if (!modelType || !endpointConfigKeyPattern.test(modelType)) {
        appStore.showError(t('admin.modelCatalog.endpointConfig.invalidKey'))
        return null
      }
      const endpoints: ModelCatalogEndpoint[] = []
      for (const endpoint of rule.endpoints) {
        const method = endpoint.method.trim().toUpperCase()
        const path = endpoint.path.trim()
        if (method !== 'GET' && method !== 'POST') {
          appStore.showError(t('admin.modelCatalog.endpointConfig.invalidMethod'))
          return null
        }
        if (!path || !path.startsWith('/') || /\s/.test(path)) {
          appStore.showError(t('admin.modelCatalog.endpointConfig.invalidPath'))
          return null
        }
        endpoints.push({ method, path })
      }
      if (endpoints.length === 0) continue
      if (!config.platforms[platformKey]) {
        config.platforms[platformKey] = {}
      }
      if (!config.platforms[platformKey][modelType]) {
        config.platforms[platformKey][modelType] = []
      }
      config.platforms[platformKey][modelType].push(...endpoints)
    }
  }
  return config
}

async function openDialog() {
  showDialog.value = true
  endpointConfigLoading.value = true
  try {
    const config = await adminAPI.modelCatalog.getEndpointConfig()
    endpointConfigToForm(config)
    endpointConfigActivePlatform.value = endpointConfigPlatforms.value[0] || ''
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('admin.modelCatalog.endpointConfig.loadError')))
  } finally {
    endpointConfigLoading.value = false
  }
}

function closeDialog() {
  showDialog.value = false
  endpointConfigActivePlatform.value = ''
  clearEndpointConfigForm()
}

async function saveEndpointConfig() {
  if (endpointConfigSaving.value) return
  const config = endpointConfigFormToAPI()
  if (!config) return

  endpointConfigSaving.value = true
  try {
    const saved = await adminAPI.modelCatalog.updateEndpointConfig(config)
    endpointConfigToForm(saved)
    endpointConfigActivePlatform.value = endpointConfigPlatforms.value[0] || ''
    appStore.showSuccess(t('admin.modelCatalog.endpointConfig.saveSuccess'))
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('admin.modelCatalog.endpointConfig.saveError')))
  } finally {
    endpointConfigSaving.value = false
  }
}
</script>

<style scoped>
.endpoint-config-dialog-body {
  display: flex;
  flex-direction: column;
  height: 70vh;
  min-height: 400px;
}

.endpoint-config-tab {
  @apply flex items-center gap-1.5 px-3 py-2.5 text-sm font-medium border-b-2 transition-colors whitespace-nowrap;
}

.endpoint-config-tab-active {
  @apply border-primary-600 text-primary-600 dark:border-primary-400 dark:text-primary-400;
}

.endpoint-config-tab-inactive {
  @apply border-transparent text-gray-500 hover:text-gray-700 hover:border-gray-300 dark:text-gray-400 dark:hover:text-gray-300;
}
</style>
