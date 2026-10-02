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
import { X } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import type { PricingCurrency } from '@/features/model-pricing/currency'
import { combineBillingExpr } from '@/features/pricing/lib/billing-expr'
import {
  generateImageResolutionPricing,
  IMAGE_RESOLUTION_TIERS,
  parseImageResolutionPricing,
  type ImageResolutionPriceRow,
} from '@/features/pricing/lib/image-resolution-pricing'
import type { TaskMatrixRow } from '@/features/pricing/lib/task-expr'
import type { BillingUsageSchema } from '@/features/pricing/types'

import { RequestSimulation } from './request-simulation'
import { TaskPricingMatrix } from './task-pricing-matrix'

type ImageTaskPricingEditorProps = {
  currency?: PricingCurrency
  billingExpr: string
  requestRuleExpr: string
  onBillingExprChange: (next: string) => void
}

const IMAGE_USAGE_SCHEMA: BillingUsageSchema = {
  resolution: {
    enum: [],
    description: 'Resolution',
  },
  image_count: {
    type: 'number',
    unit: 'count',
    unitLabel: { en: 'image', zh: '张' },
    description: 'Price per image',
  },
}

export function ImageTaskPricingEditor(props: ImageTaskPricingEditorProps) {
  const { t } = useTranslation()
  const parsed = parseImageResolutionPricing(props.billingExpr)
  if (!parsed) return null

  const rows: TaskMatrixRow[] = parsed.rows.map((row) => ({
    combination: { resolution: row.resolution },
    constant: 0,
    unitPrices: { image_count: row.price },
  }))
  const usageSchema: BillingUsageSchema = {
    ...IMAGE_USAGE_SCHEMA,
    resolution: {
      ...IMAGE_USAGE_SCHEMA.resolution,
      enum: parsed.rows.map((row) => row.resolution),
    },
  }
  const invalidPrice = parsed.rows.some((row) => !(row.price > 0))
  const missingResolutions = IMAGE_RESOLUTION_TIERS.filter(
    (resolution) => !parsed.rows.some((row) => row.resolution === resolution)
  )

  const publish = (nextRows: ImageResolutionPriceRow[]) => {
    props.onBillingExprChange(generateImageResolutionPricing(nextRows))
  }

  return (
    <div className='space-y-5' data-billing-invalid={invalidPrice || undefined}>
      <Alert>
        <AlertDescription className='text-xs'>
          {t(
            'Set prices for each set of conditions below. Cost = usage × unit price + additional charge. Token prices are per million tokens.'
          )}
        </AlertDescription>
      </Alert>

      <div className='bg-muted/30 space-y-4 rounded-md border p-3'>
        <div className='space-y-2'>
          <p className='text-sm font-medium'>{t('Supported resolutions')}</p>
          <div className='flex flex-wrap gap-2'>
            {parsed.rows.map((row) => (
              <Badge
                key={row.resolution}
                variant='outline'
                className='gap-1 pr-0.5'
              >
                {row.resolution.toUpperCase()}
                <Button
                  type='button'
                  variant='ghost'
                  size='icon-xs'
                  aria-label={t('Remove resolution: {{resolution}}', {
                    resolution: row.resolution,
                  })}
                  disabled={parsed.rows.length === 1}
                  onClick={() =>
                    publish(
                      parsed.rows.filter(
                        (candidate) => candidate.resolution !== row.resolution
                      )
                    )
                  }
                >
                  <X aria-hidden='true' />
                </Button>
              </Badge>
            ))}
          </div>
          {missingResolutions.length > 0 && (
            <Select
              value={null}
              onValueChange={(resolution) => {
                if (!resolution) return
                publish([
                  ...parsed.rows,
                  { resolution, price: parsed.fallbackPrice },
                ])
              }}
              items={missingResolutions.map((resolution) => ({
                value: resolution,
                label: resolution.toUpperCase(),
              }))}
            >
              <SelectTrigger
                className='w-full sm:w-48'
                aria-label={t('Add resolution')}
              >
                <SelectValue placeholder={t('Add resolution')} />
              </SelectTrigger>
              <SelectContent>
                {missingResolutions.map((resolution) => (
                  <SelectItem key={resolution} value={resolution}>
                    {resolution.toUpperCase()}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          )}
        </div>

        {invalidPrice && (
          <p role='alert' className='text-destructive text-sm'>
            {t('Unit price must be greater than 0')}
          </p>
        )}

        <TaskPricingMatrix
          currency={props.currency}
          rows={rows}
          usageSchema={usageSchema}
          showAdditionalCharge={false}
          matchedRowIndex={null}
          onRowChange={(index, next) =>
            publish(
              parsed.rows.map((row, rowIndex) =>
                rowIndex === index
                  ? { ...row, price: next.unitPrices.image_count ?? 0 }
                  : row
              )
            )
          }
          onFillColumn={(_, value) =>
            publish(parsed.rows.map((row) => ({ ...row, price: value })))
          }
        />
      </div>

      <RequestSimulation
        key={parsed.rows.map((row) => row.resolution).join(':')}
        expression={combineBillingExpr(
          props.billingExpr,
          props.requestRuleExpr
        )}
        initialBody={JSON.stringify(
          { image_tier: parsed.rows[0]?.resolution ?? '1k' },
          null,
          2
        )}
        currency={props.currency}
        mode='token'
      />
    </div>
  )
}
