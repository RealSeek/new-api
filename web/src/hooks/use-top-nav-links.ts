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
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { useStatus } from '@/hooks/use-status'
import {
  parseHeaderNavCustomLinks,
  resolveHeaderNavCustomLinkTitle,
} from '@/lib/header-nav-custom-links'
import { parseHeaderNavModulesFromStatus } from '@/lib/nav-modules'
import { useAuthStore } from '@/stores/auth-store'

export type TopNavLink = {
  title: string
  href: string
  disabled?: boolean
  requiresAuth?: boolean
  external?: boolean
}

/**
 * 根据后端 /api/status 返回的 HeaderNavModules 配置生成顶部导航链接。
 * 后端格式示例（JSON 字符串）：
 * {
 *   home: true,
 *   console: true,
 *   pricing: { enabled: true, requireAuth: false },
 *   rankings: { enabled: true, requireAuth: false },
 *   docs: true,
 *   about: true
 * }
 */
export function useTopNavLinks(): TopNavLink[] {
  const { t, i18n } = useTranslation()
  const { status } = useStatus()
  const { auth } = useAuthStore()

  // 解析内置模块和自定义链接配置。
  const { modules, customLinks } = useMemo(() => {
    const statusRecord = status as Record<string, unknown> | null
    return {
      modules: parseHeaderNavModulesFromStatus(statusRecord),
      customLinks: parseHeaderNavCustomLinks(
        statusRecord?.HeaderNavCustomLinks
      ),
    }
  }, [status])

  // 文档链接可以是外部地址。
  const docsLink: string | undefined = status?.docs_link as string | undefined

  const isAuthed = !!auth?.user

  const links: TopNavLink[] = []

  // 主页
  if (modules?.home !== false) {
    links.push({ title: t('Home'), href: '/' })
  }

  // 控制台
  if (modules?.console !== false) {
    links.push({ title: t('Console'), href: '/dashboard' })
  }

  // 模型广场
  const pricing = modules?.pricing
  if (pricing && typeof pricing === 'object' && pricing.enabled) {
    const requiresAuth = pricing.requireAuth && !isAuthed
    links.push({ title: t('Model Square'), href: '/pricing', requiresAuth })
  }

  customLinks.forEach((link) => {
    links.push({
      title: resolveHeaderNavCustomLinkTitle(
        link,
        i18n.resolvedLanguage ?? i18n.language
      ),
      href: link.url,
      external: true,
    })
  })

  // 排行榜
  const rankings = modules?.rankings
  if (rankings && typeof rankings === 'object' && rankings.enabled) {
    const requiresAuth = rankings.requireAuth && !isAuthed
    links.push({ title: t('Rankings'), href: '/rankings', requiresAuth })
  }

  // 文档
  if (modules?.docs !== false) {
    if (docsLink) {
      links.push({ title: t('Docs'), href: docsLink, external: true })
    } else {
      links.push({ title: t('Docs'), href: '/docs' })
    }
  }

  // 关于
  if (modules?.about !== false) {
    links.push({ title: t('About'), href: '/about' })
  }

  return links
}
