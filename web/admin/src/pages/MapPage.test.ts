import { mount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

let lastChartOption: any = null

vi.mock('echarts', () => ({
  init: vi.fn(() => ({
    setOption: (opt: any) => {
      lastChartOption = opt
    },
    resize: vi.fn(),
    dispose: vi.fn(),
  })),
  registerMap: vi.fn(),
}))

import MapPage from './MapPage.vue'

const geoJSON = {
  type: 'FeatureCollection',
  features: [{ properties: { province: '广东省', city: '深圳市', adcode: '440300' } }],
}

function stubFetch(options?: {
  geoOK?: boolean
  cityOK?: boolean
  apiItems?: Array<{
    adcode: string
    province: string
    city: string
    lat: number
    lng: number
    value: number
  }>
}) {
  const geoOK = options?.geoOK ?? true
  const cityOK = options?.cityOK ?? true
  const apiItems =
    options?.apiItems ?? [
      {
        adcode: '440300',
        province: '广东省',
        city: '深圳市',
        lat: 22.5431,
        lng: 114.0579,
        value: 5,
      },
    ]

  vi.stubGlobal(
    'fetch',
    vi.fn(async (input: string | URL | Request) => {
      const url = String(input)
      if (url.includes('/maps/china-cities.geojson')) {
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
    lastChartOption = null
    Object.defineProperty(window.navigator, 'userAgent', {
      value: 'unit-test',
      configurable: true,
    })
    stubFetch()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('loads city geojson and city api', async () => {
    const wrapper = mount(MapPage)
    await flushPage()

    expect(fetch).toHaveBeenCalled()
    const calls = (fetch as ReturnType<typeof vi.fn>).mock.calls.map((call) => String(call[0]))
    expect(calls.some((call) => call.includes('/maps/china-cities.geojson'))).toBe(true)
    expect(calls.some((call) => call.includes('/api/map/china'))).toBe(true)
    expect(calls.some((call) => call.includes('metric=conn'))).toBe(true)

    wrapper.unmount()
  })

  it('switches metric to bytes and renders human-readable units', async () => {
    stubFetch({
      apiItems: [
        {
          adcode: '440300',
          province: '广东省',
          city: '深圳市',
          lat: 22.5431,
          lng: 114.0579,
          value: 2048,
        },
      ],
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

  it('shows readable city name in tooltip for no-data regions', async () => {
    stubFetch({ apiItems: [] })
    const wrapper = mount(MapPage)
    await flushPage()

    expect(lastChartOption?.tooltip?.formatter).toBeTypeOf('function')
    const tooltip = String(lastChartOption.tooltip.formatter({ name: '440300' }))
    expect(tooltip).toContain('深圳市')
    expect(tooltip).not.toContain('440300<br/>')

    wrapper.unmount()
  })

  it('ignores unmapped rows when computing series and visualMap range', async () => {
    stubFetch({
      apiItems: [
        {
          adcode: '440300',
          province: '广东省',
          city: '深圳市',
          lat: 22.5431,
          lng: 114.0579,
          value: 5,
        },
        {
          adcode: 'unknown',
          province: '广东省',
          city: '未知城市',
          lat: 0,
          lng: 0,
          value: 999999,
        },
      ],
    })

    const wrapper = mount(MapPage)
    await flushPage()

    expect(lastChartOption?.series?.[0]?.data).toEqual([
      expect.objectContaining({ name: '440300', cityName: '深圳市', value: 5 }),
    ])
    expect(lastChartOption?.visualMap?.min).toBe(5)
    expect(lastChartOption?.visualMap?.max).toBe(6)

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
    stubFetch({ apiItems: [] })
    const wrapper = mount(MapPage)
    await flushPage()

    expect(wrapper.text()).toContain('当前窗口暂无城市指标数据')
    expect(wrapper.find('.chart').exists()).toBe(true)

    wrapper.unmount()
  })
})
