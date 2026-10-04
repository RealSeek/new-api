import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it, vi } from 'vitest'

import { ModelDescriptions } from '../model-descriptions'

vi.mock('@/features/channels/api', () => ({
  getChannels: vi.fn().mockResolvedValue({
    success: true,
    data: {
      items: [
        {
          id: 7,
          name: 'claude-kiro',
          models: 'claude-sonnet-4-6, claude-opus-5-5,claude-sonnet-4-6',
        },
        { id: 8, name: 'codex-plus', models: 'gpt-6.1-sol' },
      ],
    },
  }),
}))

vi.mock('../../api', () => ({
  getChannelModelDescriptions: vi
    .fn()
    .mockResolvedValue({ success: true, data: [] }),
  saveChannelModelDescription: vi.fn(),
}))

describe('ModelDescriptions', () => {
  it('offers only the selected channel models once a channel is picked', async () => {
    const user = userEvent.setup()
    render(
      <QueryClientProvider client={new QueryClient()}>
        <ModelDescriptions />
      </QueryClientProvider>
    )
    const modelInput = screen.getByRole('combobox', { name: 'Model name' })
    expect(modelInput).toBeDisabled()

    await user.click(screen.getByRole('combobox', { name: 'Channel' }))
    await user.click(
      await screen.findByRole('option', { name: 'claude-kiro (#7)' })
    )
    await user.click(modelInput)

    const options = await screen.findAllByRole('option')
    expect(options.map((option) => option.textContent)).toEqual([
      'claude-sonnet-4-6',
      'claude-opus-5-5',
    ])
  })
})
