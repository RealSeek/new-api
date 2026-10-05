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
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { cn } from '@/lib/utils'

import type { DocHeading } from '../lib/markdown'

export function DocToc(props: { headings: DocHeading[] }) {
  const { t } = useTranslation()
  const [activeId, setActiveId] = useState('')
  useEffect(() => {
    setActiveId('')
    const observer = new IntersectionObserver(
      (entries) => {
        for (const entry of entries) {
          if (entry.isIntersecting) setActiveId(entry.target.id)
        }
      },
      { rootMargin: '-80px 0px -65% 0px' }
    )
    for (const heading of props.headings) {
      const element = document.getElementById(heading.id)
      if (element) observer.observe(element)
    }
    return () => observer.disconnect()
  }, [props.headings])

  return (
    <nav aria-label={t('On this page')} className='space-y-2 border-l pl-4'>
      <p className='mb-4 text-xs font-semibold'>{t('On this page')}</p>
      {props.headings.map((heading) => (
        <a
          key={heading.id}
          href={`#${heading.id}`}
          aria-current={activeId === heading.id ? 'location' : undefined}
          className={cn(
            'block text-xs leading-5 transition-colors hover:text-primary',
            heading.level === 3 && 'pl-3',
            activeId === heading.id ? 'text-primary' : 'text-muted-foreground'
          )}
        >
          {heading.text}
        </a>
      ))}
    </nav>
  )
}
