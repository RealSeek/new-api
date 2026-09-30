/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { describe, expect, test } from 'vitest'

import {
  buildModelSnapshots,
  getPriceSummary,
} from '../model-pricing-snapshots'

const emptyRatios = {
  modelPrice: '{}',
  modelRatio: '{}',
  cacheRatio: '{}',
  createCacheRatio: '{}',
  completionRatio: '{}',
  imageRatio: '{}',
  audioRatio: '{}',
  audioCompletionRatio: '{}',
  billingExpr: '{}',
  imagePrice: '{}',
}

describe('按秒模型定价快照', () => {
  test('恢复按秒模式及分辨率价格摘要', () => {
    const snapshots = buildModelSnapshots({
      ...emptyRatios,
      billingMode: '{"video-test":"per_second"}',
      videoPrice: JSON.stringify({
        'video-test': {
          default_price: 0.2,
          default_duration: 5,
          billing_step: 1,
          minimum_duration: 1,
          resolution_prices: { '480p': 0.38, '720p': 0.41 },
        },
      }),
    })

    expect(snapshots).toHaveLength(1)
    expect(snapshots[0]).toMatchObject({
      name: 'video-test',
      billingMode: 'per-second',
      videoPrice: {
        default_price: 0.2,
        resolution_prices: { '480p': 0.38, '720p': 0.41 },
      },
    })
    expect(getPriceSummary(snapshots[0], (key) => key)).toBe(
      '480P $0.38/second · 720P $0.41/second'
    )
  })
})

describe('按次图片模型定价快照', () => {
  test('恢复 1K/2K/4K 图片价格并展示摘要', () => {
    const snapshots = buildModelSnapshots({
      ...emptyRatios,
      billingMode: '{}',
      imagePrice: JSON.stringify({
        'gpt-image-2': { '1k': 0.05, '2k': 0.1, '4k': 0.2 },
      }),
    })

    expect(snapshots).toHaveLength(1)
    expect(snapshots[0]).toMatchObject({
      name: 'gpt-image-2',
      imagePrice: { '1k': 0.05, '2k': 0.1, '4k': 0.2 },
    })
    expect(getPriceSummary(snapshots[0], (key) => key)).toBe(
      '1K $0.05/image · 2K $0.1/image · 4K $0.2/image'
    )
  })
})
