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
  RouterContextProvider,
} from '@tanstack/react-router'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { createInstance } from 'i18next'
import type { ReactNode } from 'react'
import { I18nextProvider } from 'react-i18next'
import { expect, it } from 'vitest'

import { LiveTopology } from '@/features/home/components/live-topology'

import { GroupStatusBoard } from '../components/group-status-board'
import type { GroupStatusItem } from '../types'

const groups: GroupStatusItem[] = [
  {
    name: 'Claude live',
    description: 'Coding',
    models: ['claude-sonnet-4'],
    endpoint_types: [],
    ratio: 1,
    cache_hit_rate: 0,
    cache_window_end: 1_800_000_000,
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
    cache_hit_rate: null,
    cache_window_end: 0,
    status: null,
  },
]

function renderGroupStatus(
  ui: ReactNode,
  client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
) {
  const router = createRouter({
    routeTree: createRootRoute(),
    history: createMemoryHistory(),
  })
  return render(ui, {
    wrapper: function GroupStatusProviders(props: { children: ReactNode }) {
      return (
        <QueryClientProvider client={client}>
          <RouterContextProvider router={router}>
            {props.children}
          </RouterContextProvider>
        </QueryClientProvider>
      )
    },
  })
}

it('filters by model and restores results when search is cleared', async () => {
  renderGroupStatus(<GroupStatusBoard groups={groups} isLoading={false} />)
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

it.each([
  { rate: 0, windowEnd: 1_800_000_000, label: '0%' },
  { rate: null, windowEnd: 0, label: '—' },
])('shows cache hit rate $label for API value $rate', (value) => {
  renderGroupStatus(
    <GroupStatusBoard
      groups={[
        {
          ...groups[0],
          cache_hit_rate: value.rate,
          cache_window_end: value.windowEnd,
        },
      ]}
      isLoading={false}
    />
  )

  const card = within(screen.getByRole('link', { name: /Claude live/ }))
  expect(card.getByText('Cache hit rate')).toBeVisible()
  expect(card.getByText('Cache hit rate').closest('dl')).toHaveClass(
    'grid-cols-2',
    'sm:grid-cols-5'
  )
  expect(
    card.getByLabelText(`Cache hit rate: ${value.label}`)
  ).toHaveTextContent(value.label)
})

it('shows the last cache hit rate when the group has no recent traffic', () => {
  renderGroupStatus(
    <GroupStatusBoard
      groups={[
        {
          ...groups[1],
          cache_hit_rate: 81.2,
          cache_window_end: 1_800_000_000,
        },
      ]}
      isLoading={false}
    />
  )

  const card = within(screen.getByRole('link', { name: /Image idle/ }))
  expect(card.getByText('No recent traffic')).toBeVisible()
  expect(card.getByLabelText('Cache hit rate: 81.2%')).toHaveTextContent(
    '81.2%'
  )
})

it('keeps the API cache window during idle refreshes and displays the next window', () => {
  const active = {
    ...groups[0],
    cache_hit_rate: 75.5,
    cache_window_end: 1_800_000_000,
  }
  const view = renderGroupStatus(
    <GroupStatusBoard groups={[active]} isLoading={false} />
  )
  expect(screen.getByLabelText('Cache hit rate: 75.5%')).toBeVisible()

  view.rerender(
    <GroupStatusBoard
      groups={[{ ...active, status: null }]}
      isLoading={false}
    />
  )
  expect(screen.getByText('No recent traffic')).toBeVisible()
  expect(screen.getByLabelText('Cache hit rate: 75.5%')).toBeVisible()

  view.rerender(
    <GroupStatusBoard
      groups={[
        {
          ...active,
          cache_hit_rate: 25,
          cache_window_end: 1_800_000_300,
        },
      ]}
      isLoading={false}
    />
  )
  expect(screen.getByLabelText('Cache hit rate: 25%')).toBeVisible()
  expect(
    screen.queryByLabelText('Cache hit rate: 75.5%')
  ).not.toBeInTheDocument()
})

it('shows the API cache hit rate in the compact topology group row', () => {
  const client = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  })
  client.setQueryData(['status'], {})
  client.setQueryData(['group-status-board', undefined, undefined], {
    summary: null,
    request_count: 40,
    window_start: 1_799_913_600,
    window_end: 1_800_000_000,
    groups: [
      {
        ...groups[0],
        cache_hit_rate: 61.2,
        cache_window_end: 1_800_000_000,
      },
    ],
  })
  const view = renderGroupStatus(<LiveTopology />, client)

  const group = within(screen.getByRole('link', { name: /Claude live/ }))
  expect(group.getByText('Cache 61.2%')).toBeVisible()
  expect(group.getByLabelText('Cache hit rate: 61.2%')).toBeVisible()
  const cache = group.getByText('Cache 61.2%')
  const model = group.getByText('claude-sonnet-4')
  const performance = group.getByText(/20\.0 t\/s/)
  expect(performance).toHaveClass('block', 'truncate')
  expect(performance).not.toContainElement(cache)
  expect(model).toHaveClass('min-w-0', 'flex-1', 'truncate')
  expect(model.parentElement).toHaveClass('flex')
  expect(model.parentElement).toContainElement(cache)
  expect(cache.parentElement).toHaveClass('shrink-0')
  expect(cache.closest('.truncate')).toBeNull()

  view.unmount()
  client.clear()
})

it('explains token weighting and retained windows in the cache rate tooltip', async () => {
  renderGroupStatus(<GroupStatusBoard groups={[groups[0]]} isLoading={false} />)
  await userEvent.setup().hover(screen.getByLabelText('Cache hit rate: 0%'))
  const tooltip = await screen.findByRole('tooltip')
  expect(tooltip).toHaveTextContent(
    'Cache read tokens / total input tokens in the latest 5-minute request window. When there are no new requests, the last valid window is kept.'
  )
  expect(within(tooltip).getByText(/Time:/)).toBeVisible()
})

it.each([
  ['zhCN', '61.2%'],
  ['zhTW', '61.2%'],
  ['en', '61.2%'],
  ['fr', '61,2%'],
  ['ja', '61.2%'],
  ['ru', '61,2%'],
  ['vi', '61,2%'],
  ['invalid_locale', '61.2%'],
])('formats cache hit rates for interface language %s', async (code, value) => {
  const language = createInstance()
  await language.init({
    lng: code,
    fallbackLng: false,
    resources: {
      [code]: { translation: { 'Cache hit rate': 'Cache hit rate' } },
    },
  })
  renderGroupStatus(
    <I18nextProvider i18n={language}>
      <GroupStatusBoard
        groups={[{ ...groups[0], cache_hit_rate: 61.2 }]}
        isLoading={false}
      />
    </I18nextProvider>
  )
  expect(screen.getByLabelText(`Cache hit rate: ${value}`)).toBeVisible()
})
