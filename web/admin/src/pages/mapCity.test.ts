import { describe, expect, it } from 'vitest'
import { cityKey, createCityJoinKeyResolver, normalizeAdcode6, normalizeCityGeoFeatures, shouldKeepCityPolygon } from './mapCity'

describe('mapCity helpers', () => {
  it('normalizes 2/4/6-digit adcode to stable 6-digit code for city key', () => {
    expect(normalizeAdcode6('50')).toBe('500000')
    expect(normalizeAdcode6('5001')).toBe('500100')
    expect(normalizeAdcode6('429004')).toBe('429004')
    expect(cityKey({ adcode: '469001', province: '海南省', city: '五指山市' })).toBe('469001')
    expect(cityKey({ adcode: '659001', province: '新疆维吾尔自治区', city: '石河子市' })).toBe('659001')
    expect(cityKey({ adcode: '429004', province: '湖北省', city: '仙桃市' })).toBe('429004')
    expect(cityKey({ adcode: '50', province: '重庆市', city: '重庆市' })).toBe('500000')
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

  it('resolves API adcode join key: keeps 4/6-digit mapping and rejects ambiguous 2-digit province adcode', () => {
    const features = normalizeCityGeoFeatures([
      { properties: { province: '广东省', city: '深圳市', adcode: '广东-深圳' } },
      { properties: { province: '海南省', city: '临高县', adcode: '469024' } },
      { properties: { province: '重庆市', city: '重庆城区', adcode: '500100' } },
      { properties: { province: '重庆市', city: '重庆郊县', adcode: '500200' } },
    ] as any[])

    const resolve = createCityJoinKeyResolver(features)
    expect(resolve({ province: '广东省', city: '深圳市', adcode: '440300' })).toBe('广东-深圳')
    expect(resolve({ province: '海南省', city: '临高县', adcode: '469024' })).toBe('469024')
    expect(resolve({ province: '北京市', city: '北京市', adcode: '110100' })).toBe('110100')

    // 2位省级 adcode 不能强制落到城市 key
    expect(resolve({ province: '广东省', city: '深圳市', adcode: '44' })).toBe('')

    // 4位/6位 adcode 仍应稳定映射
    expect(resolve({ province: '重庆市', city: '重庆市', adcode: '5001' })).toBe('500100')
    expect(resolve({ province: '重庆市', city: '重庆市', adcode: '5002' })).toBe('500200')
    expect(resolve({ province: '重庆市', city: '重庆城区', adcode: '500100' })).toBe('500100')
  })
})
