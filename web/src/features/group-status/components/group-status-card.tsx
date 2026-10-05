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
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { GroupMultiplierBadge } from '@/components/group-badge'
import {
  formatLatency,
  formatThroughput,
} from '@/features/performance-metrics/lib/format'
import { toIntlLocale } from '@/i18n/languages'
import { formatCompactNumber } from '@/lib/format'
import { getLobeIcon } from '@/lib/lobe-icon'
import { cn } from '@/lib/utils'

import { formatGroupRate } from '../lib/format'
import {
  GROUP_HEALTH_COLOR,
  getGroupHealth,
  getGroupProvider,
} from '../lib/status'
import type { GroupHealth, GroupStatusItem } from '../types'
import { GroupUptimeBars } from './group-uptime-bars'

const PREVIEW_MODEL_COUNT = 3

const HEALTH_LABEL: Record<GroupHealth, string> = {
  healthy: 'Operational',
  degraded: 'Degraded',
  down: 'Unavailable',
  idle: 'No recent traffic',
}

type GroupStatusCardProps = {
  group: GroupStatusItem
  className?: string
}

export function GroupStatusCard(props: GroupStatusCardProps) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const group = props.group
  const health = getGroupHealth(group)
  const color = GROUP_HEALTH_COLOR[health]
  const provider = getGroupProvider(group)
  const previewModels = group.models.slice(0, PREVIEW_MODEL_COUNT)
  const hiddenModelCount = group.models.length - previewModels.length

  return (
    <Link
      to='/pricing'
      search={{ group: group.name }}
      className={cn(
        'group/card border-border/60 bg-card/60 hover:border-primary/40 focus-visible:ring-ring/50 flex min-w-0 flex-col gap-3 rounded-lg border p-4 transition-colors outline-none focus-visible:ring-3',
        health === 'idle' && 'bg-card/30',
        props.className
      )}
    >
      <div className='flex flex-wrap items-start gap-3'>
        <span className='bg-muted/50 border-border/50 flex size-9 shrink-0 items-center justify-center rounded-xl border'>
          {provider ? (
            getLobeIcon(provider.icon, 18)
          ) : (
            <span className='text-muted-foreground text-xs font-semibold'>
              {group.name.slice(0, 1).toUpperCase()}
            </span>
          )}
        </span>
        <div className='min-w-24 flex-1'>
          <div className='flex items-center gap-2'>
            <span className='truncate text-sm font-semibold' title={group.name}>
              {group.name}
            </span>
            <GroupMultiplierBadge
              ratio={group.ratio}
              className='h-[18px] min-w-0 shrink-0 px-1.5 text-[11px]'
            />
          </div>
          <p
            className='text-muted-foreground mt-0.5 line-clamp-1 text-xs'
            title={group.description}
          >
            {group.description || ' '}
          </p>
        </div>
        <span
          className={cn(
            'flex shrink-0 items-center gap-1.5 text-[11px] font-medium',
            color.text
          )}
        >
          <span className='relative flex size-2'>
            {health !== 'idle' && (
              <span
                aria-hidden
                className={cn(
                  'absolute inset-0 rounded-full animate-ping opacity-60',
                  color.dot
                )}
              />
            )}
            <span className={cn('relative size-2 rounded-full', color.dot)} />
          </span>
          {t(HEALTH_LABEL[health])}
        </span>
      </div>

      <GroupUptimeBars series={group.status?.series} />

      <dl className='grid grid-cols-2 gap-2 font-mono text-[11px] tabular-nums sm:grid-cols-4'>
        <div>
          <dt className='text-muted-foreground font-sans break-words'>
            {t('Availability')}
          </dt>
          <dd
            className={cn('mt-0.5 font-medium', group.status ? color.text : '')}
          >
            {formatGroupRate(group.status?.success_rate, locale)}
          </dd>
        </div>
        <div>
          <dt className='text-muted-foreground font-sans'>
            {t('Latency short')}
          </dt>
          <dd className='mt-0.5'>
            {formatLatency(group.status?.avg_latency_ms ?? 0)}
          </dd>
        </div>
        <div>
          <dt className='text-muted-foreground font-sans'>
            {t('Throughput short')}
          </dt>
          <dd className='mt-0.5'>
            {formatThroughput(group.status?.avg_tps ?? 0)}
          </dd>
        </div>
        <div>
          <dt className='text-muted-foreground font-sans'>
            {t('24h requests')}
          </dt>
          <dd className='mt-0.5'>
            {group.status
              ? formatCompactNumber(group.status.request_count, locale)
              : '—'}
          </dd>
        </div>
      </dl>

      <div className='text-muted-foreground flex min-w-0 flex-wrap items-center gap-1 font-mono text-[11px]'>
        {previewModels.length === 0 && <span>{t('No models')}</span>}
        {previewModels.map((modelName) => (
          <span
            key={modelName}
            className='bg-muted/60 max-w-[11rem] truncate rounded-md px-1.5 py-0.5'
            title={modelName}
          >
            {modelName}
          </span>
        ))}
        {hiddenModelCount > 0 && (
          <span className='px-1'>+{hiddenModelCount}</span>
        )}
      </div>
    </Link>
  )
}
