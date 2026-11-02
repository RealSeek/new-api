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

import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/ui/tooltip'
import { toIntlLocale } from '@/i18n/languages'
import { formatNumber } from '@/lib/format'

import type { UsageLog } from '../data/schema'
import { getTokenUsageBreakdown, parseLogOther } from '../lib/format'

type TokenUsageDisplayProps = {
  log: UsageLog
  hideEmpty?: boolean
}

export function TokenUsageDisplay(props: TokenUsageDisplayProps) {
  const { t, i18n } = useTranslation()
  const tooltipId = useId()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const other = parseLogOther(props.log.other)
  const usage = getTokenUsageBreakdown(props.log, other)
  if (!usage.hasUsage) {
    if (props.hideEmpty) return null
    return <span className='text-muted-foreground text-xs'>-</span>
  }

  const inputText = formatNumber(usage.regularInput, locale)
  const outputText = formatNumber(usage.output, locale)
  const readText = formatNumber(usage.cacheRead, locale)
  const writeText = formatNumber(usage.cacheWrite, locale)

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger
          render={
            <Button
              variant='ghost'
              aria-label={t('Token usage')}
              aria-describedby={tooltipId}
              className='h-auto min-h-8 max-w-full min-w-0 flex-col items-start gap-0.5 px-0 py-0.5 text-left text-xs font-normal whitespace-normal'
            />
          }
        >
          <span className='flex flex-wrap items-center gap-x-3 gap-y-0.5 font-mono font-medium tabular-nums'>
            <span
              aria-label={`${t('Regular Input')}: ${inputText}`}
              className='inline-flex items-center gap-1'
            >
              <span className='text-muted-foreground font-sans font-normal'>
                {t('Regular Input')}
              </span>
              {inputText}
            </span>
            <span
              aria-label={`${t('Output')}: ${outputText}`}
              className='inline-flex items-center gap-1'
            >
              <span aria-hidden='true' className='text-muted-foreground'>
                /
              </span>
              <span className='text-muted-foreground font-sans font-normal'>
                {t('Output')}
              </span>
              {outputText}
            </span>
          </span>
          {(usage.cacheRead > 0 || usage.cacheWrite > 0) && (
            <span className='flex flex-wrap items-center gap-x-3 gap-y-0.5 text-[11px]'>
              {usage.cacheRead > 0 && (
                <span className='text-muted-foreground/60 inline-flex items-center gap-1'>
                  <span>{t('Cache Read')}</span>
                  <span className='font-mono tabular-nums'>{readText}</span>
                </span>
              )}
              {usage.cacheWrite > 0 && (
                <span className='text-muted-foreground/60 inline-flex items-center gap-1'>
                  <span>{t('Cache Write')}</span>
                  <span className='font-mono tabular-nums'>{writeText}</span>
                </span>
              )}
            </span>
          )}
        </TooltipTrigger>
        <TooltipContent
          id={tooltipId}
          role='tooltip'
          side='top'
          className='flex-col items-start gap-1'
        >
          <span>
            {t('Total Input = Regular Input + Cache Read + Cache Write')}
          </span>
          <span className='font-mono tabular-nums'>
            {formatNumber(usage.totalInput, locale)} = {inputText} + {readText}{' '}
            + {writeText}
          </span>
          <span>
            {t(
              'Unreused Input = Total Input - Cache Read = Regular Input + Cache Write'
            )}
          </span>
          <span className='font-mono tabular-nums'>
            {formatNumber(usage.unreusedInput, locale)} ={' '}
            {formatNumber(usage.totalInput, locale)} - {readText} = {inputText}{' '}
            + {writeText}
          </span>
          <span>{t('Cache hits = Cache Read')}</span>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  )
}
