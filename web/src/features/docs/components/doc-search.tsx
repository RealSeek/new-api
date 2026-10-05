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
import { useNavigate } from '@tanstack/react-router'
import { Search } from 'lucide-react'
import { useEffect, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import {
  Command,
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from '@/components/ui/command'

import { DOC_CONTENT } from '../lib/content'
import { parseDoc } from '../lib/markdown'
import { DOC_SECTIONS } from '../pages'

const SEARCH_SECTIONS = DOC_SECTIONS.map((section) => ({
  ...section,
  entries: section.pages.flatMap((page) => [
    { ...page, hash: '', label: page.title },
    ...parseDoc(DOC_CONTENT[page.slug]).headings.map((heading) => ({
      ...page,
      hash: heading.id,
      label: `${page.title} / ${heading.text}`,
    })),
  ]),
}))

export function DocSearch() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const [open, setOpen] = useState(false)

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if ((event.ctrlKey || event.metaKey) && event.key.toLowerCase() === 'k') {
        event.preventDefault()
        setOpen((current) => !current)
      }
    }
    document.addEventListener('keydown', onKeyDown)
    return () => document.removeEventListener('keydown', onKeyDown)
  }, [])

  return (
    <>
      <Button
        variant='outline'
        className='w-full justify-start'
        onClick={() => setOpen(true)}
      >
        <Search className='size-4' />
        {t('Search documentation')}
      </Button>
      <CommandDialog
        open={open}
        onOpenChange={setOpen}
        title={t('Search documentation')}
        description={t('Docs')}
      >
        <Command>
          <CommandInput
            placeholder={t('Search documentation')}
            aria-label={t('Search documentation')}
          />
          <CommandList>
            <CommandEmpty>{t('No results found.')}</CommandEmpty>
            {SEARCH_SECTIONS.map((section) => (
              <CommandGroup key={section.id} heading={section.title}>
                {section.entries.map((entry) => (
                  <CommandItem
                    key={`${entry.slug}#${entry.hash}`}
                    value={`${entry.label} ${entry.description} ${entry.slug} ${entry.hash}`}
                    onSelect={() => {
                      setOpen(false)
                      void navigate({
                        to: '/docs/$slug',
                        params: { slug: entry.slug },
                        hash: entry.hash,
                      })
                    }}
                  >
                    {entry.label}
                  </CommandItem>
                ))}
              </CommandGroup>
            ))}
          </CommandList>
        </Command>
      </CommandDialog>
    </>
  )
}
