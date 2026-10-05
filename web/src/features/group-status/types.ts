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
import type { PerformanceSummary } from '@/features/performance-metrics/types'

export type GroupHourPoint = {
  ts: number
  request_count: number
  success_rate: number
}

export type GroupStatusMetrics = {
  request_count: number
  success_rate: number
  avg_latency_ms: number
  avg_tps: number
  /** Shortest trailing window (hours) with enough samples to judge health. */
  recent: { hours: number; request_count: number; success_rate: number }
  /** One point per hour across the window; hours without traffic count 0. */
  series: GroupHourPoint[]
}

export type GroupStatusItem = {
  name: string
  description: string
  ratio: number
  /** Busiest models first. */
  models: string[]
  endpoint_types: string[]
  /** Null when the group had no samples in the window. */
  status: GroupStatusMetrics | null
}

export type GroupStatusBoard = {
  summary: PerformanceSummary | null
  request_count: number
  window_start: number
  window_end: number
  /** Busiest groups first. */
  groups: GroupStatusItem[]
}

export type GroupStatusResponse = {
  success: boolean
  message?: string
  data: GroupStatusBoard
}

export type GroupHealth = 'healthy' | 'degraded' | 'down' | 'idle'

export type GroupCategory =
  | 'claude'
  | 'openai'
  | 'gemini'
  | 'china'
  | 'media'
  | 'other'
