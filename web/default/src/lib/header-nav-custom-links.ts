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
  INTERFACE_LANGUAGE_OPTIONS,
  normalizeInterfaceLanguage,
  type InterfaceLanguageCode,
} from '@/i18n/languages'

import { normalizeHttpUrl } from './external-url'

export type HeaderNavCustomLinkTitles = Record<InterfaceLanguageCode, string>

export type HeaderNavCustomLink = {
  titles: HeaderNavCustomLinkTitles
  url: string
}

export function createEmptyHeaderNavCustomLinkTitles(): HeaderNavCustomLinkTitles {
  return {
    zhCN: '',
    en: '',
    fr: '',
    ru: '',
    ja: '',
    vi: '',
    zhTW: '',
  }
}

const DEFAULT_HEADER_NAV_CUSTOM_LINK_TITLES: HeaderNavCustomLinkTitles = {
  zhCN: '图片生成',
  en: 'Image Generation',
  fr: "Génération d'images",
  ru: 'Генерация изображений',
  ja: '画像生成',
  vi: 'Tạo hình ảnh',
  zhTW: '圖片生成',
}

export const DEFAULT_HEADER_NAV_CUSTOM_LINKS: HeaderNavCustomLink[] = [
  {
    titles: { ...DEFAULT_HEADER_NAV_CUSTOM_LINK_TITLES },
    url: 'https://image.realseek.wiki/',
  },
]

function cloneDefaultLinks(): HeaderNavCustomLink[] {
  return DEFAULT_HEADER_NAV_CUSTOM_LINKS.map((link) => ({
    ...link,
    titles: { ...link.titles },
  }))
}

export function parseHeaderNavCustomLinks(
  value: unknown
): HeaderNavCustomLink[] {
  if (value === null || value === undefined) {
    return cloneDefaultLinks()
  }
  if (typeof value === 'string' && value.trim() === '') {
    return cloneDefaultLinks()
  }

  let parsed: unknown = value
  if (typeof value === 'string') {
    try {
      parsed = JSON.parse(value)
    } catch {
      return cloneDefaultLinks()
    }
  }

  if (!Array.isArray(parsed)) return cloneDefaultLinks()

  const links: HeaderNavCustomLink[] = []
  parsed.forEach((item) => {
    if (!item || typeof item !== 'object') return

    const record = item as Record<string, unknown>
    const titles = createEmptyHeaderNavCustomLinkTitles()
    if (record.titles && typeof record.titles === 'object') {
      const storedTitles = record.titles as Record<string, unknown>
      INTERFACE_LANGUAGE_OPTIONS.forEach((language) => {
        const value = storedTitles[language.code]
        if (typeof value === 'string') titles[language.code] = value.trim()
      })
    }

    const legacyTitle =
      typeof record.title === 'string' ? record.title.trim() : ''
    const hasLocalizedTitle = Object.values(titles).some(Boolean)
    if (!hasLocalizedTitle && legacyTitle !== '') {
      if (legacyTitle === 'Image Generation') {
        Object.assign(titles, DEFAULT_HEADER_NAV_CUSTOM_LINK_TITLES)
      } else {
        INTERFACE_LANGUAGE_OPTIONS.forEach((language) => {
          titles[language.code] = legacyTitle
        })
      }
    }

    const url = normalizeHttpUrl(record.url)
    if (!Object.values(titles).some(Boolean) || !url) return

    links.push({ titles, url })
  })
  return links
}

export function resolveHeaderNavCustomLinkTitle(
  link: HeaderNavCustomLink,
  language: string | null | undefined
): string {
  const locale = normalizeInterfaceLanguage(language) as InterfaceLanguageCode
  return (
    link.titles[locale] ||
    link.titles.zhCN ||
    link.titles.en ||
    Object.values(link.titles).find(Boolean) ||
    ''
  )
}

export function serializeHeaderNavCustomLinks(
  links: HeaderNavCustomLink[]
): string {
  return JSON.stringify(
    links.map((link) => {
      const titles = createEmptyHeaderNavCustomLinkTitles()
      INTERFACE_LANGUAGE_OPTIONS.forEach((language) => {
        titles[language.code] = link.titles[language.code].trim()
      })
      const fallbackTitle =
        titles.zhCN || titles.en || Object.values(titles).find(Boolean) || ''

      return {
        title: fallbackTitle,
        titles,
        url: link.url.trim(),
      }
    })
  )
}
