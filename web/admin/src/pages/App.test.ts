import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import App from '../App.vue'

vi.mock('./MapPage.vue', () => ({ default: { template: '<div data-testid="map-page">map</div>' } }))
vi.mock('./RulesPage.vue', () => ({ default: { template: '<div data-testid="rules-page">rules</div>' } }))
vi.mock('./SessionsPage.vue', () => ({ default: { template: '<div data-testid="realtime-page">realtime</div>' } }))
vi.mock('./AnalyticsPage.vue', () => ({ default: { template: '<div data-testid="analytics-page">analytics</div>' } }))

describe('App tabs', () => {
  it('renders realtime and analytics tabs and switches pages', async () => {
    const wrapper = mount(App)
    expect(wrapper.find('[data-testid="map-page"]').exists()).toBe(true)

    const buttons = wrapper.findAll('button')
    await buttons[2].trigger('click')
    expect(wrapper.find('[data-testid="realtime-page"]').exists()).toBe(true)

    await buttons[3].trigger('click')
    expect(wrapper.find('[data-testid="analytics-page"]').exists()).toBe(true)
  })
})
