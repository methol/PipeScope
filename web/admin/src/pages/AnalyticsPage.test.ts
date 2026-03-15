import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import AnalyticsPage from './AnalyticsPage.vue'

async function flushPage() {
  await new Promise((resolve) => setTimeout(resolve, 0))
  await new Promise((resolve) => setTimeout(resolve, 0))
}

describe('AnalyticsPage', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('does not auto-refresh and paginates when searching', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: string | URL | Request) => {
        const url = String(input)
        if (!url.includes('/api/sessions?')) throw new Error('unexpected')
        const isFirstPage = url.includes('offset=0')
        const isSecondPage = url.includes('offset=500')
        return {
          ok: true,
          status: 200,
          json: async () => ({
            items: isFirstPage
              ? Array.from({ length: 500 }, (_, i) => ({
                  id: i + 1,
                  rule_id: 'r1',
                  src_addr: 'a',
                  dst_addr: 'b',
                  status: 'ok',
                  up_bytes: 1,
                  down_bytes: 2,
                  total_bytes: 3,
                  start_ts: Date.now(),
                  end_ts: Date.now(),
                  duration_ms: 10,
                  province: '广东',
                  city: '深圳',
                  adcode: '440300',
                }))
              : isSecondPage
              ? [
                  {
                    id: 2,
                    rule_id: 'r2',
                    src_addr: 'c',
                    dst_addr: 'd',
                    status: 'ok',
                    up_bytes: 1,
                    down_bytes: 2,
                    total_bytes: 3,
                    start_ts: Date.now(),
                    end_ts: Date.now(),
                    duration_ms: 11,
                    province: '广东',
                    city: '珠海',
                    adcode: '440400',
                  },
                ]
              : [],
          }),
        }
      }),
    )

    const wrapper = mount(AnalyticsPage)
    expect((fetch as ReturnType<typeof vi.fn>).mock.calls.length).toBe(0)

    await wrapper.find('button.btn').trigger('click')
    await Promise.resolve()
    await Promise.resolve()

    const calls = (fetch as ReturnType<typeof vi.fn>).mock.calls.map((x) => String(x[0]))
    expect(calls.some((x) => x.includes('offset=0'))).toBe(true)
    expect(calls.some((x) => x.includes('offset=500'))).toBe(true)

    await vi.advanceTimersByTimeAsync(30000)
    await Promise.resolve()
    expect((fetch as ReturnType<typeof vi.fn>).mock.calls.length).toBe(2)
  })

  it('clears previous results when a new search fails', async () => {
    vi.useRealTimers()
    let fail = false
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: string | URL | Request) => {
        if (fail) throw new Error('network down')
        const url = String(input)
        const isFirstPage = url.includes('offset=0')
        return {
          ok: true,
          status: 200,
          json: async () => ({
            items: isFirstPage
              ? [
                  {
                    id: 1,
                    rule_id: 'r1',
                    src_addr: 'a',
                    dst_addr: 'b',
                    status: 'ok',
                    up_bytes: 1,
                    down_bytes: 2,
                    total_bytes: 3,
                    start_ts: Date.now(),
                    end_ts: Date.now(),
                    duration_ms: 10,
                    province: '广东',
                    city: '深圳',
                    adcode: '440300',
                  },
                ]
              : [],
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
