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
import { getSuccessRateLevel } from '@/features/performance-metrics/lib/format'
import { resolveModelProvider } from '@/lib/model-provider'

import type { GroupCategory, GroupHealth, GroupStatusItem } from '../types'

const MEDIA_ENDPOINT_TYPES = new Set(['image-generation', 'openai-video'])

const CHINESE_MODEL_PROVIDERS = new Set([
  '360 AI',
  'Baichuan',
  'Baidu',
  'DeepSeek',
  'Doubao',
  'iFlytek',
  'InternLM',
  'Jimeng',
  'Kling',
  'MiMo',
  'MiniMax',
  'Moonshot',
  'Qwen',
  'StepFun',
  'Tencent',
  'Vidu',
  'Wan',
  'Yi',
  'Zhipu',
])

/** Share of a group's models one provider needs before the group is named after it. */
const DOMINANT_PROVIDER_SHARE = 0.6

export type GroupProvider = { name: string; icon: string }

/**
 * Health follows the shared success-rate grading over the group's recent
 * window. Groups without samples in the last 24 hours are idle, not healthy.
 */
export function getGroupHealth(group: GroupStatusItem): GroupHealth {
  if (!group.status) return 'idle'
  const level = getSuccessRateLevel(group.status.recent.success_rate)
  if (level === 'excellent' || level === 'good') return 'healthy'
  if (level === 'warning') return 'degraded'
  if (level === 'critical') return 'down'
  return 'idle'
}

/** The provider behind most of a group's models, when one clearly dominates. */
export function getGroupProvider(group: GroupStatusItem): GroupProvider | null {
  const counts = new Map<string, { provider: GroupProvider; count: number }>()
  for (const modelName of group.models) {
    const provider = resolveModelProvider(modelName)
    if (!provider) continue
    const entry = counts.get(provider.name)
    if (entry) {
      entry.count += 1
    } else {
      counts.set(provider.name, {
        provider: { name: provider.name, icon: provider.icon },
        count: 1,
      })
    }
  }
  let dominant: { provider: GroupProvider; count: number } | null = null
  for (const entry of counts.values()) {
    if (!dominant || entry.count > dominant.count) dominant = entry
  }
  if (
    !dominant ||
    dominant.count < group.models.length * DOMINANT_PROVIDER_SHARE
  ) {
    return null
  }
  return dominant.provider
}

export function getGroupCategory(group: GroupStatusItem): GroupCategory {
  const isMediaOnly =
    group.endpoint_types.length > 0 &&
    group.endpoint_types.every((type) => MEDIA_ENDPOINT_TYPES.has(type))
  if (isMediaOnly) return 'media'

  const provider = getGroupProvider(group)
  if (!provider) return 'other'
  if (provider.name === 'Anthropic') return 'claude'
  if (provider.name === 'OpenAI') return 'openai'
  if (provider.name === 'Gemini') return 'gemini'
  if (CHINESE_MODEL_PROVIDERS.has(provider.name)) return 'china'
  return 'other'
}

/** Status dot / flow colors shared by the topology and the status board. */
export const GROUP_HEALTH_COLOR: Record<
  GroupHealth,
  { dot: string; ring: string; stroke: string; text: string }
> = {
  healthy: {
    dot: 'bg-emerald-500',
    ring: 'bg-emerald-500/20',
    stroke: 'stroke-emerald-500',
    text: 'text-emerald-600 dark:text-emerald-400',
  },
  degraded: {
    dot: 'bg-amber-500',
    ring: 'bg-amber-500/20',
    stroke: 'stroke-amber-500',
    text: 'text-amber-600 dark:text-amber-400',
  },
  down: {
    dot: 'bg-red-500',
    ring: 'bg-red-500/20',
    stroke: 'stroke-red-500',
    text: 'text-red-600 dark:text-red-400',
  },
  idle: {
    dot: 'bg-muted-foreground/40',
    ring: 'bg-muted-foreground/10',
    stroke: 'stroke-muted-foreground/40',
    text: 'text-muted-foreground',
  },
}
