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
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import {
  createMemoryHistory,
  createRootRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { expect, it } from 'vitest'

import { GroupStatusBoard } from '../components/group-status-board'
import type { GroupStatusItem } from '../types'

const groups: GroupStatusItem[] = [
  {
    name: 'Claude live',
    description: 'Coding',
    models: ['claude-sonnet-4'],
    endpoint_types: [],
    ratio: 1,
    status: {
      request_count: 40,
      success_rate: 95,
      avg_latency_ms: 100,
      avg_tps: 20,
      recent: { hours: 1, request_count: 40, success_rate: 95 },
      series: [],
    },
  },
  {
    name: 'Image idle',
    description: 'Images',
    models: ['gpt-image-2'],
    endpoint_types: ['image-generation'],
    ratio: 0.5,
    status: null,
  },
]

it('filters by model and restores results when search is cleared', async () => {
  const root = createRootRoute({
    component: () => <GroupStatusBoard groups={groups} isLoading={false} />,
  })
  const router = createRouter({
    routeTree: root,
    history: createMemoryHistory(),
  })
  render(
    <QueryClientProvider client={new QueryClient()}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  )
  const user = userEvent.setup()
  const search = await screen.findByRole('textbox', {
    name: 'Search groups or models',
  })
  await user.type(search, 'gpt-image')
  expect(screen.getByRole('link', { name: /Image idle/ })).toBeVisible()
  expect(
    screen.queryByRole('link', { name: /Claude live/ })
  ).not.toBeInTheDocument()
  await user.clear(search)
  expect(screen.getByRole('link', { name: /Claude live/ })).toBeVisible()
  await user.click(screen.getByRole('switch'))
  expect(
    screen.queryByRole('link', { name: /Image idle/ })
  ).not.toBeInTheDocument()
  await user.type(search, 'missing')
  expect(screen.getByText('No matching groups')).toBeVisible()
})
