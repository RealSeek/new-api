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
import assert from 'node:assert/strict'
import { describe, test } from 'node:test'

import {
  DEFAULT_HEADER_NAV_CUSTOM_LINKS,
  createEmptyHeaderNavCustomLinkTitles,
  parseHeaderNavCustomLinks,
  resolveHeaderNavCustomLinkTitle,
  serializeHeaderNavCustomLinks,
} from './header-nav-custom-links'

describe('自定义顶栏链接配置', () => {
  test('缺少配置时返回独立的默认图片生成入口', () => {
    const first = parseHeaderNavCustomLinks(undefined)
    const second = parseHeaderNavCustomLinks(undefined)

    assert.deepEqual(first, DEFAULT_HEADER_NAV_CUSTOM_LINKS)
    assert.notEqual(first, second)
    assert.notEqual(first[0], second[0])
    assert.notEqual(first[0]?.titles, second[0]?.titles)
  })

  test('显式空数组会关闭所有自定义入口', () => {
    assert.deepEqual(parseHeaderNavCustomLinks('[]'), [])
  })

  test('兼容旧版默认名称并补齐内置翻译', () => {
    const links = parseHeaderNavCustomLinks([
      { title: 'Image Generation', url: 'https://image.realseek.wiki/' },
    ])

    assert.equal(links[0]?.titles.zhCN, '图片生成')
    assert.equal(links[0]?.titles.en, 'Image Generation')
    assert.equal(links[0]?.titles.ja, '画像生成')
  })

  test('只保留至少有一个语言名称且地址安全的入口', () => {
    const links = parseHeaderNavCustomLinks([
      {
        titles: { zhCN: ' 图片生成 ', en: ' Image Generation ' },
        url: 'https://image.realseek.wiki/',
      },
      { titles: { zhCN: '不安全' }, url: 'javascript:alert(1)' },
      { titles: {}, url: 'https://example.com/' },
    ])

    assert.equal(links.length, 1)
    assert.equal(links[0]?.titles.zhCN, '图片生成')
    assert.equal(links[0]?.titles.en, 'Image Generation')
  })

  test('按当前界面语言取名称，并回退到简体中文和英文', () => {
    const titles = createEmptyHeaderNavCustomLinkTitles()
    titles.zhCN = '帮助中心'
    titles.en = 'Help Center'
    const link = { titles, url: 'https://example.com/help' }

    assert.equal(resolveHeaderNavCustomLinkTitle(link, 'zh-CN'), '帮助中心')
    assert.equal(resolveHeaderNavCustomLinkTitle(link, 'en-US'), 'Help Center')
    assert.equal(resolveHeaderNavCustomLinkTitle(link, 'ja'), '帮助中心')
  })

  test('序列化时清理多语言名称并写入兼容回退名称', () => {
    const titles = createEmptyHeaderNavCustomLinkTitles()
    titles.zhCN = ' 文档 '
    titles.en = ' Docs '

    const serialized = serializeHeaderNavCustomLinks([
      { titles, url: ' https://example.com/docs ' },
    ])

    assert.deepEqual(JSON.parse(serialized), [
      {
        title: '文档',
        titles: {
          zhCN: '文档',
          en: 'Docs',
          fr: '',
          ru: '',
          ja: '',
          vi: '',
          zhTW: '',
        },
        url: 'https://example.com/docs',
      },
    ])
  })
})
