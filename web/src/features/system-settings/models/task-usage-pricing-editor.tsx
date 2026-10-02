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
import { AlertTriangle, X } from 'lucide-react'
import { memo, useId, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { ConfirmDialog } from '@/components/confirm-dialog'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { Combobox } from '@/components/ui/combobox'
import { Field, FieldDescription, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Switch } from '@/components/ui/switch'
import { Textarea } from '@/components/ui/textarea'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import {
  formatPricingAmount,
  USD_PRICING_CURRENCY,
  type PricingCurrency,
} from '@/features/model-pricing/currency'
import { PricingAmountInput } from '@/features/model-pricing/pricing-amount-input'
import {
  combineBillingExpr,
  splitBillingExprAndRequestRules,
} from '@/features/pricing/lib/billing-expr'
import {
  getTaskUsagePriceUnitLabelKey,
  getTaskUsageQuantityUnitLabelKey,
} from '@/features/pricing/lib/dynamic-price'
import {
  createDefaultTaskMatrixConfig,
  evaluateTaskVisualConfig,
  generateTaskExprFromConfig,
  getTaskEnumCombinations,
  getTaskEnumFields,
  getTaskNumberFields,
  taskMatrixToTiers,
  taskPricingSchema,
  tryParseTaskMatrixConfig,
  tryParseTaskVisualConfig,
  type TaskMatrixRow,
  type TaskVisualConfig,
} from '@/features/pricing/lib/task-expr'
import {
  taskPriceLabel,
  taskUsageUnitLabel,
  taskEnumLabel,
  taskPricingConditions,
} from '@/features/pricing/lib/task-price-display'
import type {
  BillingUsageExample,
  BillingUsageSchema,
} from '@/features/pricing/types'

import { formatPricingNumber } from './pricing-format'
import { RequestSimulation } from './request-simulation'
import { TaskPricingMatrix } from './task-pricing-matrix'

type TaskUsagePricingEditorProps = {
  currency?: PricingCurrency
  billingExpr: string
  requestRuleExpr: string
  usageSchema: BillingUsageSchema
  usageExamples?: BillingUsageExample[]
  onBillingExprChange: (next: string) => void
  onRequestRuleExprChange: (next: string) => void
}

type EditorMode = 'visual' | 'raw'

function taskVideoReferencePricing(
  rows: TaskMatrixRow[],
  schema: BillingUsageSchema
) {
  const field = schema.video_input
  let absent: string
  let present: string
  if (field?.type === 'boolean') {
    absent = 'false'
    present = 'true'
  } else if (
    field?.enum?.length === 2 &&
    field.enum.includes('none') &&
    field.enum.includes('video')
  ) {
    absent = 'none'
    present = 'video'
  } else {
    return null
  }
  if (!schema.resolution) return null
  const pairs = rows.flatMap((row, baseIndex) => {
    if (row.combination.video_input !== absent) return []
    const referenceIndex = rows.findIndex(
      (candidate) =>
        candidate.combination.video_input === present &&
        Object.entries(row.combination).every(
          ([key, value]) =>
            key === 'video_input' || candidate.combination[key] === value
        )
    )
    return [{ baseIndex, referenceIndex }]
  })
  // Independent reference tariffs remain editable in their original condition table.
  if (
    pairs.some(({ baseIndex, referenceIndex }) => {
      const base = rows[baseIndex]
      const reference = rows[referenceIndex]
      return (
        !reference ||
        reference.constant < base.constant ||
        getTaskNumberFields(schema).some(
          ([key]) =>
            (reference.unitPrices[key] ?? 0) !== (base.unitPrices[key] ?? 0)
        )
      )
    })
  ) {
    return null
  }
  return pairs
}

function taskVideoBillingConfig(
  rows: TaskMatrixRow[],
  schema: BillingUsageSchema,
  billingUnit: string,
  referenceCharge: boolean
) {
  const meters = getTaskNumberFields(schema).filter(
    ([, definition]) =>
      definition.unit === 'second' || definition.unit === 'token'
  )
  const selectedSchema =
    schema.resolution &&
    meters.some(([, definition]) => definition.unit === 'second') &&
    meters.some(([, definition]) => definition.unit === 'token') &&
    billingUnit !== 'mixed'
      ? Object.fromEntries(
          Object.entries(schema).filter(
            ([field]) =>
              !meters.some(([meter]) => meter === field) ||
              field === billingUnit
          )
        )
      : schema
  const nextRows = [...rows]
  const referencePairs = taskVideoReferencePricing(rows, schema)
  if (!referenceCharge) {
    for (const pair of referencePairs ?? []) {
      nextRows[pair.referenceIndex] = {
        ...rows[pair.referenceIndex],
        constant: rows[pair.baseIndex].constant,
      }
    }
  }
  const tiers = taskMatrixToTiers({ rows: nextRows }, selectedSchema)
  const lastPair = referencePairs?.at(-1)
  const fallback = tiers.at(-1)
  if (referenceCharge && lastPair && fallback && tiers.length > 1) {
    // Open resolution selectors must also respect whether a reference is present.
    fallback.conditions = [
      {
        field: 'video_input',
        value: rows[lastPair.referenceIndex].combination.video_input,
      },
    ]
    tiers.push({
      ...fallback,
      label: 'base_without_reference',
      conditions: [],
      constant: rows[lastPair.baseIndex].constant,
      unitPrices: { ...fallback.unitPrices },
    })
  }
  return { schema: selectedSchema, tiers }
}

type TaskBillingPreviewProps = {
  currency?: PricingCurrency
  config: TaskVisualConfig | null
  requestRuleExpr: string
  sample: Record<string, number | string>
  usageSchema: BillingUsageSchema
  usageExamples?: BillingUsageExample[]
  onSampleChange: (field: string, value: number | string) => void
  onSampleReplace: (sample: Record<string, number | string>) => void
}

function TaskBillingPreview(props: TaskBillingPreviewProps) {
  const { t, i18n } = useTranslation()
  const enumFields = getTaskEnumFields(props.usageSchema)
  const numberFields = getTaskNumberFields(props.usageSchema)
  const result = props.config
    ? evaluateTaskVisualConfig(props.config, props.sample, props.usageSchema)
    : null

  const formulaParts = (result?.parts ?? []).map((part) => {
    if (part.kind === 'constant') {
      return `${t('Additional charge')}: ${formatPricingAmount(part.amount, props.currency)}`
    }

    const definition = props.usageSchema[part.field ?? '']
    const quantityUnitKey = getTaskUsageQuantityUnitLabelKey(definition?.unit)
    const priceUnitKey = getTaskUsagePriceUnitLabelKey(definition?.unit)
    const quantityUnitLabel = taskUsageUnitLabel(
      definition,
      i18n.language,
      t(quantityUnitKey)
    )
    const priceUnitLabel = taskUsageUnitLabel(
      definition,
      i18n.language,
      t(priceUnitKey)
    )
    const quantityLabel =
      definition?.unit === 'second'
        ? `${formatPricingNumber(part.quantity)}${quantityUnitLabel}`
        : `${formatPricingNumber(part.quantity)} ${quantityUnitLabel}`
    return `${taskPriceLabel(definition?.description, part.field ?? '', i18n.language)}: ${quantityLabel} × ${formatPricingAmount(part.unitPrice ?? 0, props.currency)}/${priceUnitLabel}`
  })
  const formulaLeft =
    formulaParts.length > 0
      ? formulaParts.join(' + ')
      : formatPricingAmount(0, props.currency)
  const formula = `${formulaLeft} = ${formatPricingAmount(result?.total ?? 0, props.currency)}`

  return (
    <div className='bg-muted/30 flex flex-col gap-3 rounded-md border p-3'>
      <div className='flex flex-col gap-1'>
        <h4 className='text-sm font-medium'>{t('Cost calculator')}</h4>
        <p className='text-muted-foreground text-xs'>
          {t(
            'Enter sample usage to estimate the cost. Group and request multipliers are not included.'
          )}
          {props.requestRuleExpr ? (
            <> {t('Request rules apply on top of this amount.')}</>
          ) : null}
        </p>
      </div>
      {props.usageExamples && props.usageExamples.length > 0 ? (
        <Field className='gap-1.5'>
          <FieldLabel>{t('Example spec')}</FieldLabel>
          <Combobox
            options={props.usageExamples.map((example) => ({
              value: example.label,
              label: example.label,
            }))}
            value={
              props.usageExamples.find((example) =>
                Object.entries(example.facts).every(
                  ([field, value]) => props.sample[field] === value
                )
              )?.label ?? null
            }
            onValueChange={(label) => {
              const example = props.usageExamples?.find(
                (item) => item.label === label
              )
              if (example) props.onSampleReplace({ ...example.facts })
            }}
            className='w-full'
            placeholder={t('Example spec')}
          />
        </Field>
      ) : null}
      {enumFields.length + numberFields.length > 0 ? (
        <div className='grid gap-3 sm:grid-cols-2'>
          {enumFields.map(([field, definition]) => {
            const items = (definition.enum ?? []).map((value) => {
              let label = taskEnumLabel(definition, value, i18n.language)
              if (definition.type === 'boolean') {
                label = value === 'true' ? t('Yes') : t('No')
              }
              return { value, label }
            })
            return (
              <Field key={field} className='gap-1.5'>
                <FieldLabel>
                  {taskPriceLabel(definition.description, field, i18n.language)}
                </FieldLabel>
                <Combobox
                  aria-label={taskPriceLabel(
                    definition.description,
                    field,
                    i18n.language
                  )}
                  options={items}
                  value={String(props.sample[field] ?? '')}
                  onValueChange={(value) =>
                    value !== null && props.onSampleChange(field, value)
                  }
                  className='w-full'
                />
              </Field>
            )
          })}
          {numberFields.map(([field, definition]) => (
            <Field key={field} className='gap-1.5'>
              <FieldLabel>
                {t('Usage · {{price}}', {
                  price: taskPriceLabel(
                    definition.description,
                    field,
                    i18n.language
                  ),
                })}
              </FieldLabel>
              <div className='flex items-center gap-2'>
                <Input
                  type='number'
                  aria-label={t('Usage · {{price}}', {
                    price: taskPriceLabel(
                      definition.description,
                      field,
                      i18n.language
                    ),
                  })}
                  min={0}
                  step={1}
                  value={props.sample[field] ?? 0}
                  onChange={(event) => {
                    const value = Number(event.target.value)
                    props.onSampleChange(
                      field,
                      Number.isFinite(value) && value >= 0 ? value : 0
                    )
                  }}
                  className='font-mono'
                />
                <span className='text-muted-foreground shrink-0 text-xs'>
                  {taskUsageUnitLabel(
                    definition,
                    i18n.language,
                    t(getTaskUsageQuantityUnitLabelKey(definition.unit))
                  )}
                </span>
              </div>
            </Field>
          ))}
        </div>
      ) : null}
      {result ? (
        <div className='border-primary/50 bg-primary/10 flex flex-col gap-2 rounded-md border p-3 text-sm'>
          <Badge variant='outline' className='text-xs'>
            {t('Current pricing conditions')}:{' '}
            {taskPricingConditions(
              enumFields.map(([field]) => ({
                field,
                value: String(props.sample[field] ?? ''),
              })),
              props.usageSchema,
              i18n.language,
              t
            ) || t('All requests')}
          </Badge>
          <code className='font-mono text-xs break-words'>{formula}</code>
        </div>
      ) : (
        <p className='text-muted-foreground text-xs'>
          {t('Preview is unavailable for custom expressions.')}
        </p>
      )}
    </div>
  )
}

export const TaskUsagePricingEditor = memo(function TaskUsagePricingEditor(
  props: TaskUsagePricingEditorProps
) {
  const { t, i18n } = useTranslation()
  const referenceChargeId = useId()
  const [usageSchema, setUsageSchema] = useState(() =>
    taskPricingSchema(props.usageSchema, props.billingExpr)
  )
  const [customResolution, setCustomResolution] = useState('')
  const [editorMode, setEditorMode] = useState<EditorMode>(() =>
    props.billingExpr &&
    !tryParseTaskMatrixConfig(props.billingExpr, usageSchema)
      ? 'raw'
      : 'visual'
  )
  const [matrixRows, setMatrixRows] = useState<TaskMatrixRow[]>(() => {
    const parsed = tryParseTaskMatrixConfig(props.billingExpr, usageSchema)
    return (parsed ?? createDefaultTaskMatrixConfig(usageSchema)).rows
  })
  const [confirmVisualSwitch, setConfirmVisualSwitch] = useState(false)
  const videoMeters = getTaskNumberFields(usageSchema).filter(
    ([, definition]) =>
      definition.unit === 'second' || definition.unit === 'token'
  )
  const hasVideoUnits =
    Boolean(usageSchema.resolution) &&
    videoMeters.some(([, definition]) => definition.unit === 'second') &&
    videoMeters.some(([, definition]) => definition.unit === 'token')
  const [billingUnit, setBillingUnit] = useState(() => {
    const priced = videoMeters.filter(([field]) =>
      matrixRows.some((row) => row.unitPrices[field] > 0)
    )
    return priced.length > 1
      ? 'mixed'
      : (priced[0]?.[0] ??
          videoMeters.find(
            ([, definition]) => definition.unit === 'second'
          )?.[0] ??
          '')
  })
  const referencePairs = taskVideoReferencePricing(matrixRows, usageSchema)
  const [referenceCharge, setReferenceCharge] = useState(() =>
    (referencePairs ?? []).some(
      ({ baseIndex, referenceIndex }) =>
        matrixRows[referenceIndex].constant > matrixRows[baseIndex].constant
    )
  )
  const billingConfig = taskVideoBillingConfig(
    matrixRows,
    usageSchema,
    billingUnit,
    referenceCharge
  )
  const displaySchema = billingConfig.schema
  // The surcharge is an editing column, not a provider usage field.
  const matrixSchema = referencePairs
    ? Object.fromEntries([
        ...Object.entries(displaySchema).filter(
          ([key]) => key !== 'video_input'
        ),
        ...(referenceCharge
          ? [
              [
                '__reference_surcharge',
                {
                  type: 'number',
                  unit: 'count',
                  unitLabel: t('request'),
                  description: t('Reference video surcharge'),
                },
              ] as const,
            ]
          : []),
      ])
    : displaySchema
  const displayedRows = referencePairs
    ? referencePairs.map(({ baseIndex, referenceIndex }) => ({
        ...matrixRows[baseIndex],
        combination: Object.fromEntries(
          Object.entries(matrixRows[baseIndex].combination).filter(
            ([key]) => key !== 'video_input'
          )
        ),
        unitPrices: {
          ...matrixRows[baseIndex].unitPrices,
          __reference_surcharge:
            matrixRows[referenceIndex].constant -
            matrixRows[baseIndex].constant,
        },
      }))
    : matrixRows
  const [rawExpr, setRawExpr] = useState(() =>
    combineBillingExpr(props.billingExpr, props.requestRuleExpr)
  )
  const [previewSample, setPreviewSample] = useState<
    Record<string, number | string>
  >(() => {
    if (props.usageExamples?.[0]) {
      return { ...props.usageExamples[0].facts }
    }
    const sample: Record<string, number | string> = {}
    for (const [field, definition] of getTaskEnumFields(usageSchema)) {
      sample[field] = definition.enum?.[0] ?? ''
    }
    for (const [field, definition] of getTaskNumberFields(usageSchema)) {
      sample[field] = definition.unit === 'second' ? 5 : 1
    }
    return sample
  })
  const enumFields = getTaskEnumFields(usageSchema)
  const numberFields = getTaskNumberFields(displaySchema)
  const combinations = getTaskEnumCombinations(matrixSchema)
  const visualTiers = billingConfig.tiers

  let previewConfig: TaskVisualConfig | null = null
  let previewRequestRuleExpr = props.requestRuleExpr
  let matchedRowIndex: number | null = null
  if (editorMode === 'visual') {
    const generatedExpression = generateTaskExprFromConfig(
      { tiers: visualTiers },
      displaySchema
    )
    if (generatedExpression) previewConfig = { tiers: visualTiers }
    const nextMatchedRowIndex = combinations.findIndex((combination) =>
      Object.entries(combination).every(
        ([field, value]) => String(previewSample[field]) === value
      )
    )
    if (nextMatchedRowIndex >= 0) {
      matchedRowIndex = nextMatchedRowIndex
    }
  } else {
    const split = splitBillingExprAndRequestRules(rawExpr)
    previewConfig = tryParseTaskVisualConfig(split.billingExpr, usageSchema)
    previewRequestRuleExpr = split.requestRuleExpr
  }

  const publishRows = (
    nextRows: TaskMatrixRow[],
    nextSchema = usageSchema,
    nextUnit = billingUnit,
    nextReferenceCharge = referenceCharge
  ) => {
    setMatrixRows(nextRows)
    const config = taskVideoBillingConfig(
      nextRows,
      nextSchema,
      nextUnit,
      nextReferenceCharge
    )
    props.onBillingExprChange(
      generateTaskExprFromConfig({ tiers: config.tiers }, config.schema)
    )
  }

  const handleRowChange = (index: number, next: TaskMatrixRow) => {
    const nextRows = [...matrixRows]
    const pair = referencePairs?.[index]
    if (pair) {
      const unitPrices = Object.fromEntries(
        Object.entries(next.unitPrices).filter(
          ([key]) => key !== '__reference_surcharge'
        )
      )
      nextRows[pair.baseIndex] = {
        ...matrixRows[pair.baseIndex],
        constant: next.constant,
        unitPrices,
      }
      nextRows[pair.referenceIndex] = {
        ...matrixRows[pair.referenceIndex],
        constant: next.constant + next.unitPrices.__reference_surcharge,
        unitPrices,
      }
    } else {
      nextRows[index] = next
    }
    publishRows(nextRows)
  }

  const handleFillColumn = (priceKey: string, value: number) => {
    const nextRows = matrixRows.map((row) => {
      if (
        referencePairs &&
        (priceKey === 'constant' || priceKey === '__reference_surcharge')
      ) {
        return row
      }
      if (priceKey === 'constant') return { ...row, constant: value }
      return {
        ...row,
        unitPrices: { ...row.unitPrices, [priceKey]: value },
      }
    })
    for (const { baseIndex, referenceIndex } of referencePairs ?? []) {
      if (priceKey === 'constant') {
        const extra =
          matrixRows[referenceIndex].constant - matrixRows[baseIndex].constant
        nextRows[baseIndex] = { ...nextRows[baseIndex], constant: value }
        nextRows[referenceIndex] = {
          ...nextRows[referenceIndex],
          constant: value + extra,
        }
      } else if (priceKey === '__reference_surcharge') {
        nextRows[referenceIndex] = {
          ...nextRows[referenceIndex],
          constant: nextRows[baseIndex].constant + value,
        }
      }
    }
    publishRows(nextRows)
  }

  const addResolution = () => {
    const value = customResolution.trim().toLowerCase()
    if (!value || usageSchema.resolution.enum?.includes(value)) return
    const nextSchema = {
      ...usageSchema,
      resolution: {
        ...usageSchema.resolution,
        enum: [...(usageSchema.resolution.enum ?? []), value],
      },
    }
    const nextRows = createDefaultTaskMatrixConfig(nextSchema).rows.map(
      (row) =>
        matrixRows.find((current) =>
          Object.entries(row.combination).every(
            ([field, option]) => current.combination[field] === option
          )
        ) ?? row
    )
    setUsageSchema(nextSchema)
    setCustomResolution('')
    publishRows(nextRows, nextSchema)
  }

  const handleRawChange = (value: string) => {
    setRawExpr(value)
    const split = splitBillingExprAndRequestRules(value)
    props.onBillingExprChange(split.billingExpr)
    props.onRequestRuleExprChange(split.requestRuleExpr)
  }

  const rawSplit = splitBillingExprAndRequestRules(rawExpr)
  const rawMatrix = tryParseTaskMatrixConfig(
    rawSplit.billingExpr,
    props.usageSchema
  )

  const handleModeChange = (nextMode: EditorMode) => {
    if (nextMode === editorMode) return
    if (nextMode === 'visual') {
      setConfirmVisualSwitch(true)
      return
    }
    setRawExpr(combineBillingExpr(props.billingExpr, props.requestRuleExpr))
    setEditorMode('raw')
  }

  const handleConfirmVisualSwitch = () => {
    const nextSchema = taskPricingSchema(
      props.usageSchema,
      rawSplit.billingExpr
    )
    const nextRows = (rawMatrix ?? createDefaultTaskMatrixConfig(nextSchema))
      .rows
    setUsageSchema(nextSchema)
    setMatrixRows(nextRows)
    const pricedMeters = videoMeters.filter(([field]) =>
      nextRows.some((row) => row.unitPrices[field] > 0)
    )
    const nextUnit =
      pricedMeters.length > 1
        ? 'mixed'
        : (pricedMeters[0]?.[0] ??
          videoMeters.find(
            ([, definition]) => definition.unit === 'second'
          )?.[0] ??
          '')
    setBillingUnit(nextUnit)
    const nextReferenceCharge = (
      taskVideoReferencePricing(nextRows, nextSchema) ?? []
    ).some(
      ({ baseIndex, referenceIndex }) =>
        nextRows[referenceIndex].constant > nextRows[baseIndex].constant
    )
    setReferenceCharge(nextReferenceCharge)
    const config = taskVideoBillingConfig(
      nextRows,
      nextSchema,
      nextUnit,
      nextReferenceCharge
    )
    props.onBillingExprChange(
      generateTaskExprFromConfig({ tiers: config.tiers }, config.schema)
    )
    props.onRequestRuleExprChange(rawMatrix ? rawSplit.requestRuleExpr : '')
    setConfirmVisualSwitch(false)
    setEditorMode('visual')
  }

  const handlePreviewSampleChange = (field: string, value: number | string) => {
    setPreviewSample((current) => ({ ...current, [field]: value }))
  }

  const allRowsFree = matrixRows.every(
    (row) =>
      row.constant === 0 &&
      numberFields.every(([field]) => !(row.unitPrices[field] > 0))
  )

  return (
    <div className='space-y-5'>
      <ConfirmDialog
        open={confirmVisualSwitch}
        onOpenChange={setConfirmVisualSwitch}
        title={t('Switch to visual pricing?')}
        desc={
          rawMatrix
            ? t(
                'Switching regenerates the expression from the price table and replaces its original formatting and tier names. Changes apply only after saving.'
              )
            : t(
                'This expression cannot be represented by the price table. Switching discards the entire expression, including request rules, and resets all prices to zero. Configure prices before saving.'
              )
        }
        confirmText={t('Switch to visual editor')}
        destructive={!rawMatrix}
        handleConfirm={handleConfirmVisualSwitch}
      />
      <div className='grid gap-3 sm:grid-cols-[minmax(0,1fr)_auto] sm:items-end'>
        <Field className='gap-2'>
          <FieldLabel>{t('Editor mode')}</FieldLabel>
          <Select
            items={[
              { value: 'visual', label: t('Visual editor') },
              { value: 'raw', label: t('Expression editor') },
            ]}
            value={editorMode}
            onValueChange={(value) =>
              value !== null && handleModeChange(value as EditorMode)
            }
          >
            <SelectTrigger
              aria-label={t('Editor mode')}
              className='w-full sm:w-56'
              size='sm'
            >
              <SelectValue />
            </SelectTrigger>
            <SelectContent alignItemWithTrigger={false}>
              <SelectGroup>
                <SelectItem value='visual'>{t('Visual editor')}</SelectItem>
                <SelectItem value='raw'>{t('Expression editor')}</SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </Field>
      </div>

      <Alert>
        <AlertDescription className='text-xs'>
          {t(
            'Prices are in {{currency}}, with units shown below. Use USD when editing expressions directly.',
            { currency: (props.currency ?? USD_PRICING_CURRENCY).label }
          )}
        </AlertDescription>
      </Alert>

      <div className='bg-muted/30 space-y-3 rounded-md border p-3'>
        {editorMode === 'visual' ? (
          <>
            {hasVideoUnits ? (
              <Field className='gap-2'>
                <FieldLabel>{t('Billing unit')}</FieldLabel>
                <ToggleGroup
                  value={[billingUnit]}
                  variant='outline'
                  aria-label={t('Billing unit')}
                  onValueChange={(values) => {
                    const value = values[0]
                    if (!value || value === billingUnit) return
                    setBillingUnit(value)
                    publishRows(matrixRows, usageSchema, value)
                  }}
                >
                  {videoMeters.map(([field, definition]) => (
                    <ToggleGroupItem key={field} value={field}>
                      {definition.unit === 'second'
                        ? t('Per-second')
                        : t('Per-token')}
                    </ToggleGroupItem>
                  ))}
                  {billingUnit === 'mixed' ? (
                    <ToggleGroupItem value='mixed'>
                      {t('Usage prices')}
                    </ToggleGroupItem>
                  ) : null}
                </ToggleGroup>
                <FieldDescription>
                  {t(
                    'Switching units retains editing prices. Saving uses only the selected billing unit.'
                  )}
                </FieldDescription>
              </Field>
            ) : null}
            {referencePairs ? (
              <Field className='gap-2'>
                <div className='flex items-center gap-2'>
                  <Switch
                    id={referenceChargeId}
                    aria-label={t('Charge extra for reference video')}
                    checked={referenceCharge}
                    onCheckedChange={(checked) => {
                      setReferenceCharge(checked)
                      publishRows(matrixRows, usageSchema, billingUnit, checked)
                    }}
                  />
                  <FieldLabel htmlFor={referenceChargeId}>
                    {t('Charge extra for reference video')}
                  </FieldLabel>
                </div>
                <FieldDescription>
                  {t(
                    'When a reference video is present, add the fixed fee for its resolution once per request.'
                  )}
                </FieldDescription>
              </Field>
            ) : null}
            {usageSchema.resolution?.allowCustomValues ? (
              <Field className='gap-2'>
                <FieldLabel>{t('Supported resolutions')}</FieldLabel>
                <div className='flex flex-wrap gap-2'>
                  {usageSchema.resolution.enum?.map((resolution) => (
                    <Badge
                      key={resolution}
                      variant='outline'
                      className='gap-1 pr-0.5'
                    >
                      {resolution.toUpperCase()}
                      <Button
                        type='button'
                        variant='ghost'
                        size='icon-xs'
                        aria-label={t('Remove resolution: {{resolution}}', {
                          resolution,
                        })}
                        disabled={usageSchema.resolution.enum?.length === 1}
                        onClick={() => {
                          const nextSchema = {
                            ...usageSchema,
                            resolution: {
                              ...usageSchema.resolution,
                              enum: usageSchema.resolution.enum?.filter(
                                (value) => value !== resolution
                              ),
                            },
                          }
                          setUsageSchema(nextSchema)
                          if (previewSample.resolution === resolution) {
                            setPreviewSample({
                              ...previewSample,
                              resolution: nextSchema.resolution.enum?.[0] ?? '',
                            })
                          }
                          publishRows(
                            matrixRows.filter(
                              (row) => row.combination.resolution !== resolution
                            ),
                            nextSchema
                          )
                        }}
                      >
                        <X aria-hidden='true' />
                      </Button>
                    </Badge>
                  ))}
                </div>
                <FieldLabel>{t('Custom resolution')}</FieldLabel>
                <div className='flex items-center gap-2'>
                  <Input
                    aria-label={t('Custom resolution')}
                    value={customResolution}
                    placeholder='1440p'
                    onChange={(event) =>
                      setCustomResolution(event.target.value)
                    }
                    onKeyDown={(event) => {
                      if (event.key === 'Enter') {
                        event.preventDefault()
                        addResolution()
                      }
                    }}
                  />
                  <Button
                    type='button'
                    variant='outline'
                    disabled={
                      !customResolution.trim() ||
                      usageSchema.resolution.enum?.includes(
                        customResolution.trim().toLowerCase()
                      )
                    }
                    onClick={addResolution}
                  >
                    {t('Add resolution')}
                  </Button>
                </div>
              </Field>
            ) : null}
            {enumFields.length > 0 ? (
              <div className='flex flex-col gap-3'>
                <p className='text-muted-foreground text-xs'>
                  {t(
                    'Set prices for each set of conditions below. Cost = usage × unit price + additional charge. Token prices are per million tokens.'
                  )}
                </p>
                <TaskPricingMatrix
                  currency={props.currency}
                  rows={displayedRows}
                  usageSchema={matrixSchema}
                  matchedRowIndex={matchedRowIndex}
                  onRowChange={handleRowChange}
                  onFillColumn={handleFillColumn}
                />
              </div>
            ) : (
              <>
                {allRowsFree ? (
                  <Alert className='border-amber-500/40 bg-amber-500/10 text-amber-800 dark:text-amber-200'>
                    <AlertTriangle aria-hidden='true' />
                    <AlertDescription className='text-xs text-current'>
                      {t(
                        'All combinations are priced at zero. Matching requests will be billed as free.'
                      )}
                    </AlertDescription>
                  </Alert>
                ) : null}
                <div className='flex flex-col gap-4'>
                  <div className='flex flex-col gap-2'>
                    <Label className='text-xs font-medium'>
                      {t('Usage prices')}
                    </Label>
                    <div className='grid gap-3 sm:grid-cols-2'>
                      {numberFields.map(([field, definition]) => {
                        const description = taskPriceLabel(
                          definition.description,
                          field,
                          i18n.language
                        )
                        return (
                          <Field key={field} className='gap-1.5'>
                            <FieldLabel>{description}</FieldLabel>
                            <div className='flex items-center gap-2'>
                              <PricingAmountInput
                                currency={props.currency}
                                aria-label={field}
                                min={0}
                                step={0.000001}
                                value={matrixRows[0].unitPrices[field] ?? 0}
                                onFocus={(event) => {
                                  if (Number(event.currentTarget.value) === 0) {
                                    event.currentTarget.select()
                                  }
                                }}
                                onChange={(usd) => {
                                  const value = Number(usd)
                                  handleRowChange(0, {
                                    ...matrixRows[0],
                                    unitPrices: {
                                      ...matrixRows[0].unitPrices,
                                      [field]:
                                        Number.isFinite(value) && value >= 0
                                          ? value
                                          : 0,
                                    },
                                  })
                                }}
                                className='font-mono'
                              />
                              <span className='text-muted-foreground shrink-0 text-xs'>
                                {
                                  (props.currency ?? USD_PRICING_CURRENCY)
                                    .symbol
                                }
                                /
                                {taskUsageUnitLabel(
                                  definition,
                                  i18n.language,
                                  t(
                                    getTaskUsagePriceUnitLabelKey(
                                      definition.unit
                                    )
                                  )
                                )}
                              </span>
                            </div>
                          </Field>
                        )
                      })}
                      <Field className='gap-1.5'>
                        <FieldLabel>{t('Additional charge')}</FieldLabel>
                        <div className='flex items-center gap-2'>
                          <PricingAmountInput
                            currency={props.currency}
                            aria-label={t('Additional charge')}
                            min={0}
                            step={0.000001}
                            value={matrixRows[0].constant}
                            onFocus={(event) => {
                              if (Number(event.currentTarget.value) === 0) {
                                event.currentTarget.select()
                              }
                            }}
                            onChange={(usd) => {
                              const value = Number(usd)
                              handleRowChange(0, {
                                ...matrixRows[0],
                                constant:
                                  Number.isFinite(value) && value >= 0
                                    ? value
                                    : 0,
                              })
                            }}
                            className='font-mono'
                          />
                          <span className='text-muted-foreground shrink-0 text-xs'>
                            {(props.currency ?? USD_PRICING_CURRENCY).symbol}/
                            {t('request')}
                          </span>
                        </div>
                      </Field>
                    </div>
                  </div>
                </div>
              </>
            )}

            <TaskBillingPreview
              currency={props.currency}
              config={previewConfig}
              requestRuleExpr={previewRequestRuleExpr}
              sample={previewSample}
              usageSchema={displaySchema}
              usageExamples={props.usageExamples}
              onSampleChange={handlePreviewSampleChange}
              onSampleReplace={setPreviewSample}
            />

            <Field className='gap-2 border-t pt-3'>
              <FieldLabel>{t('Request rule pricing')}</FieldLabel>
              <Textarea
                value={props.requestRuleExpr}
                onChange={(event) =>
                  props.onRequestRuleExprChange(event.target.value)
                }
                placeholder='(header("x-priority") == "high" ? 2 : 1)'
                rows={3}
                className='font-mono text-xs'
                spellCheck={false}
              />
              <FieldDescription>
                {t(
                  'Optional request-rule multiplier expression. Leave empty when no request rule applies.'
                )}
              </FieldDescription>
            </Field>
          </>
        ) : (
          <div className='space-y-3'>
            <Alert>
              <AlertDescription className='space-y-1 text-xs'>
                <div>
                  {t('Usage parameters')}:{' '}
                  {Object.keys(usageSchema)
                    .sort((left, right) => left.localeCompare(right))
                    .map((field) => `u(${JSON.stringify(field)})`)
                    .join(', ')}
                </div>
                <div>
                  {t('Functions')}: <code>tier(name, value)</code>,{' '}
                  <code>header(name)</code>, <code>param(path)</code>
                </div>
                {!rawMatrix ? (
                  <div>
                    {t(
                      'This expression cannot be represented by the price table. You can switch to visual editing by confirming that the expression will be discarded.'
                    )}
                  </div>
                ) : null}
              </AlertDescription>
            </Alert>
            <Textarea
              aria-label={t('Billing expression')}
              value={rawExpr}
              onChange={(event) => handleRawChange(event.target.value)}
              placeholder='tier("base", u("seconds") * 0.4)'
              rows={7}
              className='font-mono text-xs'
              spellCheck={false}
            />
            <TaskBillingPreview
              currency={props.currency}
              config={previewConfig}
              requestRuleExpr={previewRequestRuleExpr}
              sample={previewSample}
              usageSchema={usageSchema}
              usageExamples={props.usageExamples}
              onSampleChange={handlePreviewSampleChange}
              onSampleReplace={setPreviewSample}
            />
          </div>
        )}
      </div>
      <RequestSimulation
        expression={
          editorMode === 'raw'
            ? rawExpr
            : combineBillingExpr(
                generateTaskExprFromConfig(
                  { tiers: visualTiers },
                  displaySchema
                ),
                props.requestRuleExpr
              )
        }
        usage={Object.fromEntries(
          Object.entries(previewSample).map(([field, value]) => [
            field,
            usageSchema[field]?.type === 'boolean'
              ? String(value) === 'true'
              : value,
          ])
        )}
        usageSchema={usageSchema}
        currency={props.currency}
        mode='task'
      />
    </div>
  )
})
