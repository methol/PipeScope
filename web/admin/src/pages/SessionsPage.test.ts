import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import SessionsPage from './SessionsPage.vue'

async function flushPage() {
  await Promise.resolve()
  await Promise.resolve()
  await Promise.resolve()
  await Promise.resolve()
}

describe('SessionsPage', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: string | URL | Request) => {
        const url = String(input)
        if (url.includes('/api/sessions/options?')) {
          return {
            ok: true,
            status: 200,
            json: async () => ({ rules: ['r1', 'r2'] }),
          }
        }
        if (url.includes('/api/sessions?')) {
          return {
            ok: true,
            status: 200,
            json: async () => ({ items: [] }),
          }
        }
        throw new Error('unexpected')
      }),
    )
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('uses fixed 5m window, keeps auto-refresh, and exposes bounded rule/limit selectors', async () => {
    const wrapper = mount(SessionsPage)
    await vi.advanceTimersByTimeAsync(0)
    await flushPage()

    expect(wrapper.text()).toContain('固定窗口：5m')
    const selects = wrapper.findAll('select')
    expect(selects).toHaveLength(2)
    expect(selects[1].findAll('option').map((o) => o.text())).toEqual(['100', '1000', '10000'])

    const calls = (fetch as ReturnType<typeof vi.fn>).mock.calls.map((x) => String(x[0]))
    expect(calls.some((x) => x.includes('/api/sessions/options?window=15m'))).toBe(true)
    expect(calls.some((x) => x.includes('/api/sessions?window=5m'))).toBe(true)
    expect(calls.some((x) => x.includes('limit=100'))).toBe(true)

    await vi.advanceTimersByTimeAsync(5000)
    await flushPage()
    expect((fetch as ReturnType<typeof vi.fn>).mock.calls.length).toBeGreaterThan(2)
  })

  it('requests the bounded 10000 limit when the largest option is selected', async () => {
    const wrapper = mount(SessionsPage)
    await vi.advanceTimersByTimeAsync(0)
    await flushPage()

    await wrapper.findAll('select')[1].setValue('10000')
    await flushPage()

    const calls = (fetch as ReturnType<typeof vi.fn>).mock.calls.map((x) => String(x[0]))
    expect(calls.some((x) => x.includes('/api/sessions?') && x.includes('limit=10000'))).toBe(true)
  })

})
