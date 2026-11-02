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

import { toIntlLocale } from '@/i18n/languages'
import { formatNumber } from '@/lib/format'

import type { UsageLog } from '../data/schema'
import { getTokenUsageBreakdown, parseLogOther } from '../lib/format'

type TokenUsageDisplayProps = {
  log: UsageLog
}

export function TokenUsageDisplay(props: TokenUsageDisplayProps) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const usage = getTokenUsageBreakdown(
    props.log,
    parseLogOther(props.log.other)
  )
  if (!usage.hasUsage) {
    return <span className='text-muted-foreground text-xs'>-</span>
  }

  const inputText = formatNumber(usage.regularInput, locale)
  const outputText = formatNumber(usage.output, locale)
  const readText = formatNumber(usage.cacheRead, locale)
  const writeText = formatNumber(usage.cacheWrite, locale)

  return (
    <div className='flex flex-col gap-0.5'>
      <span className='font-mono text-xs font-medium whitespace-nowrap tabular-nums'>
        <span aria-label={`${t('Regular Input')}: ${inputText}`}>
          {inputText}
        </span>{' '}
        / <span aria-label={`${t('Output')}: ${outputText}`}>{outputText}</span>
      </span>
      {(usage.cacheRead > 0 || usage.cacheWrite > 0) && (
        <div className='flex items-center gap-1 text-[11px]'>
          {usage.cacheRead > 0 && (
            <span
              className='text-muted-foreground/60'
              aria-label={`${t('Cache Read')}: ${readText}`}
            >
              {t('Cache')}↓ {readText}
            </span>
          )}
          {usage.cacheWrite > 0 && (
            <span
              className='text-muted-foreground/60'
              aria-label={`${t('Cache Write')}: ${writeText}`}
            >
              ↑ {writeText}
            </span>
          )}
        </div>
      )}
    </div>
  )
}
