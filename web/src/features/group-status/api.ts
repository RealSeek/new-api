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
import { useQuery } from '@tanstack/react-query'

import { useStatus } from '@/hooks/use-status'
import { api } from '@/lib/api'
import { getModuleAccessFromStatus } from '@/lib/nav-modules'
import { requireServerSuccess } from '@/lib/server-error-message'
import { useAuthStore } from '@/stores/auth-store'

import type { GroupStatusResponse } from './types'

/** The server caches the board for 30s, so polling faster gains nothing. */
export const GROUP_STATUS_REFRESH_MS = 30_000

/**
 * Live group status board. It follows the model square's visibility: an
 * anonymous visitor only loads it when the pricing module is public, so a
 * private site never bounces a public page to sign-in.
 */
export function useGroupStatusBoard() {
  const { status } = useStatus()
  const user = useAuthStore((state) => state.auth.user)
  const isAuthenticated = !!user
  const pricingAccess = getModuleAccessFromStatus(
    status as Record<string, unknown> | null,
    'pricing'
  )
  const canView =
    isAuthenticated || (pricingAccess.enabled && !pricingAccess.requireAuth)

  const query = useQuery({
    queryKey: ['group-status-board', user?.id, user?.group],
    queryFn: async () => {
      const res = await api.get<GroupStatusResponse>(
        '/api/perf-metrics/groups',
        { skipErrorHandler: true, skipAuthRefresh: !isAuthenticated }
      )
      return requireServerSuccess(res.data).data
    },
    enabled: status !== null && canView,
    refetchInterval: GROUP_STATUS_REFRESH_MS,
    staleTime: GROUP_STATUS_REFRESH_MS,
    meta: { errorToast: false },
  })

  return { ...query, canView }
}
