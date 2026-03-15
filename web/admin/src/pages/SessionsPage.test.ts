import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import SessionsPage from './SessionsPage.vue'

describe('SessionsPage', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: string | URL | Request) => {
        const url = String(input)
        if (!url.includes('/api/sessions?')) throw new Error('unexpected')
        return {
          ok: true,
          status: 200,
          json: async () => ({ items: [] }),
        }
      }),
    )
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.unstubAllGlobals()
  })

  it('uses fixed 5m window without selector and keeps auto-refresh', async () => {
    const wrapper = mount(SessionsPage)
    await Promise.resolve()

    expect(wrapper.text()).toContain('固定窗口：5m')
    expect(wrapper.find('select').exists()).toBe(false)

    const calls = (fetch as ReturnType<typeof vi.fn>).mock.calls.map((x) => String(x[0]))
    expect(calls.some((x) => x.includes('window=5m'))).toBe(true)

    await vi.advanceTimersByTimeAsync(5000)
    await Promise.resolve()
    await Promise.resolve()
    expect((fetch as ReturnType<typeof vi.fn>).mock.calls.length).toBeGreaterThan(1)
  })
})
