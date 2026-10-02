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
import {
  resolveLocalizedText,
  type LocalizedTextValue,
} from '@/lib/localized-text'

import type {
  BillingUsageFieldSchema,
  BillingUsageSchema,
  PricingModel,
} from '../types'
import {
  splitBillingExprAndRequestRules,
  type ParsedTaskTier,
  type TaskTierCondition,
} from './billing-expr'
import { formatBillingCondition } from './billing-expression/condition-display'
import { compileBillingExpression } from './billing-expression/parser'
import {
  visitExpression,
  type ExpressionNode,
} from './billing-expression/types'
import { withPluginPricing } from './plugin-pricing'
import {
  getTaskMatrixDisplayTiers,
  getTaskPricingDisplayTiers,
} from './task-matrix-display'

/** Compact resolution rows share the expression prices used by the detail table. */
export function getTaskResolutionPriceRows(
  model: PricingModel
): (ParsedTaskTier & { resolution: string })[] | null {
  if (model.billing_plugin_variants?.length) {
    if (model.billing_plugin_variants.length !== 1) return null
    return getTaskResolutionPriceRows(
      withPluginPricing(model, model.billing_plugin_variants[0])
    )
  }
  const schema = model.billing_usage_schema
  if (
    model.billing_mode !== 'tiered_expr' ||
    !schema?.resolution ||
    !model.billing_expr
  ) {
    return null
  }
  const matrix = getTaskMatrixDisplayTiers(model.billing_expr, schema)
  if (!matrix) return null
  // Schema meters not read by the saved expression must not appear as free prices.
  const fields = [
    ...new Set(
      getTaskPricingDisplayTiers(model.billing_expr, schema).flatMap((tier) =>
        Object.keys(tier.unitPrices)
      )
    ),
  ]
  const resolutions = new Map<string, ParsedTaskTier[]>()
  for (const tier of matrix) {
    const resolution = tier.conditions.find(
      (condition) => condition.field === 'resolution'
    )?.value
    if (resolution === undefined) return null
    const rows = resolutions.get(resolution) ?? []
    rows.push(tier)
    resolutions.set(resolution, rows)
  }
  return [...resolutions].flatMap(([resolution, rows]) => {
    const first = rows[0]
    const uniform = rows.every(
      (row) =>
        row.constant === first.constant &&
        fields.every(
          (field) => row.unitPrices[field] === first.unitPrices[field]
        )
    )
    return (uniform ? [first] : rows).map((row) => ({
      ...row,
      resolution,
      conditions: uniform
        ? []
        : row.conditions.filter(
            (condition) => condition.field !== 'resolution'
          ),
      unitPrices: Object.fromEntries(
        fields.map((field) => [field, row.unitPrices[field]])
      ),
    }))
  })
}

export type ImageResolutionPriceRow = {
  resolution: string
  price: number
}

function readFixedTierPrice(node: ExpressionNode): number | null {
  if (
    node.kind !== 'call' ||
    node.name !== 'tier' ||
    node.args[1]?.kind !== 'call' ||
    node.args[1].name !== 'fixed' ||
    node.args[1].args[0]?.kind !== 'literal' ||
    typeof node.args[1].args[0].value !== 'number'
  ) {
    return null
  }
  return node.args[1].args[0].value
}

