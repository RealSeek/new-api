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
import {
  getMarkdown,
  parseMarkdownToStructure,
  type CodeBlockNode,
  type HeadingNode,
  type ParsedNode,
} from 'stream-markdown-parser'

export type DocWidgetName = 'base-urls' | 'groups'

export type DocHeading = {
  id: string
  text: string
  level: 2 | 3
}

export type DocBlock =
  | { kind: 'node'; key: string; node: ParsedNode }
  | { kind: 'heading'; key: string; node: HeadingNode; id: string }
  | { kind: 'code-group'; key: string; blocks: CodeBlockNode[] }
  | { kind: 'widget'; key: string; name: DocWidgetName }

export type ParsedDoc = {
  blocks: DocBlock[]
  headings: DocHeading[]
}

export type DocPlaceholders = {
  baseUrl: string
  siteName: string
  workbenchUrl: string
}

const WIDGET_NAMES: readonly DocWidgetName[] = ['base-urls', 'groups']
const WIDGET_PATTERN = /^<!--\s*widget:([a-z-]+)\s*-->\s*$/
const CODE_GROUP_START = /^<!--\s*code-group\s*-->\s*$/
const CODE_GROUP_END = /^<!--\s*\/code-group\s*-->\s*$/
const HTML_COMMENT = /^<!--[\s\S]*-->\s*$/

let markdownInstance: ReturnType<typeof getMarkdown> | null = null

function getDocsMarkdown() {
  markdownInstance ??= getMarkdown('docs')
  return markdownInstance
}

export function applyDocPlaceholders(
  source: string,
  values: DocPlaceholders
): string {
  return source
    .replaceAll('{{BASE_URL}}', values.baseUrl)
    .replaceAll('{{SITE_NAME}}', values.siteName)
    .replaceAll('{{WORKBENCH_URL}}', values.workbenchUrl)
}

/** Heading text as shown in the table of contents, without inline markup. */
export function getHeadingLabel(text: string): string {
  return text.replaceAll(/[`*_]/g, '').trim()
}

/** Anchor ids keep CJK characters so Chinese headings stay readable in URLs. */
export function slugifyHeading(text: string): string {
  return (
    getHeadingLabel(text)
      .toLowerCase()
      .replaceAll(/[^\p{L}\p{N}]+/gu, '-')
      .replaceAll(/^-+|-+$/g, '') || 'section'
  )
}

function getHtmlContent(node: ParsedNode): string | null {
  if (node.type !== 'html_block') return null
  const content = (node as { content?: unknown }).content
  return typeof content === 'string' ? content.trim() : null
}

export function parseDoc(source: string): ParsedDoc {
  const markdown = getDocsMarkdown()
  const nodes = parseMarkdownToStructure(source, markdown, {
    final: true,
    validateLink: markdown.options.validateLink,
  })

  const blocks: DocBlock[] = []
  const headings: DocHeading[] = []
  const usedIds = new Map<string, number>()
  let codeGroup: CodeBlockNode[] | null = null

  nodes.forEach((node, index) => {
    const key = `${node.type}-${index}`
    const html = getHtmlContent(node)

    if (html != null && CODE_GROUP_START.test(html)) {
      codeGroup = []
      return
    }
    if (html != null && CODE_GROUP_END.test(html)) {
      if (codeGroup && codeGroup.length > 0) {
        blocks.push({ kind: 'code-group', key, blocks: codeGroup })
      }
      codeGroup = null
      return
    }
    if (codeGroup && node.type === 'code_block') {
      codeGroup.push(node as CodeBlockNode)
      return
    }

    const widget = html?.match(WIDGET_PATTERN)?.[1]
    if (widget) {
      if ((WIDGET_NAMES as readonly string[]).includes(widget)) {
        blocks.push({ kind: 'widget', key, name: widget as DocWidgetName })
      }
      return
    }
    // Other comments are authoring notes, not content.
    if (html != null && HTML_COMMENT.test(html)) return

    if (node.type === 'heading') {
      const heading = node as HeadingNode
      if (heading.level === 2 || heading.level === 3) {
        const base = slugifyHeading(heading.text)
        const seen = usedIds.get(base) ?? 0
        usedIds.set(base, seen + 1)
        const id = seen === 0 ? base : `${base}-${seen + 1}`
        headings.push({
          id,
          text: getHeadingLabel(heading.text),
          level: heading.level,
        })
        blocks.push({ kind: 'heading', key, node: heading, id })
        return
      }
    }

    blocks.push({ kind: 'node', key, node })
  })

  return { blocks, headings }
}

/** Section headings of a page for search, read without a full parse. */
export function getDocSearchHeadings(source: string): string[] {
  return Array.from(source.matchAll(/^#{2,3}\s+(.+)$/gm), (match) =>
    getHeadingLabel(match[1])
  )
}
