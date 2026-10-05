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
import { Link, useLocation } from '@tanstack/react-router'
import { ArrowLeft, ArrowRight, BookOpen, Menu } from 'lucide-react'
import { useEffect, useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { PublicLayout } from '@/components/layout'
import { Button } from '@/components/ui/button'
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet'
import { useGatewayBaseUrl } from '@/features/group-status/lib/use-base-url'
import { useSystemConfig } from '@/hooks/use-system-config'

import { DocArticle } from './components/doc-article'
import { DocNavigation } from './components/doc-navigation'
import { DocSearch } from './components/doc-search'
import { DocToc } from './components/doc-toc'
import { DOC_CONTENT } from './lib/content'
import { applyDocPlaceholders, parseDoc } from './lib/markdown'
import {
  DEFAULT_DOC_SLUG,
  DOC_SECTIONS,
  findDocPage,
  WORKBENCH_URL,
} from './pages'

export function Docs(props: { slug?: string }) {
  const { t } = useTranslation()
  const { systemName } = useSystemConfig()
  const baseUrl = useGatewayBaseUrl()
  const location = useLocation()
  const [menuOpen, setMenuOpen] = useState(false)
  const current = props.slug ? findDocPage(props.slug) : null
  const parsed = useMemo(
    () =>
      parseDoc(
        applyDocPlaceholders(
          DOC_CONTENT[props.slug ?? DEFAULT_DOC_SLUG] ?? '',
          { baseUrl, siteName: systemName, workbenchUrl: WORKBENCH_URL }
        )
      ),
    [baseUrl, props.slug, systemName]
  )

  useEffect(() => {
    if (location.hash) {
      document
        .getElementById(decodeURIComponent(location.hash))
        ?.scrollIntoView()
    }
  }, [location.hash, props.slug, parsed])

  return (
    <PublicLayout showMainContainer={false}>
      <div className='mx-auto max-w-[1440px] px-4 pt-24 pb-16 sm:px-6'>
        <div className='mb-6 flex items-center gap-3 border-b pb-4'>
          <Button
            variant='outline'
            size='icon'
            className='lg:hidden'
            aria-label={t('Documentation navigation')}
            onClick={() => setMenuOpen(true)}
          >
            <Menu />
          </Button>
          <div className='w-full max-w-sm'>
            <DocSearch />
          </div>
        </div>
        <Sheet open={menuOpen} onOpenChange={setMenuOpen}>
          <SheetContent side='left' className='overflow-y-auto'>
            <SheetHeader>
              <SheetTitle>{t('Docs')}</SheetTitle>
            </SheetHeader>
            <div className='px-5 pb-6'>
              <DocNavigation
                slug={props.slug}
                onNavigate={() => setMenuOpen(false)}
              />
            </div>
          </SheetContent>
        </Sheet>
        <div className='grid items-start gap-10 lg:grid-cols-[220px_minmax(0,1fr)] xl:grid-cols-[220px_minmax(0,1fr)_190px]'>
          <aside className='sticky top-24 hidden max-h-[calc(100svh-7rem)] overflow-y-auto pr-2 lg:block'>
            <Link
              to='/docs'
              className='mb-4 flex items-center gap-2 text-sm font-semibold'
            >
              <BookOpen className='size-4' />
              {t('Docs')}
            </Link>
            <DocNavigation slug={props.slug} />
          </aside>
          <main className='min-w-0'>
            {current ? (
              <>
                <div className='text-muted-foreground mb-3 text-xs'>
                  {current.section?.title}
                </div>
                <h1 className='text-3xl font-semibold'>{current.page.title}</h1>
                <p className='text-muted-foreground mt-3 mb-8 text-sm leading-6'>
                  {current.page.description}
                </p>
                <DocArticle blocks={parsed.blocks} />
                <nav
                  aria-label={t('Documentation navigation')}
                  className='mt-12 grid grid-cols-2 gap-4 border-t pt-6'
                >
                  {current.previous ? (
                    <Link
                      to='/docs/$slug'
                      params={{ slug: current.previous.slug }}
                      className='hover:text-primary min-w-0 text-sm'
                    >
                      <span className='text-muted-foreground mb-2 flex items-center gap-2 text-xs'>
                        <ArrowLeft className='size-3' />
                        {t('Previous')}
                      </span>
                      {current.previous.title}
                    </Link>
                  ) : (
                    <div />
                  )}
                  {current.next && (
                    <Link
                      to='/docs/$slug'
                      params={{ slug: current.next.slug }}
                      className='hover:text-primary min-w-0 text-right text-sm'
                    >
                      <span className='text-muted-foreground mb-2 flex items-center justify-end gap-2 text-xs'>
                        {t('Next')}
                        <ArrowRight className='size-3' />
                      </span>
                      {current.next.title}
                    </Link>
                  )}
                </nav>
              </>
            ) : (
              <>
                <h1 className='text-3xl font-semibold'>{t('Docs')}</h1>
                <p className='text-muted-foreground mt-3 text-sm'>
                  {systemName}
                </p>
                <div className='my-8 flex flex-wrap gap-3'>
                  <Button
                    render={
                      <Link
                        to='/docs/$slug'
                        params={{ slug: 'introduction' }}
                      />
                    }
                  >
                    <BookOpen className='size-4' />
                    {t('Quick Start')}
                  </Button>
                  <Button
                    variant='outline'
                    render={
                      <Link
                        to='/docs/$slug'
                        params={{ slug: 'api-overview' }}
                      />
                    }
                  >
                    <ArrowRight className='size-4' />
                    {t('API reference')}
                  </Button>
                </div>
                {DOC_SECTIONS.map((section) => (
                  <section key={section.id} className='border-t py-6'>
                    <h2 className='mb-4 flex items-center gap-2 text-base font-semibold'>
                      <section.icon className='size-4' />
                      {section.title}
                    </h2>
                    <div className='grid gap-2 sm:grid-cols-2'>
                      {section.pages.map((page) => (
                        <Link
                          key={page.slug}
                          to='/docs/$slug'
                          params={{ slug: page.slug }}
                          className='hover:bg-muted flex items-start gap-3 rounded-lg p-3 transition-colors'
                        >
                          <div className='min-w-0 flex-1'>
                            <span className='text-sm font-medium'>
                              {page.title}
                            </span>
                            <p className='text-muted-foreground mt-1 text-xs leading-5'>
                              {page.description}
                            </p>
                          </div>
                          <ArrowRight className='text-muted-foreground mt-1 size-4 shrink-0' />
                        </Link>
                      ))}
                    </div>
                  </section>
                ))}
              </>
            )}
          </main>
          {current && (
            <aside className='sticky top-24 hidden max-h-[calc(100svh-7rem)] overflow-y-auto xl:block'>
              <DocToc headings={parsed.headings} />
            </aside>
          )}
        </div>
      </div>
    </PublicLayout>
  )
}
