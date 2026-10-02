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
import { splitBillingExprAndRequestRules } from './billing-expr'
import { compileBillingExpression } from './billing-expression/parser'
import type { ExpressionNode } from './billing-expression/types'

export const IMAGE_RESOLUTION_TIERS = ['1k', '2k', '4k'] as const

export type ImageResolutionPriceRow = {
  resolution: string
  price: number
}

export type ImageResolutionPricing = {
  rows: ImageResolutionPriceRow[]
  fallbackPrice: number
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

/** Reads the canonical per-image resolution expression used by image relays. */
export function parseImageResolutionPricing(
  expression: string | null | undefined
): ImageResolutionPricing | null {
  if (!expression) return null
  const split = splitBillingExprAndRequestRules(expression)
  const compiled = compileBillingExpression(split.billingExpr)
  if (compiled.status !== 'ready') return null

  let pricing = compiled.ast
  if (pricing.kind !== 'binary' || pricing.operator !== '*') return null
  if (pricing.left.kind === 'variable' && pricing.left.name === 'image_count') {
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
      !IMAGE_RESOLUTION_TIERS.includes(
        condition.right.value as (typeof IMAGE_RESOLUTION_TIERS)[number]
      )
    ) {
      return null
    }
    const price = readFixedTierPrice(branch.yes)
    if (price === null || price < 0) return null
    rows.push({ resolution: condition.right.value, price })
    branch = branch.no
  }
  const fallbackPrice = readFixedTierPrice(branch)
  if (fallbackPrice === null || fallbackPrice < 0 || rows.length === 0) {
    return null
  }
  rows.sort(
    (left, right) =>
      IMAGE_RESOLUTION_TIERS.indexOf(
        left.resolution as (typeof IMAGE_RESOLUTION_TIERS)[number]
      ) -
      IMAGE_RESOLUTION_TIERS.indexOf(
        right.resolution as (typeof IMAGE_RESOLUTION_TIERS)[number]
      )
  )
  return { rows, fallbackPrice }
}

export function generateImageResolutionPricing(
  rows: ImageResolutionPriceRow[]
): string {
  const ordered = IMAGE_RESOLUTION_TIERS.flatMap((resolution) => {
    const row = rows.find((candidate) => candidate.resolution === resolution)
    return row ? [row] : []
  })
  if (ordered.length === 0) return ''
  const fallbackPrice = ordered[0].price
  const branches = ordered.map(
    ({ resolution, price }) =>
      `param("image_tier") == ${JSON.stringify(resolution)} ? tier(${JSON.stringify(resolution)}, fixed(${price}))`
  )
  return `(${branches.join(' : ')} : tier("image", fixed(${fallbackPrice}))) * image_count`
}
