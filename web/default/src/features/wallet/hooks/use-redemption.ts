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
import { useState, useCallback } from 'react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'

import { formatQuota } from '@/lib/format'

import { redeemTopupCodes } from '../api'

// ============================================================================
// Redemption Hook
// ============================================================================

export function useRedemption() {
  const { t } = useTranslation()
  const [redeeming, setRedeeming] = useState(false)

  const redeemCodes = useCallback(
    async (codeInput: string): Promise<boolean> => {
      const codes = codeInput
        .split(/\r?\n/)
        .map((code) => code.trim())
        .filter(Boolean)

      if (codes.length === 0) {
        toast.error(t('Please enter a redemption code'))
        return false
      }
      if (codes.length > 100) {
        toast.error(
          t('You can redeem up to {{count}} codes at once', { count: 100 })
        )
        return false
      }
      if (new Set(codes).size !== codes.length) {
        toast.error(t('Duplicate redemption codes are not allowed'))
        return false
      }

      try {
        setRedeeming(true)
        const response = await redeemTopupCodes(codes)

        if (response.success && typeof response.data === 'number') {
          const quotaAdded = formatQuota(response.data)
          if (codes.length === 1) {
            toast.success(
              t('Redemption successful! Added: {{quota}}', {
                quota: quotaAdded,
              })
            )
          } else {
            toast.success(
              t('Redeemed {{count}} codes successfully! Added: {{quota}}', {
                count: codes.length,
                quota: quotaAdded,
              })
            )
          }
          return true
        }

        toast.error(response.message || t('Redemption failed'))
        return false
      } catch {
        toast.error(t('Redemption failed'))
        return false
      } finally {
        setRedeeming(false)
      }
    },
    [t]
  )

  return {
    redeeming,
    redeemCodes,
  }
}
