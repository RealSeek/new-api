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
import { Link } from '@tanstack/react-router'
import { useTranslation } from 'react-i18next'

import { cn } from '@/lib/utils'

import { DOC_SECTIONS } from '../pages'

export function DocNavigation(props: {
  slug?: string
  onNavigate?: () => void
}) {
  const { t } = useTranslation()
  return (
    <nav aria-label={t('Docs')} className='space-y-6'>
      {DOC_SECTIONS.map((section) => (
        <div key={section.id}>
          <div className='mb-2 flex items-center gap-2 text-xs font-semibold'>
            <section.icon className='text-muted-foreground size-4' />
            {section.title}
          </div>
          <div className='space-y-1 border-l pl-3'>
            {section.pages.map((page) => (
              <Link
                key={page.slug}
                to='/docs/$slug'
                params={{ slug: page.slug }}
                onClick={props.onNavigate}
                aria-current={props.slug === page.slug ? 'page' : undefined}
                className={cn(
                  'block rounded-md px-2 py-1.5 text-sm transition-colors hover:bg-muted',
                  props.slug === page.slug
                    ? 'bg-primary/10 text-primary font-medium'
                    : 'text-muted-foreground'
                )}
              >
                {page.title}
              </Link>
            ))}
          </div>
        </div>
      ))}
    </nav>
  )
}
