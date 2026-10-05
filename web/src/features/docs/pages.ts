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
  BookMarked,
  Code,
  LayoutGrid,
  LifeBuoy,
  Plug,
  Rocket,
  WandSparkles,
  type LucideIcon,
} from 'lucide-react'

/** Content is Chinese-only, so titles are plain strings rather than i18n keys. */
export type DocPage = {
  slug: string
  title: string
  description: string
}

export type DocSection = {
  id: string
  title: string
  icon: LucideIcon
  pages: DocPage[]
}

export const DOC_SECTIONS: DocSection[] = [
  {
    id: 'start',
    title: '快速开始',
    icon: Rocket,
    pages: [
      {
        slug: 'introduction',
        title: '平台介绍',
        description: '了解平台能做什么、接口地址与基本概念',
      },
      {
        slug: 'register',
        title: '注册与登录',
        description: '创建账户、验证邮箱并登录控制台',
      },
      {
        slug: 'billing',
        title: '充值与计费',
        description: '余额充值、模型价格与分组倍率的计算方式',
      },
      {
        slug: 'api-keys',
        title: '创建令牌',
        description: '创建 API Key，选择分组与额度限制',
      },
      {
        slug: 'first-request',
        title: '第一次调用',
        description: '用 curl 或 SDK 发出第一个请求',
      },
    ],
  },
  {
    id: 'groups',
    title: '分组与模型',
    icon: LayoutGrid,
    pages: [
      {
        slug: 'groups',
        title: '分组说明',
        description: '各分组的用途、倍率与实时可用率',
      },
      {
        slug: 'models',
        title: '模型列表',
        description: '查询可用模型与模型命名规则',
      },
    ],
  },
  {
    id: 'clients',
    title: '客户端接入',
    icon: Plug,
    pages: [
      {
        slug: 'cc-switch',
        title: 'CC Switch',
        description: '一键把令牌导入 Claude Code、Codex 与 Gemini CLI',
      },
      {
        slug: 'claude-code',
        title: 'Claude Code',
        description: '配置 Claude Code 使用本站 Anthropic 接口',
      },
      {
        slug: 'codex',
        title: 'Codex CLI',
        description: '配置 Codex 使用本站 Responses 接口',
      },
      {
        slug: 'gemini-cli',
        title: 'Gemini CLI',
        description: '配置 Gemini CLI 使用本站 Gemini 原生接口',
      },
      {
        slug: 'chat-clients',
        title: '聊天客户端',
        description: 'Cherry Studio、LobeHub 等客户端的接入方法',
      },
    ],
  },
  {
    id: 'api',
    title: 'API 参考',
    icon: Code,
    pages: [
      {
        slug: 'api-overview',
        title: '接口概览',
        description: 'Base URL、鉴权方式、流式输出与请求 ID',
      },
      {
        slug: 'chat-completions',
        title: 'Chat Completions',
        description: 'OpenAI 兼容的对话补全接口',
      },
      {
        slug: 'responses',
        title: 'Responses',
        description: 'OpenAI Responses 接口与 Codex 用法',
      },
      {
        slug: 'anthropic-messages',
        title: 'Anthropic Messages',
        description: 'Claude 原生 Messages 接口',
      },
      {
        slug: 'gemini-native',
        title: 'Gemini 原生接口',
        description: 'generateContent 与 streamGenerateContent',
      },
      {
        slug: 'images',
        title: '图片生成',
        description: '使用 GPT、Gemini、Grok 模型生成图片',
      },
      {
        slug: 'videos',
        title: '视频生成',
        description: '视频统一接口：创建、查询、下载与回调',
      },
      {
        slug: 'errors',
        title: '错误处理',
        description: '常见错误码的含义与排查方法',
      },
    ],
  },
  {
    id: 'workbench',
    title: 'AI 工作台',
    icon: WandSparkles,
    pages: [
      {
        slug: 'workbench',
        title: '工作台入门',
        description: '用本站账户登录 AI 工作台并完成初始化',
      },
      {
        slug: 'workbench-features',
        title: '功能指南',
        description: '对话、生图、视频、画布与作品广场',
      },
    ],
  },
  {
    id: 'help',
    title: '帮助',
    icon: LifeBuoy,
    pages: [
      {
        slug: 'faq',
        title: '常见问题',
        description: '账户、计费、分组与客户端的常见问题',
      },
    ],
  },
]

export const DOC_PAGES: DocPage[] = DOC_SECTIONS.flatMap(
  (section) => section.pages
)

export const DEFAULT_DOC_SLUG = DOC_PAGES[0].slug

export function findDocPage(slug: string) {
  const index = DOC_PAGES.findIndex((page) => page.slug === slug)
  if (index === -1) return null
  const section = DOC_SECTIONS.find((item) =>
    item.pages.some((page) => page.slug === slug)
  )
  return {
    page: DOC_PAGES[index],
    section,
    previous: DOC_PAGES[index - 1],
    next: DOC_PAGES[index + 1],
  }
}

/** The AI workbench is a separate site that signs in with this site's accounts. */
export const WORKBENCH_URL = 'https://art.realseek.wiki'

export const DOCS_SECTION_ICON = BookMarked
