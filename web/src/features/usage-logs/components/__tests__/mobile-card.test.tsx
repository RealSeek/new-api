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
  flexRender,
  getCoreRowModel,
  useReactTable,
  type VisibilityState,
} from '@tanstack/react-table'
import { act, render, screen, within, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import i18next, { createInstance } from 'i18next'
import { I18nextProvider } from 'react-i18next'
import { expect, it, vi } from 'vitest'

import { usageLogSchema, type UsageLog } from '../../data/schema'
import { useCommonLogsColumns } from '../columns/common-logs-columns'
import { UsageLogsMobileList } from '../usage-logs-mobile-card'
import { UsageLogsProvider, useUsageLogsContext } from '../usage-logs-provider'

const longName = 'enterprise-production-failover-2026-without-any-short-alias'
const log = usageLogSchema.parse({
  id: 1,
  user_id: 2,
  created_at: 1788840000,
  type: 2,
  content: '',
  model_name: longName,
  username: 'production-admin',
  channel: 372,
  channel_name: longName,
  token_name: 'backend-production-token',
  group: 'enterprise-production',
  quota: 123456789,
  use_time: 1.5,
  prompt_tokens: 1200,
  completion_tokens: 800,
  other: JSON.stringify({
    cache_tokens: 300,
    cache_creation_tokens_5m: 200,
    model_ratio: 1,
  }),
})

function Fixture(props: {
  admin?: boolean
  visibility?: VisibilityState
  logs?: UsageLog[]
  loading?: boolean
  desktop?: boolean
}) {
  const columns = useCommonLogsColumns(props.admin ?? true, false)
  const context = useUsageLogsContext()
  const table = useReactTable({
    data: props.logs ?? [log],
    columns,
    getCoreRowModel: getCoreRowModel(),
    state: { columnVisibility: props.visibility ?? {} },
  })
  if (props.desktop) {
    const cell = table
      .getRowModel()
      .rows[0].getAllCells()
      .find((candidate) => candidate.column.id === 'prompt_tokens')
    if (!cell) throw new Error('The log must have a token column')
    return flexRender(cell.column.columnDef.cell, cell.getContext())
  }
  return (
    <>
      <button type='button' onClick={() => context.setSensitiveVisible(false)}>
        Hide sensitive data
      </button>
      <UsageLogsMobileList
        table={table}
        logCategory='common'
        isLoading={props.loading}
      />
    </>
  )
}

function renderLogs(
  props: Parameters<typeof Fixture>[0] = {},
  language = i18next
) {
  return render(
    <I18nextProvider i18n={language}>
      <QueryClientProvider
        client={
          new QueryClient({ defaultOptions: { queries: { retry: false } } })
        }
      >
        <UsageLogsProvider>
          <Fixture {...props} />
        </UsageLogsProvider>
      </QueryClientProvider>
    </I18nextProvider>
  )
}

it('shows model mismatch evidence when tapping the mobile model badge', async () => {
  const user = userEvent.setup()
  renderLogs({
    logs: [
      {
        ...log,
        other: JSON.stringify({
          response_model: {
            requested_model: longName,
            upstream_model: 'mapped-model',
            returned_model: 'unexpected-model',
          },
        }),
      },
    ],
  })
  await user.click(
    screen.getByRole('button', {
      name: `Model: ${longName}, Response model: unexpected-model`,
    })
  )
  const dialog = await screen.findByRole('dialog', { name: 'Model' })
  expect(
    within(dialog).getByText('Response model: unexpected-model')
  ).toBeVisible()
  expect(within(dialog).getByText('mapped-model')).toBeVisible()
  expect(within(dialog).getByText('unexpected-model')).toBeVisible()
})

it('opens long channel text on tap and copies the complete value', async () => {
  const user = userEvent.setup()
  const copy = vi.spyOn(navigator.clipboard, 'writeText').mockResolvedValue()
  renderLogs()
  await user.click(
    screen.getByRole('button', { name: `Channel: ${longName} #372` })
  )
  const dialog = await screen.findByRole('dialog', { name: 'Channel' })
  expect(within(dialog).getByText(`${longName} #372`)).toHaveClass(
    '[overflow-wrap:anywhere]'
  )
  await user.click(
    within(dialog).getByRole('button', { name: 'Copy to clipboard' })
  )
  expect(copy).toHaveBeenCalledWith(`${longName} #372`)
  await user.keyboard('{Escape}')
  await waitFor(() =>
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
  )
  expect(
    screen.getByRole('button', { name: `Channel: ${longName} #372` })
  ).toHaveFocus()
})

it.each([false, true])(
  'copies the full mobile model name with the keyboard without opening details when there is no mapping or difference (response observed: %s)',
  async (observed) => {
    const user = userEvent.setup()
    const copy = vi.spyOn(navigator.clipboard, 'writeText').mockResolvedValue()
    renderLogs({
      logs: [
        {
          ...log,
          other: observed
            ? JSON.stringify({
                response_model: {
                  requested_model: longName,
                  upstream_model: longName,
                  returned_model: longName,
                },
              })
            : log.other,
        },
      ],
    })
    const button = screen.getByRole('button', { name: `Model: ${longName}` })
    const modelText = within(button).getByText(longName)
    expect(modelText).toHaveClass(
      'line-clamp-2',
      'leading-5',
      '[overflow-wrap:anywhere]'
    )
    expect(modelText.closest('[data-slot="status-badge"]')).toHaveClass(
      'max-w-full',
      'min-h-6',
      'py-px'
    )
    button.focus()
    await user.keyboard('{Enter}')
    expect(copy).toHaveBeenCalledWith(longName)
    expect(
      screen.queryByRole('dialog', { name: 'Model' })
    ).not.toBeInTheDocument()
  }
)

it('hides sensitive names and disables full-text inspection when privacy is enabled', async () => {
  const user = userEvent.setup()
  renderLogs()
  await user.click(screen.getByRole('button', { name: 'Hide sensitive data' }))
  expect(screen.queryByText('production-admin')).not.toBeInTheDocument()
  expect(
    screen.queryByRole('button', { name: /Channel:/ })
  ).not.toBeInTheDocument()
  expect(
    screen.queryByRole('button', { name: /Token:/ })
  ).not.toBeInTheDocument()
})

it('respects hidden columns and omits admin fields in the self view', () => {
  renderLogs({
    admin: false,
    visibility: { token_name: false, model_name: false },
  })
  expect(
    screen.queryByRole('button', {
      name: /Channel:|User:|Token:|Group:|Model:/,
    })
  ).not.toBeInTheDocument()
  expect(screen.queryByText('enterprise-production')).not.toBeInTheDocument()
})

it.each([false, true])(
  'keeps regular input and labeled cache quantities separate (desktop=%s)',
  async (desktop) => {
    const user = userEvent.setup()
    renderLogs({ desktop })
    const tokens = screen.getByRole('button', { name: 'Token usage' })
    expect(
      within(tokens).getByLabelText('Regular Input: 700')
    ).toHaveTextContent('700')
    expect(within(tokens).getByLabelText('Output: 800')).toHaveTextContent(
      '800'
    )
    const read = within(tokens).getByText('Cache Read').parentElement
    const write = within(tokens).getByText('Cache Write').parentElement
    expect(read).toHaveTextContent('300')
    expect(write).toHaveTextContent('200')
    expect(read).toHaveClass('text-muted-foreground/60')
    expect(write).toHaveClass('text-muted-foreground/60')
    expect(tokens.querySelector('svg')).not.toBeInTheDocument()
    expect(read?.compareDocumentPosition(write as HTMLElement)).toBe(
      Node.DOCUMENT_POSITION_FOLLOWING
    )
    expect(read?.parentElement).toHaveClass('flex-wrap')
    expect(tokens).toHaveClass('max-w-full', 'items-start')
    await user.hover(tokens)
    const tooltip = await screen.findByRole('tooltip')
    expect(tooltip).toHaveTextContent(
      'Total Input = Regular Input + Cache Read + Cache Write'
    )
    expect(tooltip).toHaveTextContent('1,200 = 700 + 300 + 200')
    expect(tooltip).toHaveTextContent(
      'Unreused Input = Total Input - Cache Read = Regular Input + Cache Write'
    )
    expect(tooltip).toHaveTextContent('900 = 1,200 - 300 = 700 + 200')
    expect(tooltip).toHaveTextContent('Cache hits = Cache Read')
    await user.keyboard('{Escape}')
    await waitFor(() =>
      expect(screen.queryByRole('tooltip')).not.toBeInTheDocument()
    )
  }
)

it.each([
  {
    total: 1717,
    output: 522,
    read: 0,
    write: 1713,
    ordinary: 4,
    unreused: 1717,
  },
  {
    total: 21038,
    output: 435,
    read: 1827,
    write: 19207,
    ordinary: 4,
    unreused: 19211,
  },
  {
    total: 67743,
    output: 676,
    read: 60628,
    write: 7113,
    ordinary: 2,
    unreused: 7115,
  },
])(
  'shows $ordinary regular input from total $total without changing cache counts',
  async ({ total, output, read, write, ordinary, unreused }) => {
    renderLogs({
      desktop: true,
      logs: [
        {
          ...log,
          prompt_tokens: total,
          completion_tokens: output,
          other: JSON.stringify({
            usage_semantic: 'openai',
            input_tokens_total: total,
            cache_tokens: read,
            cache_creation_tokens: write,
          }),
        },
      ],
    })
    expect(screen.getByLabelText(`Regular Input: ${ordinary}`)).toBeVisible()
    if (read === 0) {
      expect(screen.queryByText('Cache Read')).not.toBeInTheDocument()
    }
    expect(screen.getByText('Cache Write').parentElement).toHaveTextContent(
      write.toLocaleString('en-US')
    )
    await userEvent
      .setup()
      .hover(screen.getByRole('button', { name: 'Token usage' }))
    expect(await screen.findByRole('tooltip')).toHaveTextContent(
      `${unreused.toLocaleString('en-US')} = ${total.toLocaleString('en-US')} - ${read.toLocaleString('en-US')} = ${ordinary} + ${write.toLocaleString('en-US')}`
    )
  }
)

it.each([
  { semantic: 'anthropic', claude: false, total: undefined, prompt: 4 },
  { semantic: undefined, claude: true, total: undefined, prompt: 4 },
  { semantic: 'anthropic', claude: true, total: 21038, prompt: 4 },
  { semantic: 'openai', claude: true, total: undefined, prompt: 21038 },
])(
  'does not subtract cache twice for recorded semantics $semantic and legacy Claude=$claude',
  async ({ semantic, claude, total, prompt }) => {
    renderLogs({
      desktop: true,
      logs: [
        {
          ...log,
          model_name: 'opaque-model',
          prompt_tokens: prompt,
          completion_tokens: 435,
          other: JSON.stringify({
            usage_semantic: semantic,
            claude,
            input_tokens_total: total,
            cache_tokens: 1827,
            cache_creation_tokens_5m: 19000,
            cache_creation_tokens_1h: 207,
          }),
        },
      ],
    })
    const tokens = screen.getByRole('button', { name: 'Token usage' })
    expect(within(tokens).getByLabelText('Regular Input: 4')).toBeVisible()
  }
)

it('hides zero cache rows while retaining ordinary input and output', () => {
  renderLogs({ desktop: true, logs: [{ ...log, other: '{}' }] })
  expect(screen.getByLabelText('Regular Input: 1,200')).toBeVisible()
  expect(screen.getByLabelText('Output: 800')).toBeVisible()
  expect(screen.queryByText('Cache Read')).not.toBeInTheDocument()
  expect(screen.queryByText('Cache Write')).not.toBeInTheDocument()
})

it.each([
  {
    cache_write_tokens: 50,
    cache_creation_tokens: 40,
    cache_creation_tokens_5m: 10,
  },
  {
    cache_creation_tokens: 50,
    cache_creation_tokens_5m: 10,
    cache_creation_tokens_1h: 20,
    cache_creation_tokens_12h: 20,
  },
])(
  'uses the complete cache write total when TTL details are partial: %j',
  (cache) => {
    renderLogs({
      desktop: true,
      logs: [{ ...log, prompt_tokens: 52, other: JSON.stringify(cache) }],
    })
    expect(screen.getByLabelText('Regular Input: 2')).toBeVisible()
    expect(screen.getByText('Cache Write').parentElement).toHaveTextContent(
      '50'
    )
  }
)

it('shows a fully cached Anthropic request even when ordinary input and output are zero', () => {
  renderLogs({
    logs: [
      {
        ...log,
        prompt_tokens: 0,
        completion_tokens: 0,
        other: JSON.stringify({
          usage_semantic: 'anthropic',
          cache_tokens: 1827,
          cache_creation_tokens: 19207,
        }),
      },
    ],
  })
  expect(screen.getByLabelText('Regular Input: 0')).toBeVisible()
  expect(screen.getByText('Cache Read').parentElement).toHaveTextContent(
    '1,827'
  )
  expect(screen.getByText('Cache Write').parentElement).toHaveTextContent(
    '19,207'
  )
})

it.each([
  ['zhCN', '1,200'],
  ['zhTW', '1,200'],
  ['en', '1,200'],
  ['fr', '1\u202f200'],
  ['ja', '1,200'],
  ['ru', '1\u00a0200'],
  ['vi', '1.200'],
  ['invalid_locale', '1,200'],
])(
  'formats token quantities for interface language %s',
  async (code, number) => {
    const language = createInstance()
    await language.init({
      lng: code,
      fallbackLng: false,
      resources: { [code]: { translation: { 'Token usage': 'Token usage' } } },
    })
    renderLogs({ desktop: true, logs: [{ ...log, other: '{}' }] }, language)
    expect(
      screen.getByLabelText(`Regular Input: ${number}`, {
        normalizer: (value) => value,
      })
    ).toBeVisible()
  }
)

it('updates token formatting when the interface language changes', async () => {
  const language = createInstance()
  await language.init({
    lng: 'en',
    resources: {
      en: { translation: { 'Token usage': 'Token usage' } },
      fr: { translation: { 'Token usage': 'Token usage' } },
    },
  })
  renderLogs({ desktop: true, logs: [{ ...log, other: '{}' }] }, language)
  expect(screen.getByLabelText('Regular Input: 1,200')).toBeVisible()
  await act(() => language.changeLanguage('fr'))
  expect(
    screen.getByLabelText('Regular Input: 1\u202f200', {
      normalizer: (value) => value,
    })
  ).toBeVisible()
})

it('shows the established empty state when no logs exist', () => {
  renderLogs({ logs: [] })
  expect(screen.getByText('No Logs Found')).toBeVisible()
})

it.each([false, true])(
  'keeps timing in the right column for streaming=%s',
  (streaming) => {
    renderLogs({
      logs: [
        {
          ...log,
          is_stream: streaming,
          use_time: 58,
          other: JSON.stringify({ frt: 58000 }),
        },
      ],
    })
    const row = screen
      .getByRole('button', { name: /^Time:/ })
      .closest('[data-slot="log-time-and-timing"]')
    expect(row).toHaveClass('grid', 'grid-cols-2')
    expect(
      screen.getByRole('button', { name: /^Time:/ }).parentElement
    ).toHaveClass('flex-col', 'justify-between')
    expect(
      within(row as HTMLElement)
        .getByText('Duration')
        .closest('.col-start-2')
    ).not.toBeNull()
    if (streaming) {
      expect(within(row as HTMLElement).getByText('First token')).toBeVisible()
    }
  }
)

it('retains the user avatar and model badge in the mobile summary', () => {
  renderLogs()
  expect(screen.getByText('P')).toBeVisible()
  const modelButton = screen.getByRole('button', { name: `Model: ${longName}` })
  expect(modelButton.querySelector('[data-slot="status-badge"]')).not.toBeNull()
})

it('omits unused token and throughput placeholders for async jobs', () => {
  renderLogs({
    logs: [
      {
        ...log,
        prompt_tokens: 0,
        completion_tokens: 0,
        other: JSON.stringify({ is_task: true }),
      },
    ],
  })
  expect(screen.getByText('Async')).toBeVisible()
  expect(screen.queryByText('Input')).not.toBeInTheDocument()
  const timing = screen
    .getByRole('button', { name: /^Time:/ })
    .closest('[data-slot="log-time-and-timing"]')
  expect(within(timing as HTMLElement).queryByText('—')).not.toBeInTheDocument()
})

it('labels a task whose result was returned in the response as synchronous', () => {
  renderLogs({
    logs: [
      {
        ...log,
        prompt_tokens: 0,
        completion_tokens: 0,
        other: JSON.stringify({ is_task: true, task_sync: true }),
      },
    ],
  })
  expect(screen.getByText('Sync')).toBeVisible()
  expect(screen.queryByText('Async')).not.toBeInTheDocument()
})

it('shows mapped model names in full when inspecting a mobile model badge', async () => {
  const user = userEvent.setup()
  renderLogs({
    logs: [
      {
        ...log,
        other: JSON.stringify({
          is_model_mapped: true,
          upstream_model_name:
            'provider-production-mapped-model-with-a-long-name',
        }),
      },
    ],
  })
  await user.click(screen.getByRole('button', { name: `Model: ${longName}` }))
  const dialog = await screen.findByRole('dialog', { name: 'Model' })
  expect(within(dialog).getByText('Actual Model')).toBeVisible()
  expect(
    within(dialog).getByText(
      'provider-production-mapped-model-with-a-long-name'
    )
  ).toBeVisible()
})

it('shows loading placeholders without displaying stale log fields', () => {
  renderLogs({ loading: true })
  expect(screen.getByRole('status', { name: 'Loading' })).toHaveAttribute(
    'aria-busy',
    'true'
  )
  expect(
    screen.queryByRole('button', { name: /^Model:/ })
  ).not.toBeInTheDocument()
})
