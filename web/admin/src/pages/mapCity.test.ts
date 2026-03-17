import { describe, expect, it } from 'vitest'
import {
  cityKey,
  createCityJoinKeyResolver,
  extractProvinceBoundarySegments,
  normalizeAdcode6,
  normalizeCityGeoFeatures,
  shouldKeepCityPolygon,
} from './mapCity'

describe('mapCity helpers', () => {
  it('normalizes adcode and prefers unique normalized city-level keys', () => {
    expect(normalizeAdcode6('50')).toBe('500000')
    expect(normalizeAdcode6('5001')).toBe('500100')
    expect(normalizeAdcode6('429004')).toBe('429004')
    expect(cityKey({ adcode: '469001', province: '海南省', city: '五指山市' })).toBe('469001')
    expect(cityKey({ adcode: '659001', province: '新疆维吾尔自治区', city: '石河子市' })).toBe('659001')
    expect(cityKey({ adcode: '429004', province: '湖北省', city: '仙桃市' })).toBe('429004')
    expect(cityKey({ adcode: '50', province: '重庆市', city: '重庆市' })).toBe('')
    expect(cityKey({ adcode: '5001', province: '重庆市', city: '重庆市' })).toBe('500100')
  })

  it('retains municipality county/district polygons (e.g. Chongqing)', () => {
    expect(shouldKeepCityPolygon('渝中区', '重庆市')).toBe(true)
    expect(shouldKeepCityPolygon('城口县', '重庆市')).toBe(true)
    expect(shouldKeepCityPolygon('黄浦区', '上海市')).toBe(true)
  })

  it('still filters non-municipality county/district polygons except direct-admin county-level regions', () => {
    expect(shouldKeepCityPolygon('浦东新区', '浙江省')).toBe(false)
    expect(shouldKeepCityPolygon('城口县', '四川省')).toBe(false)
    expect(shouldKeepCityPolygon('神农架林区', '湖北省', '429021')).toBe(true)
    expect(shouldKeepCityPolygon('临高县', '海南省', '469024')).toBe(true)
    expect(shouldKeepCityPolygon('深圳市', '广东省')).toBe(true)
  })

  it('normalizes features with city_key/city_name and preserves municipality/direct-admin features', () => {
    const features = [
      { properties: { province: '重庆市', city: '渝中区', adcode: '500103' } },
      { properties: { province: '四川省', city: '城口县', adcode: '510229' } },
      { properties: { province: '湖北省', city: '神农架林区', adcode: '429021' } },
      { properties: { province: '海南省', city: '临高县', adcode: '469024' } },
      { properties: { province: '广东省', city: '深圳市', adcode: '440300' } },
    ]

    const out = normalizeCityGeoFeatures(features as any[])
    expect(out).toHaveLength(4)
    expect(out.map((f) => f.properties.city_name)).toEqual(['渝中区', '神农架林区', '临高县', '深圳市'])
    expect(out.map((f) => f.properties.city_key)).toEqual(['500103', '429021', '469024', '440300'])
  })

  it('resolves API join key to normalized adcode without alias collisions', () => {
    const features = normalizeCityGeoFeatures([
      { properties: { province: '海南省', city: '临高县', adcode: '469024' } },
      { properties: { province: '海南省', city: '五指山市', adcode: '469001' } },
      { properties: { province: '重庆市', city: '重庆城区', adcode: '500100' } },
      { properties: { province: '重庆市', city: '重庆郊县', adcode: '500200' } },
    ] as any[])

    const resolve = createCityJoinKeyResolver(features)
    expect(resolve({ province: '海南省', city: '临高县', adcode: '469024' })).toBe('469024')
    expect(resolve({ province: '海南省', city: '五指山市', adcode: '469001' })).toBe('469001')
    expect(resolve({ province: '北京市', city: '北京市', adcode: '110100' })).toBe('')

    // 2位省级 adcode 不映射到城市 key
    expect(resolve({ province: '广东省', city: '深圳市', adcode: '44' })).toBe('')

    // 4位/6位 adcode 统一落到规范化 6 位 city key
    expect(resolve({ province: '重庆市', city: '重庆市', adcode: '5001' })).toBe('500100')
    expect(resolve({ province: '重庆市', city: '重庆城区', adcode: '500100' })).toBe('500100')
    expect(resolve({ province: '重庆市', city: '重庆郊县', adcode: '500200' })).toBe('500200')
  })

  it('extracts province boundary segments without internal city seams', () => {
    const features = [
      {
        properties: { province: '甲省', city: '甲市一', adcode: '110100' },
        geometry: { type: 'Polygon', coordinates: [[[0, 0], [1, 0], [1, 1], [0, 1], [0, 0]]] },
      },
      {
        properties: { province: '甲省', city: '甲市二', adcode: '110200' },
        geometry: { type: 'Polygon', coordinates: [[[1, 0], [2, 0], [2, 1], [1, 1], [1, 0]]] },
      },
      {
        properties: { province: '乙省', city: '乙市一', adcode: '120100' },
        geometry: { type: 'Polygon', coordinates: [[[2, 0], [3, 0], [3, 1], [2, 1], [2, 0]]] },
      },
    ]

    const keyOf = (segment: number[][]) =>
      segment
        .map((point) => point.map((value) => value.toFixed(6)).join(','))
        .sort()
        .join('|')

    const segments = extractProvinceBoundarySegments(features as any[])
    const keys = new Set(segments.map((segment) => keyOf(segment)))

    expect(keys.has(keyOf([[1, 0], [1, 1]]))).toBe(false)
    expect(keys.has(keyOf([[0, 0], [0, 1]]))).toBe(true)
    expect(keys.has(keyOf([[2, 0], [2, 1]]))).toBe(true)
  })
})
