import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
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
  features: [
    {
      properties: { province: '广东省', city: '深圳市', adcode: '440300' },
      geometry: { type: 'Polygon', coordinates: [[[113, 22], [114, 22], [114, 23], [113, 23], [113, 22]]] },
    },
  ],
}

function stubFetch(options?: {
  geoOK?: boolean
  cityOK?: boolean
  geoJSON?: any
  apiItems?: Array<{
    adcode: string
    province: string
    city: string
    lat: number
    lng: number
    value?: number
    connValue?: number
    bytesValue?: number
  }>
}) {
  const geoOK = options?.geoOK ?? true
  const cityOK = options?.cityOK ?? true
  const mapGeoJSON = options?.geoJSON ?? geoJSON
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
      if (url.includes('/api/analytics/options?')) {
        return {
          ok: true,
          status: 200,
          json: async () => ({
            rules: ['r1'],
            provinces: ['广东省'],
            cities: [{ province: '广东省', city: '深圳市' }],
            statuses: ['ok'],
          }),
        }
      }
      if (url.includes('/maps/china-cities.geojson')) {
        return {
          ok: geoOK,
          status: geoOK ? 200 : 500,
          json: async () => mapGeoJSON,
        }
      }
      if (url.includes('/api/map/china')) {
        const metric = /[?&]metric=([^&]+)/.exec(url)?.[1] || 'conn'
        const items = apiItems.map((item) => ({
          ...item,
          value: metric === 'bytes' ? (item.bytesValue ?? (item.value ?? 0) * 1024) : (item.connValue ?? item.value ?? 0),
        }))
        return {
          ok: cityOK,
          status: cityOK ? 200 : 500,
          json: async () => ({ items }),
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

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason?: unknown) => void
  const promise = new Promise<T>((innerResolve, innerReject) => {
    resolve = innerResolve
    reject = innerReject
  })
  return { promise, resolve, reject }
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
    expect(calls.some((call) => call.includes('metric=bytes'))).toBe(true)

    wrapper.unmount()
  })

  it('defaults to 1d window requests', async () => {
    const wrapper = mount(MapPage)
    await flushPage()

    const calls = (fetch as ReturnType<typeof vi.fn>).mock.calls.map((call) => String(call[0]))
    const apiCalls = calls.filter((call) => call.includes('/api/'))
    expect(apiCalls.length).toBeGreaterThan(0)
    expect(apiCalls.every((call) => call.includes('window=1d'))).toBe(true)

    wrapper.unmount()
  })

  it('renders interactive city heatmap with tooltip and province boundary overlay', async () => {
    const wrapper = mount(MapPage)
    await flushPage()

    expect(lastChartOption?.series?.filter((series: any) => series.type === 'map')).toHaveLength(1)
    expect(lastChartOption?.series?.some((series: any) => series.type === 'lines')).toBe(true)

    const tooltip = String(lastChartOption.tooltip.formatter({ name: '440300' }))
    expect(tooltip).toContain('广东省 / 深圳市')
    expect(tooltip).toContain('深圳市')
    expect(tooltip).toContain('连接数: 0')
    expect(tooltip).toContain('流量: 0 B')

    wrapper.unmount()
  })

  it('keeps city hover interaction enabled while province boundary overlay stays silent', async () => {
    const wrapper = mount(MapPage)
    await flushPage()

    expect(lastChartOption?.geo?.silent).not.toBe(true)
    expect(lastChartOption?.geo?.emphasis?.disabled).not.toBe(true)
    expect(lastChartOption?.series?.[1]?.type).toBe('lines')
    expect(lastChartOption?.series?.[1]?.silent).toBe(true)

    wrapper.unmount()
  })

  it('renders stronger province boundary styling on top of city polygons', async () => {
    const wrapper = mount(MapPage)
    await flushPage()

    expect(lastChartOption?.series?.[1]?.type).toBe('lines')
    expect(lastChartOption?.series?.[1]?.zlevel).toBe(2)
    expect(lastChartOption?.series?.[1]?.z).toBe(20)
    expect(lastChartOption?.series?.[1]?.lineStyle).toMatchObject({
      color: '#16324f',
      width: 2.8,
      opacity: 1,
    })

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

    await wrapper.findAll('select')[3].setValue('bytes')
    await flushPage()

    const calls = (fetch as ReturnType<typeof vi.fn>).mock.calls.map((call) => String(call[0]))
    expect(calls.some((call) => call.includes('metric=bytes'))).toBe(true)
    expect(wrapper.text()).toContain('2.00 MB')

    wrapper.unmount()
  })

  it('renders compact city stats in the right sidebar', async () => {
    const wrapper = mount(MapPage)
    await flushPage()

    expect(wrapper.find('.map-layout').exists()).toBe(true)
    expect(wrapper.find('.map-sidebar .city-list').exists()).toBe(true)
    expect(wrapper.find('.map-main .city-list').exists()).toBe(false)

    const firstItem = wrapper.get('.city-list li')
    expect(firstItem.text()).toContain('深圳市')
    expect(firstItem.text()).not.toContain('连 5')
    expect(firstItem.text()).not.toContain('流 5.00 KB')
    expect(firstItem.text()).not.toContain('连接 5 · 流量 5.00 KB')

    const chips = firstItem.findAll('.city-stat')
    expect(chips).toHaveLength(2)
    expect(chips[0].attributes('title')).toBe('连接数 5')
    expect(chips[1].attributes('title')).toBe('流量 5.00 KB')
    expect(firstItem.findAll('.city-stat-icon')).toHaveLength(2)
    expect(firstItem.find('.city-stat-icon--conn').exists()).toBe(true)
    expect(firstItem.find('.city-stat-icon--bytes').exists()).toBe(true)
    expect(firstItem.findAll('.city-stat-icon').every((icon) => icon.attributes('aria-hidden') === 'true')).toBe(true)
    expect(firstItem.findAll('.sr-only').map((node) => node.text())).toEqual(['连接数', '流量'])
    expect(firstItem.findAll('.city-stat-value').map((node) => node.text())).toEqual(['5', '5.00 KB'])
    expect(wrapper.text()).not.toContain('已载入 1 城市 · Top 1000 上限')

    wrapper.unmount()
  })

  it('renders city list inside a bounded scroll container', async () => {
    const wrapper = mount(MapPage)
    await flushPage()

    const scrollArea = wrapper.find('.city-list-scroll')
    expect(scrollArea.exists()).toBe(true)
    expect(scrollArea.find('.city-list').exists()).toBe(true)

    wrapper.unmount()
  })

  it('renders map and sidebar inside shared bounded layout shells', async () => {
    const wrapper = mount(MapPage)
    await flushPage()

    const mapShell = wrapper.find('.map-main-shell')
    expect(mapShell.exists()).toBe(true)
    expect(mapShell.find('.chart').exists()).toBe(true)

    const sidebarBody = wrapper.find('.map-sidebar-body')
    expect(sidebarBody.exists()).toBe(true)
    expect(sidebarBody.find('.city-list-scroll').exists()).toBe(true)
    expect(sidebarBody.find('.city-list').exists()).toBe(true)

    wrapper.unmount()
  })

  it('uses a viewport-aware desktop map height formula to better fill page height', () => {
    const source = readFileSync(resolve(process.cwd(), 'src/pages/MapPage.vue'), 'utf8')

    expect(source).toContain('--map-panel-height: clamp(500px, calc(100dvh - 320px), 680px);')
  })

  it('uses explicit geo bounds to reduce bottom blank space', async () => {
    const wrapper = mount(MapPage)
    await flushPage()

    expect(lastChartOption?.geo).toMatchObject({
      top: 16,
      bottom: 64,
      left: 12,
      right: 12,
    })

    wrapper.unmount()
  })

  it('uses one metric selector for map coloring and sidebar order', async () => {
    stubFetch({
      geoJSON: {
        type: 'FeatureCollection',
        features: [
          {
            properties: { province: '广东省', city: '深圳市', adcode: '440300' },
            geometry: { type: 'Polygon', coordinates: [[[113, 22], [114, 22], [114, 23], [113, 23], [113, 22]]] },
          },
          {
            properties: { province: '广东省', city: '广州市', adcode: '440100' },
            geometry: { type: 'Polygon', coordinates: [[[112, 23], [113, 23], [113, 24], [112, 24], [112, 23]]] },
          },
        ],
      },
      apiItems: [
        {
          adcode: '440300',
          province: '广东省',
          city: '深圳市',
          lat: 22.5431,
          lng: 114.0579,
          connValue: 20,
          bytesValue: 1024,
        },
        {
          adcode: '440100',
          province: '广东省',
          city: '广州市',
          lat: 23.1291,
          lng: 113.2644,
          connValue: 5,
          bytesValue: 4096,
        },
      ],
    })

    const wrapper = mount(MapPage)
    await flushPage()

    const listBefore = wrapper.findAll('.city-list li').map((item) => item.text())
    expect(listBefore[0]).toContain('深圳市')
    expect(wrapper.findAll('select')).toHaveLength(5)

    await wrapper.findAll('select')[3].setValue('bytes')
    await flushPage()

    const listAfter = wrapper.findAll('.city-list li').map((item) => item.text())
    expect(listAfter[0]).toContain('广州市')
    expect(lastChartOption?.visualMap?.formatter(4096)).toBe('4.00 KB')

    wrapper.unmount()
  })

  it('shows readable city name and zero metrics in tooltip for no-data regions', async () => {
    stubFetch({ apiItems: [] })
    const wrapper = mount(MapPage)
    await flushPage()

    expect(lastChartOption?.tooltip?.formatter).toBeTypeOf('function')
    const tooltip = String(lastChartOption.tooltip.formatter({ name: '440300' }))
    expect(tooltip).toContain('广东省 / 深圳市')
    expect(tooltip).toContain('深圳市')
    expect(tooltip).not.toContain('440300<br/>')
    expect(tooltip).toContain('连接数: 0')
    expect(tooltip).toContain('流量: 0 B')

    wrapper.unmount()
  })

  it('shows readable city name in hover label for no-data regions', async () => {
    stubFetch({ apiItems: [] })
    const wrapper = mount(MapPage)
    await flushPage()

    expect(lastChartOption?.series?.[0]?.emphasis?.label?.formatter).toBeTypeOf('function')
    const label = String(lastChartOption.series[0].emphasis.label.formatter({ name: '440300' }))
    expect(label).toBe('深圳市')

    wrapper.unmount()
  })

  it('keeps no-data hover naming stable for direct-admin city keys that previously collided', async () => {
    stubFetch({
      geoJSON: {
        type: 'FeatureCollection',
        features: [
          {
            properties: { province: '海南省', city: '五指山市', adcode: '469001' },
            geometry: { type: 'Polygon', coordinates: [[[109, 18], [110, 18], [110, 19], [109, 19], [109, 18]]] },
          },
          {
            properties: { province: '海南省', city: '临高县', adcode: '469024' },
            geometry: { type: 'Polygon', coordinates: [[[108, 19], [109, 19], [109, 20], [108, 20], [108, 19]]] },
          },
        ],
      },
      apiItems: [],
    })
    const wrapper = mount(MapPage)
    await flushPage()

    const tooltip = String(lastChartOption.tooltip.formatter({ name: '469024' }))
    expect(tooltip).toContain('临高县')
    expect(tooltip).not.toContain('五指山市')

    const label = String(lastChartOption.series[0].emphasis.label.formatter({ name: '469024' }))
    expect(label).toBe('临高县')

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

  it('keeps the geo base layer with province boundary lines overlay', async () => {
    const wrapper = mount(MapPage)
    await flushPage()

    expect(lastChartOption?.geo?.map).toBe('china-cities')
    expect(lastChartOption?.series).toHaveLength(2)
    expect(lastChartOption?.series?.[0]?.type).toBe('map')
    expect(lastChartOption?.series?.[1]?.type).toBe('lines')
    expect(lastChartOption?.series?.[1]?.name).toBe('省界')

    wrapper.unmount()
  })

  it('omits redundant map meta hints while preserving loading error and empty-state messages', async () => {
    const optionsRequest = deferred<{
      ok: boolean
      status: number
      json: () => Promise<any>
    }>()
    const geoRequest = deferred<{
      ok: boolean
      status: number
      json: () => Promise<any>
    }>()

    vi.stubGlobal(
      'fetch',
      vi.fn(async (input: string | URL | Request) => {
        const url = String(input)
        if (url.includes('/api/analytics/options?')) {
          return optionsRequest.promise
        }
        if (url.includes('/maps/china-cities.geojson')) {
          return geoRequest.promise
        }
        if (url.includes('/api/map/china')) {
          return {
            ok: true,
            status: 200,
            json: async () => ({ items: [] }),
          }
        }
        throw new Error(`unexpected fetch url: ${url}`)
      }),
    )

    const loadingWrapper = mount(MapPage)
    await flushPage()

    expect(loadingWrapper.text()).not.toContain('城市连接热度（市级边界） · 分析型页面（不自动刷新）')
    expect(loadingWrapper.text()).not.toContain('已载入 1 城市 · Top 1000 上限')
    expect(loadingWrapper.text()).toContain('筛选项加载中...')

    optionsRequest.resolve({
      ok: true,
      status: 200,
      json: async () => ({
        rules: ['r1'],
        provinces: ['广东省'],
        cities: [{ province: '广东省', city: '深圳市' }],
        statuses: ['ok'],
      }),
    })
    await flushPage()

    expect(loadingWrapper.text()).toContain('加载中...')

    geoRequest.resolve({
      ok: true,
      status: 200,
      json: async () => geoJSON,
    })
    await flushPage()

    expect(loadingWrapper.text()).toContain('当前窗口暂无城市指标数据')
    loadingWrapper.unmount()

    stubFetch({ geoOK: false })
    const errorWrapper = mount(MapPage)
    await flushPage()

    expect(errorWrapper.text()).toContain('底图加载失败')
    expect(errorWrapper.text()).not.toContain('城市连接热度（市级边界） · 分析型页面（不自动刷新）')
    expect(errorWrapper.text()).not.toContain('已载入 1 城市 · Top 1000 上限')

    errorWrapper.unmount()
  })

  it('renders full returned city list without hardcoded 12-row truncation', async () => {
    const features = Array.from({ length: 13 }).map((_, idx) => {
      const adcode = String(440300 + idx)
      const baseLng = 113 + idx * 0.1
      const baseLat = 22 + idx * 0.1
      return {
        properties: { province: '广东省', city: `城市${idx + 1}`, adcode },
        geometry: {
          type: 'Polygon',
          coordinates: [
            [
              [baseLng, baseLat],
              [baseLng + 0.05, baseLat],
              [baseLng + 0.05, baseLat + 0.05],
              [baseLng, baseLat + 0.05],
              [baseLng, baseLat],
            ],
          ],
        },
      }
    })

    stubFetch({
      geoJSON: { type: 'FeatureCollection', features },
      apiItems: Array.from({ length: 13 }).map((_, idx) => ({
        adcode: String(440300 + idx),
        province: '广东省',
        city: `城市${idx + 1}`,
        lat: 22.5 + idx * 0.01,
        lng: 114.0 + idx * 0.01,
        value: 100 - idx,
      })),
    })
    const wrapper = mount(MapPage)
    await flushPage()

    expect(wrapper.text()).toContain('城市13')

    wrapper.unmount()
  })
})
