import { mount } from '@vue/test-utils'
import { describe, expect, it, vi, beforeEach, afterEach } from 'vitest'
import MapPage from './MapPage.vue'

describe('MapPage', () => {
  const mockResponse = {
    items: [
      {
        adcode: '440300',
        province: '广东',
        city: '深圳',
        lat: 22.5431,
        lng: 114.0579,
        value: 5,
      },
    ],
  }

  beforeEach(() => {
    vi.stubGlobal(
      'fetch',
      vi.fn(async () => ({
        ok: true,
        json: async () => mockResponse,
      })),
    )
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('renders map page and calls china map api', async () => {
    const wrapper = mount(MapPage)
    await new Promise((resolve) => setTimeout(resolve, 0))

    expect(fetch).toHaveBeenCalled()
    const call = (fetch as ReturnType<typeof vi.fn>).mock.calls[0]?.[0]
    expect(String(call)).toContain('/api/map/china')
    expect(String(call)).toContain('window=15m')

    wrapper.unmount()
  })
})
