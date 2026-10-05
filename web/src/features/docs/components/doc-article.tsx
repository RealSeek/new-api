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
import { renderChildren } from '@/components/ai-elements/response-renderer'
import { renderCodeBlock } from '@/components/ai-elements/response-renderer-blocks'
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs'
import { cn } from '@/lib/utils'

import type { DocBlock } from '../lib/markdown'
import { DocWidget } from './doc-widgets'

const HEADING_CLASS = {
  2: 'mt-12 mb-4 scroll-mt-24 border-b pb-2 text-xl font-semibold tracking-tight first:mt-0',
  3: 'mt-8 mb-3 scroll-mt-24 text-base font-semibold',
} as const

type DocArticleProps = {
  blocks: DocBlock[]
  className?: string
}

export function DocArticle(props: DocArticleProps) {
  return (
    <div
      className={cn(
        'min-w-0 text-[15px] leading-7 text-pretty',
        props.className
      )}
    >
      {props.blocks.map((block) => {
        switch (block.kind) {
          case 'heading': {
            const level = block.node.level === 2 ? 2 : 3
            const Tag = level === 2 ? 'h2' : 'h3'
            return (
              <Tag
                key={block.key}
                id={block.id}
                className={HEADING_CLASS[level]}
              >
                <a href={`#${block.id}`} className='hover:text-primary'>
                  {renderChildren(block.node.children)}
                </a>
              </Tag>
            )
          }
          case 'code-group':
            return (
              <Tabs key={block.key} defaultValue='0' className='my-4 gap-2'>
                <TabsList>
                  {block.blocks.map((code, index) => (
                    <TabsTrigger
                      key={`${code.language}-${code.code}`}
                      value={String(index)}
                    >
                      {code.language || 'text'}
                    </TabsTrigger>
                  ))}
                </TabsList>
                {block.blocks.map((code, index) => (
                  <TabsContent
                    key={`${code.language}-${code.code}`}
                    value={String(index)}
                  >
                    {renderCodeBlock(code, `${block.key}-${index}`)}
                  </TabsContent>
                ))}
              </Tabs>
            )
          case 'widget':
            return <DocWidget key={block.key} name={block.name} />
          default:
            return <div key={block.key}>{renderChildren([block.node])}</div>
        }
      })}
    </div>
  )
}
