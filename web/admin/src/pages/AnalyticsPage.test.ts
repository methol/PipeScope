import { mount } from '@vue/test-utils'
import { afterEach, describe, expect, it, vi } from 'vitest'
import AnalyticsPage from './AnalyticsPage.vue'

async function flushPage() {
  await new Promise((resolve) => setTimeout(resolve, 0))
  await new Promise((resolve) => setTimeout(resolve, 0))
}

describe('AnalyticsPage', () => {
  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('does not auto-refresh and requests backend aggregation once when searching', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: string | URL | Request) => {
        const url = String(input)
        if (!url.includes('/api/analytics?')) throw new Error('unexpected')
        return {
          ok: true,
          status: 200,
          json: async () => ({
            overview: {
              conn_count: 501,
              total_bytes: 1503,
              avg_duration_ms: 12,
              active_rules: 2,
              active_cities: 2,
            },
            top_cities: [
              { name: '广东深圳', conn_count: 500, total_bytes: 1500 },
              { name: '广东珠海', conn_count: 1, total_bytes: 3 },
            ],
            top_rules: [
              { name: 'r1', conn_count: 500, total_bytes: 1500 },
              { name: 'r2', conn_count: 1, total_bytes: 3 },
            ],
          }),
        }
      }),
    )

    const wrapper = mount(AnalyticsPage)
    expect((fetch as ReturnType<typeof vi.fn>).mock.calls.length).toBe(0)

    await wrapper.find('button.btn').trigger('click')
    await flushPage()

    const calls = (fetch as ReturnType<typeof vi.fn>).mock.calls.map((x) => String(x[0]))
    expect(calls.length).toBe(1)
    expect(calls[0]).toContain('/api/analytics?')
    expect(wrapper.text()).toContain('连接数：501')
    expect(wrapper.text()).toContain('活跃规则：2')
    expect(wrapper.text()).toContain('广东深圳 - 1.46 KB')
  })

  it('clears previous results when a new search fails', async () => {
    let fail = false
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: string | URL | Request) => {
        if (fail) throw new Error('network down')
        const url = String(input)
        if (!url.includes('/api/analytics?')) throw new Error('unexpected')
        return {
          ok: true,
          status: 200,
          json: async () => ({
            overview: {
              conn_count: 1,
              total_bytes: 3,
              avg_duration_ms: 10,
              active_rules: 1,
              active_cities: 1,
            },
            top_cities: [{ name: '广东深圳', conn_count: 1, total_bytes: 3 }],
            top_rules: [{ name: 'r1', conn_count: 1, total_bytes: 3 }],
          }),
        }
      }),
    )

    const wrapper = mount(AnalyticsPage)

    await wrapper.find('button.btn').trigger('click')
    await flushPage()
    expect(wrapper.text()).toContain('连接数：1')

    fail = true
    await wrapper.find('button.btn').trigger('click')
    await flushPage()

    expect(wrapper.text()).toContain('network down')
    expect(wrapper.text()).toContain('连接数：0')
  })
})
