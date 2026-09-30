import { describe, expect, test } from 'vitest'

import { shouldShowRequestConversion } from '../format'

describe('请求转换区块可见性', () => {
  test('日志归属者有请求路径时可见（不限管理员）', () => {
    expect(shouldShowRequestConversion(2, '/v1/images/edits', [])).toBe(true)
  })

  test('只有格式转换链路时同样可见', () => {
    expect(
      shouldShowRequestConversion(2, undefined, ['OpenAI Images'])
    ).toBe(true)
  })

  test('退款日志不展示请求转换', () => {
    expect(shouldShowRequestConversion(6, '/v1/images/edits', [])).toBe(false)
  })

  test('既无路径也无转换链路时不展示', () => {
    expect(shouldShowRequestConversion(2, undefined, [])).toBe(false)
  })
})
