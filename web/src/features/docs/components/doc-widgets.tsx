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
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { ErrorState } from '@/components/error-state'
import { useGroupStatusBoard } from '@/features/group-status/api'
import { GroupStatusBoard } from '@/features/group-status/components/group-status-board'
import { useGatewayBaseUrl } from '@/features/group-status/lib/use-base-url'
import { getServerErrorMessage } from '@/lib/server-error-message'

import type { DocWidgetName } from '../lib/markdown'

/** Copyable base URL per request format, resolved from the configured server address. */
function BaseUrlsWidget() {
  const { t } = useTranslation()
  const baseUrl = useGatewayBaseUrl()
  const formats = [
    {
      key: 'openai',
      label: t('OpenAI compatible'),
      url: `${baseUrl}/v1`,
      hint: t('OpenAI SDK, Codex, Cherry Studio and most clients'),
    },
    {
      key: 'anthropic',
      label: t('Anthropic compatible'),
      url: baseUrl,
      hint: t('Anthropic SDK and Claude Code'),
    },
    {
      key: 'gemini',
      label: t('Gemini native'),
      url: baseUrl,
      hint: t('Google Gen AI SDK and Gemini CLI'),
    },
  ]

  return (
    <div className='my-6 grid gap-3 sm:grid-cols-3'>
      {formats.map((format) => (
        <div
          key={format.key}
          className='hover:border-primary/40 min-w-0 rounded-lg border p-4 transition-colors'
        >
          <div className='text-muted-foreground text-xs font-medium'>
            {format.label}
          </div>
          <div className='mt-2 flex items-center gap-1'>
            <code
              className='min-w-0 flex-1 truncate font-mono text-sm'
              title={format.url}
            >
              {format.url}
            </code>
            <CopyButton value={format.url} className='size-7' />
          </div>
          <p className='text-muted-foreground mt-2 text-xs leading-5'>
            {format.hint}
          </p>
        </div>
      ))}
    </div>
  )
}

/** The same live board as the homepage, so the docs never list stale groups. */
function GroupsWidget() {
  const { t } = useTranslation()
  const board = useGroupStatusBoard()

  if (!board.canView) {
    return (
      <p className='text-muted-foreground my-6 rounded-xl border border-dashed p-4 text-sm'>
        {t('Sign in to view live group status.')}{' '}
        <Link
          to='/sign-in'
          className='text-primary underline-offset-4 hover:underline'
        >
          {t('Sign in')}
        </Link>
      </p>
    )
  }

  return (
    <div className='my-6'>
      {board.isError ? (
        <ErrorState
          description={getServerErrorMessage(board.error)}
          onRetry={() => {
            void board.refetch()
          }}
        />
      ) : (
        <GroupStatusBoard
          groups={board.data?.groups ?? []}
          isLoading={board.isLoading}
        />
      )}
    </div>
  )
}

export function DocWidget(props: { name: DocWidgetName }) {
  if (props.name === 'groups') return <GroupsWidget />
  return <BaseUrlsWidget />
}
