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
import { useTranslation } from 'react-i18next'

import { getSuccessRateDotClass } from '@/features/performance-metrics/lib/format'
import { toIntlLocale } from '@/i18n/languages'
import { formatNumber } from '@/lib/format'
import { cn } from '@/lib/utils'

import { formatGroupRate } from '../lib/format'
import type { GroupHourPoint } from '../types'

const IDLE_SLOTS = Array.from({ length: 24 }, (_, slot) => slot)

type GroupUptimeBarsProps = {
  /** Hourly points from the board; omit for a group without samples. */
  series?: GroupHourPoint[]
  className?: string
}

/** One bar per hour of the 24-hour window; hours without traffic stay gray. */
export function GroupUptimeBars(props: GroupUptimeBarsProps) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const hourFormatter = new Intl.DateTimeFormat(locale, {
    hour: '2-digit',
    minute: '2-digit',
  })
  const barClassName = 'h-full flex-1 rounded-[2px]'

  return (
    <div
      role='img'
      aria-label={t('Hourly availability for the last 24 hours')}
      className={cn('flex h-5 items-end gap-[2px]', props.className)}
    >
      {props.series
        ? props.series.map((point) => {
            const hour = hourFormatter.format(point.ts * 1000)
            if (point.request_count === 0) {
              return (
                <span
                  key={point.ts}
                  title={`${hour} · ${t('No traffic')}`}
                  className={cn(barClassName, 'bg-muted-foreground/15')}
                />
              )
            }
            return (
              <span
                key={point.ts}
                title={`${hour} · ${formatGroupRate(point.success_rate, locale)} · ${t('Requests')} ${formatNumber(point.request_count, locale)}`}
                className={cn(
                  barClassName,
                  'transition-opacity hover:opacity-70',
                  getSuccessRateDotClass(point.success_rate)
                )}
              />
            )
          })
        : IDLE_SLOTS.map((slot) => (
            <span
              key={slot}
              className={cn(barClassName, 'bg-muted-foreground/10')}
            />
          ))}
    </div>
  )
}
