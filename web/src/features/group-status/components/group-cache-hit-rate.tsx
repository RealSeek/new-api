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
import { useId } from 'react'
import { useTranslation } from 'react-i18next'

import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { toIntlLocale } from '@/i18n/languages'
import { formatNumber, formatTimestampToDate } from '@/lib/format'

import type { GroupStatusItem } from '../types'

type GroupCacheHitRateProps = {
  group: GroupStatusItem
  compact?: boolean
}

export function GroupCacheHitRate(props: GroupCacheHitRateProps) {
  const { t, i18n } = useTranslation()
  const tooltipId = useId()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const rate = props.group.cache_hit_rate
  const value = rate === null ? '—' : `${formatNumber(rate, locale)}%`

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger
          aria-label={`${t('Cache hit rate')}: ${value}`}
          aria-describedby={tooltipId}
          render={
            <span
              tabIndex={0}
              className='focus-visible:ring-ring/50 cursor-help rounded-sm outline-none focus-visible:ring-2'
            />
          }
        >
          {props.compact ? t('Cache {{rate}}', { rate: value }) : value}
        </TooltipTrigger>
        <TooltipContent
          id={tooltipId}
          role='tooltip'
          className='flex-col items-start gap-1'
        >
          <span>
            {t(
              'Cache read tokens / total input tokens in the latest 5-minute request window. When there are no new requests, the last valid window is kept.'
            )}
          </span>
          {props.group.cache_window_end > 0 && (
            <span className='font-mono tabular-nums'>
              {t('Time')}:{' '}
              {formatTimestampToDate(props.group.cache_window_end - 300)} –{' '}
              {formatTimestampToDate(props.group.cache_window_end)}
            </span>
          )}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}
