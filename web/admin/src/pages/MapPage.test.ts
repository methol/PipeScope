import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import MapPage from './MapPage.vue'

const geoJSON = {
  type: 'FeatureCollection',
  features: [{ properties: { province: '广东省' } }],
}

function stubFetch(options?: {
  geoOK?: boolean
  cityOK?: boolean
  provinceSummaryOK?: boolean
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
  const cityOK = options?.cityOK ?? true
  const provinceSummaryOK = options?.provinceSummaryOK ?? true
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
          ok: cityOK,
          status: cityOK ? 200 : 500,
          json: async () => ({ items: apiItems }),
        }
      }
      if (url.includes('/api/map/province-summary')) {
        return {
          ok: provinceSummaryOK,
          status: provinceSummaryOK ? 200 : 500,
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

  it('aggregates duplicated provinces after normalization', async () => {
    stubFetch({
      apiItems: [],
      provinceSummaryItems: [
        { province: '广东', value: 3 },
        { province: '广东省', value: 7 },
      ],
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

  it('keeps city heat data when province summary request fails', async () => {
    stubFetch({
      provinceSummaryOK: false,
      apiItems: [
        {
          adcode: '440300',
          province: '广东省',
          city: '深圳',
          lat: 22.5431,
          lng: 114.0579,
          value: 5,
        },
      ],
    })
    const wrapper = mount(MapPage)
    await flushPage()

    expect(wrapper.text()).toContain('省级汇总加载失败（已降级展示城市热力）')
    expect(wrapper.text()).toContain('广东省深圳')

    wrapper.unmount()
  })
})
