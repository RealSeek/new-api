import { render, screen, within } from '@testing-library/react'
import { describe, expect, test, vi } from 'vitest'

import type { PricingModel } from '../../types'
import { ModelCard } from '../model-card'

vi.mock('@/lib/lobe-icon', () => ({
  getLobeIcon: () => null,
}))

describe('模型广场模型卡片', () => {
  test('视频任务表达式按分辨率逐行显示，合并参考视频同价行并忽略旧价格', () => {
    const model: PricingModel = {
      id: 3,
      model_name: 'seedance-2.0',
      quota_type: 0,
      model_ratio: 1,
      completion_ratio: 1,
      enable_groups: [],
      billing_mode: 'tiered_expr',
      billing_expr:
        'u("resolution") == "480p" ? tier("480p", u("seconds") * 0.57) : u("resolution") == "720p" ? tier("720p", u("seconds") * 0.95) : u("resolution") == "1080p" ? tier("1080p", u("seconds") * 2.2) : tier("4k", u("seconds") * 4.3)',
      billing_usage_schema: {
        resolution: {
          enum: ['480p', '720p', '1080p', '4k'],
          allowCustomValues: true,
        },
        seconds: { type: 'number', unit: 'second' },
        tokens: { type: 'number', unit: 'token' },
        video_input: { type: 'boolean' },
      },
      resolution_alias_prices: [
        { resolution: '480p', price: 0.1, unit: 'second' },
      ],
    }
    render(<ModelCard model={model} onClick={vi.fn()} />)
    const list = screen.getByRole('list', { name: 'Resolution prices' })
    expect(list).toHaveClass('flex-col')
    const rows = within(list).getAllByRole('listitem')
    expect(rows).toHaveLength(4)
    for (const [index, resolution, price] of [
      [0, '480P', '$0.57'],
      [1, '720P', '$0.95'],
      [2, '1080P', '$2.2'],
      [3, '4K', '$4.3'],
    ] as const) {
      expect(rows[index]).toHaveTextContent(resolution)
      expect(within(rows[index]).getByText(price)).toBeVisible()
      expect(rows[index]).toHaveTextContent('/ second')
    }
    expect(screen.queryByText('$0.57 – $4.3')).not.toBeInTheDocument()
    expect(screen.queryByText('$0.1')).not.toBeInTheDocument()
    expect(screen.queryByText(/1M token|video_input/)).not.toBeInTheDocument()
  })

  test('任务卡片只显示表达式配置的分辨率和非零计费单位', () => {
    const model: PricingModel = {
      id: 6,
      model_name: 'MiniMax-H3',
      quota_type: 0,
      model_ratio: 1,
      completion_ratio: 1,
      enable_groups: [],
      billing_mode: 'tiered_expr',
      billing_expr:
        'u("resolution") == "768P" ? tier("768P", u("input_images") * 0 + u("seconds") * 0.12) : tier("base", u("input_images") * 0 + u("seconds") * 0.12)',
      billing_usage_schema: {
        resolution: { enum: ['768P', '2K'] },
        input_images: { type: 'number', unit: 'count' },
        seconds: { type: 'number', unit: 'second' },
      },
    }

    render(<ModelCard model={model} onClick={vi.fn()} />)

    const rows = within(
      screen.getByRole('list', { name: 'Resolution prices' })
    ).getAllByRole('listitem')
    expect(rows).toHaveLength(1)
    expect(rows[0]).toHaveTextContent('768P')
    expect(rows[0]).toHaveTextContent('$0.12 / second')
    expect(rows[0]).not.toHaveTextContent('/ request')
    expect(screen.queryByText('2K')).not.toBeInTheDocument()
  })

  test('任务 Token 定价显示自定义分辨率，并保留百万 Token 单位和分组倍率', () => {
    const model: PricingModel = {
      id: 4,
      model_name: 'seedance-custom',
      quota_type: 0,
      model_ratio: 1,
      completion_ratio: 1,
      enable_groups: ['vip'],
      group_ratio: { vip: 2 },
      billing_mode: 'tiered_expr',
      billing_expr:
        'u("resolution") == "1440p" ? tier("1440p", u("tokens") * 70 / 1000000) : tier("720p", u("tokens") * 42 / 1000000)',
      billing_usage_schema: {
        resolution: { enum: ['720p'], allowCustomValues: true },
        seconds: { type: 'number', unit: 'second' },
        tokens: { type: 'number', unit: 'token' },
      },
    }
    render(
      <ModelCard
        model={model}
        onClick={vi.fn()}
        selectedGroup='vip'
        tokenUnit='K'
      />
    )
    const rows = within(
      screen.getByRole('list', { name: 'Resolution prices' })
    ).getAllByRole('listitem')
    expect(rows).toHaveLength(2)
    expect(rows[0]).toHaveTextContent('720P')
    expect(rows[0]).toHaveTextContent('$84 / 1M token')
    expect(rows[1]).toHaveTextContent('1440P')
    expect(rows[1]).toHaveTextContent('$140 / 1M token')
    expect(screen.queryByText(/\/ second/)).not.toBeInTheDocument()
  })

  test('参考视频需要附加费时保留两种条件及每次附加费', () => {
    const model: PricingModel = {
      id: 5,
      model_name: 'seedance-reference',
      quota_type: 0,
      model_ratio: 1,
      completion_ratio: 1,
      enable_groups: [],
      billing_mode: 'tiered_expr',
      billing_expr:
        'u("video_input") == true ? tier("reference", u("seconds") * 0.95 + 1.2) : tier("base", u("seconds") * 0.95)',
      billing_usage_schema: {
        resolution: { enum: ['720p'] },
        seconds: { type: 'number', unit: 'second' },
        video_input: {
          type: 'boolean',
          description: { en: 'Reference video present' },
        },
      },
    }
    render(<ModelCard model={model} onClick={vi.fn()} />)
    const rows = within(
      screen.getByRole('list', { name: 'Resolution prices' })
    ).getAllByRole('listitem')
    expect(rows).toHaveLength(2)
    expect(rows[0]).toHaveTextContent('Reference video present: No')
    expect(rows[0]).toHaveTextContent('$0.95 / second')
    expect(rows[1]).toHaveTextContent('Reference video present: Yes')
    expect(rows[1]).toHaveTextContent('Additional charge')
    expect(rows[1]).toHaveTextContent('$1.2 / request')
  })

  test('按秒模型的多个分辨率价格按行竖直排列', () => {
    const model: PricingModel = {
      id: 1,
      model_name: 'seedance-2.5',
      quota_type: 2,
      model_ratio: 1,
      completion_ratio: 1,
      enable_groups: [],
      video_price: {
        default_price: 0.35,
        default_duration: 5,
        billing_step: 5,
        minimum_duration: 5,
        resolution_prices: {
          '480p': 0.35,
          '720p': 0.5,
        },
      },
    }

    render(<ModelCard model={model} onClick={() => undefined} />)

    const priceList = screen.getByRole('list', { name: 'Resolution prices' })
    expect(priceList).toHaveClass('flex-col')
    const priceRows = within(priceList).getAllByRole('listitem')
    expect(priceRows).toHaveLength(2)
    expect(within(priceRows[0]).getByText('480P')).toBeVisible()
    expect(within(priceRows[0]).getByText('$0.35')).toBeVisible()
    expect(priceRows[0]).toHaveTextContent('/ second')
    expect(within(priceRows[1]).getByText('720P')).toBeVisible()
    expect(within(priceRows[1]).getByText('$0.5')).toBeVisible()
    expect(priceRows[1]).toHaveTextContent('/ second')
  })

  test('按分辨率计价的图片模型展示 1K/2K/4K 单张价格', () => {
    const model: PricingModel = {
      id: 2,
      model_name: 'gpt-image-2',
      quota_type: 1,
      model_ratio: 1,
      model_price: 0.05,
      completion_ratio: 1,
      enable_groups: [],
      image_price: {
        '1k': 0.05,
        '2k': 0.1,
        '4k': 0.2,
      },
    }

    render(<ModelCard model={model} onClick={() => undefined} />)

    const priceList = screen.getByRole('list', { name: 'Resolution prices' })
    expect(priceList).toHaveClass('flex-col')
    const priceRows = within(priceList).getAllByRole('listitem')
    expect(priceRows).toHaveLength(3)
    expect(within(priceRows[0]).getByText('1K')).toBeVisible()
    expect(within(priceRows[0]).getByText('$0.05')).toBeVisible()
    expect(priceRows[0]).toHaveTextContent('/ image')
    expect(within(priceRows[1]).getByText('2K')).toBeVisible()
    expect(within(priceRows[1]).getByText('$0.1')).toBeVisible()
    expect(within(priceRows[2]).getByText('4K')).toBeVisible()
    expect(within(priceRows[2]).getByText('$0.2')).toBeVisible()
    expect(priceRows[2]).toHaveTextContent('/ image')
  })
})
