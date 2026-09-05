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
})
