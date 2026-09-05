import { describe, expect, test } from 'vitest'

import { CHANNEL_TYPE_RS_GATEWAY } from '../../constants'
import { channelSchema } from '../../types'
import {
  CHANNEL_FORM_DEFAULT_VALUES,
  readGatewayEndpoints,
  writeGatewayEndpoints,
  transformFormDataToCreatePayload,
  transformFormDataToUpdatePayload,
  transformChannelToFormDefaults,
} from '../channel-form'

describe('网关端点保存', () => {
  test('选择图片生成后创建、更新和重新打开均保留同一端点', () => {
    const settings = writeGatewayEndpoints('{"custom_flag":true}', [
      'image-generation',
    ])
    const form = {
      ...CHANNEL_FORM_DEFAULT_VALUES,
      name: '图片网关',
      key: 'test-key',
      models: 'gpt-image-2',
      type: CHANNEL_TYPE_RS_GATEWAY,
      settings,
    }
    const created = transformFormDataToCreatePayload(form).channel
    const updated = transformFormDataToUpdatePayload(form, 61)
    for (const payload of [created, updated]) {
      expect(JSON.parse(payload.settings || '{}')).toMatchObject({
        custom_flag: true,
        supported_endpoint_types: ['image-generation'],
      })
      const reopened = transformChannelToFormDefaults(
        channelSchema.parse({
          ...payload,
          id: 61,
          status: 1,
          key: '',
          other: '',
          remark: '',
          created_time: 0,
          test_time: 0,
          response_time: 0,
          balance_updated_time: 0,
        })
      )
      expect(readGatewayEndpoints(reopened.settings)).toEqual([
        'image-generation',
      ])
    }
  })

  test('清空选择覆盖旧 other 字段，不恢复失效的旧端点', () => {
    const settings = writeGatewayEndpoints('{}', [])
    expect(
      readGatewayEndpoints(settings, '{"supported_endpoint_types":["gemini"]}')
    ).toEqual([])
  })

  test('旧版 other 中的端点可以回显并迁入 settings', () => {
    const other = '{"supported_endpoint_types":["anthropic"]}'
    expect(readGatewayEndpoints('{}', other)).toEqual(['anthropic'])
    const payload = transformFormDataToUpdatePayload(
      { ...CHANNEL_FORM_DEFAULT_VALUES, type: CHANNEL_TYPE_RS_GATEWAY, other },
      61
    )
    expect(readGatewayEndpoints(payload.settings)).toEqual(['anthropic'])
  })
})
