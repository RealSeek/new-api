import { render, screen, within } from '@testing-library/react'
import { describe, expect, test, vi } from 'vitest'

import type { PricingModel } from '../../types'
import { ModelCard } from '../model-card'

vi.mock('@/lib/lobe-icon', () => ({
  getLobeIcon: () => null,
}))

describe('模型广场模型卡片', () => {
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
