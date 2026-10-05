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
import { describe, expect, it } from 'vitest'

import { formatGroupRate } from '../lib/format'
import { getGroupCategory, getGroupHealth } from '../lib/status'
import type { GroupStatusItem } from '../types'

const group: GroupStatusItem = {
  name: 'group',
  description: '',
  ratio: 1,
  models: [],
  endpoint_types: [],
  status: null,
}

describe('public group status', () => {
  it.each([
    [100, 'healthy'],
    [90, 'healthy'],
    [89.9, 'degraded'],
    [70, 'degraded'],
    [69.9, 'down'],
    [0, 'down'],
  ] as const)('grades recent availability %s as %s', (rate, health) => {
    expect(
      getGroupHealth({
        ...group,
        status: {
          request_count: 100,
          success_rate: 100,
          avg_latency_ms: 0,
          avg_tps: 0,
          series: [],
          recent: { hours: 1, request_count: 20, success_rate: rate },
        },
      })
    ).toBe(health)
  })

  it('keeps a group without samples idle', () => {
    expect(getGroupHealth(group)).toBe('idle')
  })

  it('does not round partial availability up to 100%', () => {
    expect(formatGroupRate(99.96, 'en')).toBe('99.9%')
    expect(formatGroupRate(null, 'en')).toBe('—')
  })

  it('classifies media-only groups by endpoints rather than mixed provider names', () => {
    expect(
      getGroupCategory({
        ...group,
        models: ['gpt-image-2', 'gemini-3-pro-image'],
        endpoint_types: ['image-generation'],
      })
    ).toBe('media')
    expect(
      getGroupCategory({
        ...group,
        models: ['claude-sonnet-4', 'claude-opus-4'],
        endpoint_types: ['anthropic'],
      })
    ).toBe('claude')
    expect(
      getGroupCategory({
        ...group,
        models: ['gpt-4o', 'claude-sonnet-4'],
        endpoint_types: ['openai'],
      })
    ).toBe('other')
  })
})
