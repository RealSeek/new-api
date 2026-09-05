import { fireEvent, render, screen, waitFor } from '@testing-library/react'
import { describe, expect, test, vi } from 'vitest'
import { ModelDetailsApi } from '../model-details-api'
import type { PricingModel } from '../../types'

vi.mock('@/hooks/use-status', () => ({
  useStatus: () => ({ status: { server_address: 'https://ai.example.test' } }),
}))
vi.mock('@/components/ai-elements/code-block', () => ({
  CodeBlock: ({ code }: { code: string }) => (
    <pre data-testid='sample'>{code}</pre>
  ),
  CodeBlockCopyButton: () => null,
}))

describe('图片生成代码示例', () => {
  test('图片端点提供本站请求及 Python、TypeScript、JavaScript 的 Base64 保存示例', async () => {
    render(
      <ModelDetailsApi
        model={
          {
            model_name: 'gpt-image-2',
            supported_endpoint_types: ['image-generation'],
          } as PricingModel
        }
        endpointMap={{
          'image-generation': {
            path: '/v1/images/generations',
            method: 'POST',
          },
        }}
      />
    )
    expect(screen.getByTestId('sample')).toHaveTextContent(
      'https://ai.example.test/v1/images/generations'
    )
    expect(screen.getByTestId('sample')).toHaveTextContent('gpt-image-2')
    for (const language of ['Python', 'TypeScript', 'JavaScript']) {
      fireEvent.click(screen.getByRole('tab', { name: language }))
      await waitFor(() =>
        expect(screen.getByTestId('sample')).toHaveTextContent('b64_json')
      )
      expect(screen.getByTestId('sample')).toHaveTextContent('image.png')
      expect(screen.getByTestId('sample')).toHaveTextContent(
        'https://ai.example.test'
      )
      expect(screen.getByTestId('sample')).not.toHaveTextContent(
        'api.openai.com'
      )
    }
  })
})
