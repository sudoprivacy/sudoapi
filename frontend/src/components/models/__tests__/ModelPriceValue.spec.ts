// sudoapi: Model Square model catalog.

import { describe, expect, it } from 'vitest'
import { mount } from '@vue/test-utils'
import ModelPriceValue from '../ModelPriceValue.vue'

describe('ModelPriceValue', () => {
  it('shows discounted price with the original base price struck through', () => {
    const wrapper = mount(ModelPriceValue, {
      props: {
        value: 10,
        mult: 0.8,
        unit: 'MTok',
      },
    })

    expect(wrapper.text()).toContain('$8.00')
    expect(wrapper.text()).toContain('$10.00')
    expect(wrapper.text()).toContain('/ MTok')
    expect(wrapper.find('.line-through').exists()).toBe(true)
    expect(wrapper.find('.line-through').text()).toBe('$10.00')
  })

  it('hides original price when there is no discount', () => {
    const wrapper = mount(ModelPriceValue, {
      props: {
        value: 10,
        mult: 1,
        unit: 'MTok',
      },
    })

    expect(wrapper.text()).toContain('$10.00')
    expect(wrapper.find('.line-through').exists()).toBe(false)
  })
})
