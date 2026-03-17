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

  it('does not auto-refresh analytics and requests backend aggregation once when searching', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: string | URL | Request) => {
        const url = String(input)
        if (url.includes('/api/analytics/options?')) {
          return {
            ok: true,
            status: 200,
            json: async () => ({
              rules: ['r1', 'r2'],
              provinces: ['广东'],
              cities: [
                { province: '广东', city: '深圳' },
                { province: '广东', city: '珠海' },
              ],
              statuses: ['ok', 'err'],
            }),
          }
        }
        if (url.includes('/api/analytics?')) {
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
        }
        throw new Error('unexpected')
      }),
    )

    const wrapper = mount(AnalyticsPage)
    await flushPage()

    const beforeSearchCalls = (fetch as ReturnType<typeof vi.fn>).mock.calls
      .map((x) => String(x[0]))
      .filter((url) => url.includes('/api/analytics?'))
    expect(beforeSearchCalls.length).toBe(0)

    await wrapper.find('button.btn').trigger('click')
    await flushPage()

    const calls = (fetch as ReturnType<typeof vi.fn>).mock.calls.map((x) => String(x[0]))
    expect(calls.filter((url) => url.includes('/api/analytics?')).length).toBe(1)
    expect(wrapper.text()).toContain('连接数：501')
    expect(wrapper.text()).toContain('活跃规则：2')
    expect(wrapper.text()).toContain('广东深圳 - 1.46 KB')
  })

  it('passes src_ip to analytics queries', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: string | URL | Request) => {
        const url = String(input)
        if (url.includes('/api/analytics/options?')) {
          return {
            ok: true,
            status: 200,
            json: async () => ({
              rules: ['r1'],
              provinces: ['广东'],
              cities: [{ province: '广东', city: '深圳' }],
              statuses: ['ok'],
            }),
          }
        }
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
    await flushPage()

    await wrapper.find('input').setValue('10.0.0.8')
    await wrapper.find('button.btn').trigger('click')
    await flushPage()

    const calls = (fetch as ReturnType<typeof vi.fn>).mock.calls.map((x) => String(x[0]))
    expect(calls.some((url) => url.includes('/api/analytics?') && url.includes('src_ip=10.0.0.8'))).toBe(true)
  })

  it('filters city options by selected province', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: string | URL | Request) => {
        const url = String(input)
        if (url.includes('/api/analytics/options?')) {
          return {
            ok: true,
            status: 200,
            json: async () => ({
              rules: ['r1'],
              provinces: ['广东', '浙江'],
              cities: [
                { province: '广东', city: '深圳' },
                { province: '广东', city: '珠海' },
                { province: '浙江', city: '杭州' },
              ],
              statuses: ['ok'],
            }),
          }
        }
        return {
          ok: true,
          status: 200,
          json: async () => ({
            overview: {
              conn_count: 0,
              total_bytes: 0,
              avg_duration_ms: 0,
              active_rules: 0,
              active_cities: 0,
            },
            top_cities: [],
            top_rules: [],
          }),
        }
      }),
    )

    const wrapper = mount(AnalyticsPage)
    await flushPage()

    const selects = wrapper.findAll('select')
    const provinceSelect = selects[2]
    const citySelect = selects[3]

    expect(citySelect.findAll('option').map((o) => o.text())).toEqual(['全部', '深圳', '珠海', '杭州'])

    await provinceSelect.setValue('浙江')
    await flushPage()

    expect(citySelect.findAll('option').map((o) => o.text())).toEqual(['全部', '杭州'])
  })

  it('clears previous results when a new search fails', async () => {
    let fail = false
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: string | URL | Request) => {
        const url = String(input)
        if (url.includes('/api/analytics/options?')) {
          return {
            ok: true,
            status: 200,
            json: async () => ({
              rules: ['r1'],
              provinces: ['广东'],
              cities: [{ province: '广东', city: '深圳' }],
              statuses: ['ok'],
            }),
          }
        }
        if (fail) throw new Error('network down')
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
    await flushPage()

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
