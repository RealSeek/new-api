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
import { describe, expect, it } from 'vitest'

import { DOC_CONTENT } from '../lib/content'
import { applyDocPlaceholders, parseDoc } from '../lib/markdown'
import { DOC_PAGES } from '../pages'

describe('documentation content', () => {
  it('keeps duplicate CJK headings navigable with unique anchors', () => {
    const doc = parseDoc('## 首次调用\n\n### 配置 `API`\n\n## 首次调用')
    expect(doc.headings).toEqual([
      { id: '首次调用', text: '首次调用', level: 2 },
      { id: '配置-api', text: '配置 API', level: 3 },
      { id: '首次调用-2', text: '首次调用', level: 2 },
    ])
  })

  it('renders authoring markers as widgets and selectable code examples', () => {
    const doc = parseDoc(
      '<!-- widget:base-urls -->\n\n<!-- code-group -->\n\n```bash\ncurl example\n```\n\n```powershell\ncurl.exe example\n```\n\n<!-- /code-group -->'
    )
    expect(doc.blocks.map((block) => block.kind)).toEqual([
      'widget',
      'code-group',
    ])
    expect(doc.blocks[0]).toMatchObject({ name: 'base-urls' })
    expect(doc.blocks[1]).toMatchObject({
      blocks: [{ language: 'bash' }, { language: 'powershell' }],
    })
  })

  it('replaces every configured URL and site name in examples', () => {
    expect(
      applyDocPlaceholders(
        '{{BASE_URL}}/v1 {{SITE_NAME}} {{WORKBENCH_URL}} {{BASE_URL}}',
        {
          baseUrl: 'https://api.example',
          siteName: 'Example',
          workbenchUrl: 'https://art.example',
        }
      )
    ).toBe(
      'https://api.example/v1 Example https://art.example https://api.example'
    )
  })

  it('provides each navigable page and resolves all internal documentation links', () => {
    const slugs = new Set(DOC_PAGES.map((page) => page.slug))
    for (const page of DOC_PAGES) {
      const content = DOC_CONTENT[page.slug]
      expect(content, page.slug).toBeTruthy()
      const parsed = parseDoc(content)
      expect(parsed.blocks.length, page.slug).toBeGreaterThan(0)
      for (const match of content.matchAll(
        /\]\(\/docs\/([^)#\s]+)(?:#([^)]*))?\)/g
      )) {
        expect(slugs.has(match[1]), `${page.slug} -> ${match[1]}`).toBe(true)
        if (match[2]) {
          expect(
            parseDoc(DOC_CONTENT[match[1]]).headings.some(
              (heading) => heading.id === decodeURIComponent(match[2])
            )
          ).toBe(true)
        }
      }
    }
  })
})
