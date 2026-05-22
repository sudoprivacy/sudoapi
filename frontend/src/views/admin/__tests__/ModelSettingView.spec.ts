import { flushPromises, mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ModelSettingView from '../ModelSettingView.vue'

const { getStatus, upload } = vi.hoisted(() => ({
  getStatus: vi.fn(),
  upload: vi.fn(),
}))

vi.mock('@/api/admin/modelSetting', () => ({
  default: { getStatus, upload },
}))

vi.mock('@/components/icons/Icon.vue', () => ({
  default: { template: '<span />' },
}))

vi.mock('@/utils/apiError', () => ({
  extractApiErrorMessage: (error: unknown, fallback: string) =>
    error instanceof Error ? error.message : fallback,
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({
    t: (key: string, params?: Record<string, unknown>) => {
      if (key === 'admin.modelSetting.uploadSuccess') return `loaded ${params?.count}`
      return key
    },
  }),
}))

const status = {
  file_path: '/tmp/models.csv',
  file_name: 'models.csv',
  source: 'uploaded',
  model_count: 2,
  updated_at: '2026-05-22T00:00:00Z',
  summary: {
    total_rows: 2,
    loaded_rows: 2,
    duplicate_rows: 0,
    skipped_rows: 0,
    header_row_count: 1,
  },
}

describe('ModelSettingView', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getStatus.mockResolvedValue(status)
    upload.mockResolvedValue({ ...status, model_count: 3, summary: { ...status.summary, loaded_rows: 3 } })
  })

  it('loads and displays current status', async () => {
    const wrapper = mount(ModelSettingView)
    await flushPromises()
    expect(getStatus).toHaveBeenCalled()
    expect(wrapper.text()).toContain('/tmp/models.csv')
    expect(wrapper.text()).toContain('uploaded')
  })

  it('uploads selected csv and displays success', async () => {
    const wrapper = mount(ModelSettingView)
    await flushPromises()

    const file = new File(['serial_number,id\n1,gpt-a\n'], 'models.csv', { type: 'text/csv' })
    const input = wrapper.find('input[type="file"]')
    Object.defineProperty(input.element, 'files', {
      value: [file],
      configurable: true,
    })
    await input.trigger('change')
    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()

    expect(upload).toHaveBeenCalledWith(file)
    expect(wrapper.text()).toContain('loaded 3')
  })

  it('shows upload error without replacing success message', async () => {
    upload.mockRejectedValueOnce(new Error('bad csv'))
    const wrapper = mount(ModelSettingView)
    await flushPromises()

    const file = new File(['bad'], 'models.csv', { type: 'text/csv' })
    const input = wrapper.find('input[type="file"]')
    Object.defineProperty(input.element, 'files', {
      value: [file],
      configurable: true,
    })
    await input.trigger('change')
    await wrapper.find('button.btn-primary').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('bad csv')
  })
})