/** Reads the canonical image-resolution expression produced by legacy migration. */
export function getImageResolutionPriceRows(
  model: PricingModel
): ImageResolutionPriceRow[] | null {
  if (
    model.billing_mode !== 'tiered_expr' ||
    model.billing_usage_schema ||
    !model.billing_expr
  ) {
    return null
  }
  const split = splitBillingExprAndRequestRules(model.billing_expr)
  if (split.requestRuleExpr?.trim()) return null
  const compiled = compileBillingExpression(split.billingExpr)
  if (compiled.status !== 'ready') return null

  let pricing = compiled.ast
  if (pricing.kind !== 'binary' || pricing.operator !== '*') return null
  if (
    pricing.left.kind === 'variable' &&
    pricing.left.name === 'image_count'
  ) {
    pricing = pricing.right
  } else if (
    pricing.right.kind === 'variable' &&
    pricing.right.name === 'image_count'
  ) {
    pricing = pricing.left
  } else {
    return null
  }

  const rows: ImageResolutionPriceRow[] = []
  let branch = pricing
  while (branch.kind === 'conditional') {
    const condition = branch.condition
    if (
      condition.kind !== 'binary' ||
      condition.operator !== '==' ||
      condition.left.kind !== 'call' ||
      condition.left.name !== 'param' ||
      condition.left.args[0]?.kind !== 'literal' ||
      condition.left.args[0].value !== 'image_tier' ||
      condition.right.kind !== 'literal' ||
      typeof condition.right.value !== 'string' ||
      !['1k', '2k', '4k'].includes(condition.right.value)
    ) {
      return null
    }
    const price = readFixedTierPrice(branch.yes)
    if (price === null || price < 0) return null
    rows.push({ resolution: condition.right.value, price })
    branch = branch.no
  }
  if (readFixedTierPrice(branch) === null || rows.length === 0) return null
  return rows.sort(
    (left, right) =>
      Number.parseInt(left.resolution) - Number.parseInt(right.resolution)
  )
}

export function taskPriceLabel(
  description: LocalizedTextValue | undefined,
  field: string,
  language: string
): string {
  const localized =
    typeof description === 'object' && description
      ? { ...description, en: description.en?.trim() || field }
      : description
  return resolveLocalizedText(localized, language) || field
}

export function taskEnumLabel(
  definition: BillingUsageFieldSchema | undefined,
  value: string,
  language: string
): string {
  return taskPriceLabel(definition?.enumLabels?.[value], value, language)
}

export function taskUsageUnitLabel(
  definition: { unit?: string; unitLabel?: LocalizedTextValue } | undefined,
  language: string,
  fallback: string
): string {
  if (definition?.unit !== 'count') return fallback
  return resolveLocalizedText(definition.unitLabel, language) || fallback
}

export function taskPricingConditions(
  conditions: TaskTierCondition[],
  schema: BillingUsageSchema | undefined,
  language: string,
  t: (key: string) => string
): string {
  return conditions
    .map(({ field, value }) => {
      const definition = schema?.[field]
      const label = taskPriceLabel(definition?.description, field, language)
      if (definition?.type === 'boolean') {
        return `${label}: ${value === 'true' ? t('Yes') : t('No')}`
      }
      const optionLabel = taskEnumLabel(definition, value, language)
      return optionLabel !== value ? optionLabel : `${label}: ${optionLabel}`
    })
    .join(' · ')
}

export function hasSimpleTaskPricing(model: PricingModel): boolean {
  if (
    !model.billing_usage_schema ||
    model.billing_mode !== 'tiered_expr' ||
    !model.billing_expr
  ) {
    return false
  }
  const split = splitBillingExprAndRequestRules(model.billing_expr)
  if (split.requestRuleExpr?.trim()) return false
  const tiers = getTaskPricingDisplayTiers(
    split.billingExpr,
    model.billing_usage_schema
  )
  return tiers.length === 1 && !tiers[0].conditionText
}

export function taskTierConditions(
  tier: ParsedTaskTier,
  schema: BillingUsageSchema | undefined,
  language: string,
  t: (key: string) => string
): string {
  const usage = taskPricingConditions(tier.conditions, schema, language, t)
  const time = tier.conditionText
    ? (formatBillingCondition(tier.conditionText, t, language) ??
      tier.conditionText)
    : ''
  return [usage, time].filter(Boolean).join(' · ')
}

export function pricingDisplayFallbackKey(
  expression: string,
  schema: BillingUsageSchema | null | undefined
): string {
  const compiled = compileBillingExpression(expression)
  let missingUsageMetadata = false
  if (compiled.status === 'ready') {
    visitExpression(compiled.ast, (node) => {
      if (node.kind !== 'call' || node.name !== 'u') return
      const key = node.args[0]
      if (
        !schema ||
        key.kind !== 'literal' ||
        typeof key.value !== 'string' ||
        !Object.hasOwn(schema, key.value)
      ) {
        missingUsageMetadata = true
      }
    })
  }
  return missingUsageMetadata
    ? 'Task usage metadata is unavailable. Pricing details cannot be displayed.'
    : 'This expression cannot be expanded into a price table. View the original expression below.'
}
