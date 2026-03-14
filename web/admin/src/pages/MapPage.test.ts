import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import MapPage from './MapPage.vue'

const geoJSON = {
  type: 'FeatureCollection',
  features: [{ properties: { province: '广东省' } }],
}

function stubFetch(options?: {
  geoOK?: boolean
  apiItems?: Array<{
    adcode: string
    province: string
    city: string
    lat: number
    lng: number
    value: number
  }>
  provinceSummaryItems?: Array<{
    province: string
    value: number
  }>
}) {
  const geoOK = options?.geoOK ?? true
  const apiItems =
    options?.apiItems ?? [
      {
        adcode: '440300',
        province: '广东省',
        city: '深圳',
        lat: 22.5431,
        lng: 114.0579,
        value: 5,
      },
    ]
  const provinceSummaryItems = options?.provinceSummaryItems ?? [{ province: '广东省', value: 5 }]

  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: string | URL | Request) => {
      const url = String(input)
      if (url.includes('/maps/china-counties.geojson')) {
        return {
          ok: geoOK,
          status: geoOK ? 200 : 500,
          json: async () => geoJSON,
        }
      }
      if (url.includes('/api/map/china')) {
        return {
          ok: true,
          status: 200,
          json: async () => ({ items: apiItems }),
        }
      }
      if (url.includes('/api/map/province-summary')) {
        return {
          ok: true,
          status: 200,
          json: async () => ({ items: provinceSummaryItems }),
        }
      }
      throw new Error(`unexpected fetch url: ${url}`)
    }),
  )
}

async function flushPage() {
  await new Promise((resolve) => setTimeout(resolve, 0))
  await new Promise((resolve) => setTimeout(resolve, 0))
}

describe('MapPage', () => {
  beforeEach(() => {
    stubFetch()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('loads geojson, city data and province summary api', async () => {
    const wrapper = mount(MapPage)
    await flushPage()

    expect(fetch).toHaveBeenCalled()
    const calls = (fetch as ReturnType<typeof vi.fn>).mock.calls.map((call) => String(call[0]))
    expect(calls.some((call) => call.includes('/maps/china-counties.geojson'))).toBe(true)
    expect(calls.some((call) => call.includes('/api/map/china'))).toBe(true)
    expect(calls.some((call) => call.includes('/api/map/province-summary'))).toBe(true)
    expect(calls.some((call) => call.includes('metric=conn'))).toBe(true)

    wrapper.unmount()
  })


  it('normalizes province names to map feature namespace and reports coverage', async () => {
    stubFetch({
      apiItems: [],
      provinceSummaryItems: [{ province: '广东', value: 3 }],
    })
    const wrapper = mount(MapPage)
    await flushPage()

    expect(wrapper.text()).toContain('省份命中率: 1/1')

    wrapper.unmount()
  })

  it('switches metric to bytes and renders human-readable units', async () => {
    stubFetch({
      apiItems: [
        {
          adcode: '440300',
          province: '广东省',
          city: '深圳',
          lat: 22.5431,
          lng: 114.0579,
          value: 2048,
        },
      ],
      provinceSummaryItems: [{ province: '广东省', value: 2048 }],
    })

    const wrapper = mount(MapPage)
    await flushPage()

    await wrapper.findAll('select')[1].setValue('bytes')
    await flushPage()

    const calls = (fetch as ReturnType<typeof vi.fn>).mock.calls.map((call) => String(call[0]))
    expect(calls.some((call) => call.includes('metric=bytes'))).toBe(true)
    expect(wrapper.text()).toContain('2.00 KB')

    wrapper.unmount()
  })

  it('shows a visible error when geojson loading fails', async () => {
    stubFetch({ geoOK: false })
    const wrapper = mount(MapPage)
    await flushPage()

    expect(wrapper.text()).toContain('底图加载失败')

    wrapper.unmount()
  })

  it('handles empty api data without crashing', async () => {
    stubFetch({ apiItems: [], provinceSummaryItems: [] })
    const wrapper = mount(MapPage)
    await flushPage()

    expect(wrapper.text()).toContain('当前窗口暂无城市指标数据')
    expect(wrapper.find('.chart').exists()).toBe(true)

    wrapper.unmount()
  })
})
