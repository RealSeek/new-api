import { render, screen } from '@testing-library/react'
import { describe, expect, test, vi } from 'vitest'

import { ModelBadge } from '../model-badge'

vi.mock('@/lib/lobe-icon', () => ({
  getLobeIcon: () => null,
}))

describe('用量日志模型徽标', () => {
  test('图片模型展示命中的分辨率档位', () => {
    render(<ModelBadge modelName='gpt-image-2' imageResolution='2k' />)

    expect(screen.getByText('gpt-image-2')).toBeVisible()
    expect(screen.getByText('2K')).toBeVisible()
  })

  test('未命中分辨率档位时不展示档位标记', () => {
    render(<ModelBadge modelName='gpt-image-2' />)

    expect(screen.queryByText('1K')).toBeNull()
    expect(screen.queryByText('2K')).toBeNull()
    expect(screen.queryByText('4K')).toBeNull()
  })
})
