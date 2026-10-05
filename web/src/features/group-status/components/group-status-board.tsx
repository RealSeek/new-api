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
import { Search } from 'lucide-react'
import { AnimatePresence, motion, useReducedMotion } from 'motion/react'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { EmptyState } from '@/components/empty-state'
import {
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
} from '@/components/ui/input-group'
import { Skeleton } from '@/components/ui/skeleton'
import { Switch } from '@/components/ui/switch'
import { ToggleGroup, ToggleGroupItem } from '@/components/ui/toggle-group'
import { MOTION_TRANSITION } from '@/lib/motion'

import { getGroupCategory } from '../lib/status'
import type { GroupCategory, GroupStatusItem } from '../types'
import { GroupStatusCard } from './group-status-card'

type CategoryFilter = GroupCategory | 'all'
type SortKey = 'traffic' | 'availability' | 'ratio'

const CATEGORY_ORDER: GroupCategory[] = [
  'claude',
  'openai',
  'gemini',
  'china',
  'media',
  'other',
]

/** Brand names stay as-is; the remaining labels are i18n keys. */
const CATEGORY_LABEL: Record<GroupCategory, { text: string; brand: boolean }> =
  {
    claude: { text: 'Claude', brand: true },
    openai: { text: 'OpenAI', brand: true },
    gemini: { text: 'Gemini', brand: true },
    china: { text: 'Chinese models', brand: false },
    media: { text: 'Image & video', brand: false },
    other: { text: 'Other', brand: false },
  }

const SKELETON_CARDS = Array.from({ length: 6 }, (_, slot) => slot)

type GroupStatusBoardProps = {
  groups: GroupStatusItem[]
  isLoading: boolean
}

export function GroupStatusBoard(props: GroupStatusBoardProps) {
  const { t } = useTranslation()
  const shouldReduceMotion = useReducedMotion()
  const [category, setCategory] = useState<CategoryFilter>('all')
  const [sortKey, setSortKey] = useState<SortKey>('traffic')
  const [search, setSearch] = useState('')
  const [hideIdle, setHideIdle] = useState(false)

  const categories = useMemo(
    () =>
      new Map(
        props.groups.map((group) => [group.name, getGroupCategory(group)])
      ),
    [props.groups]
  )
  const categoryCounts = useMemo(() => {
    const counts = new Map<GroupCategory, number>()
    for (const value of categories.values()) {
      counts.set(value, (counts.get(value) ?? 0) + 1)
    }
    return counts
  }, [categories])

  const visibleGroups = useMemo(() => {
    const query = search.trim().toLowerCase()
    const filtered = props.groups.filter((group) => {
      if (category !== 'all' && categories.get(group.name) !== category) {
        return false
      }
      if (hideIdle && !group.status) return false
      if (!query) return true
      return (
        group.name.toLowerCase().includes(query) ||
        group.description.toLowerCase().includes(query) ||
        group.models.some((modelName) =>
          modelName.toLowerCase().includes(query)
        )
      )
    })
    if (sortKey === 'availability') {
      // Idle groups have no rate to rank by, so they go last.
      return filtered.sort(
        (a, b) =>
          (b.status?.success_rate ?? -1) - (a.status?.success_rate ?? -1)
      )
    }
    if (sortKey === 'ratio') {
      return filtered.sort((a, b) => a.ratio - b.ratio)
    }
    // The server already orders groups by 24h traffic.
    return filtered
  }, [categories, category, hideIdle, props.groups, search, sortKey])

  const activeCategories = CATEGORY_ORDER.filter(
    (value) => (categoryCounts.get(value) ?? 0) > 0
  )

  return (
    <div className='flex flex-col gap-5'>
      <div className='flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between'>
        <ToggleGroup
          value={[category]}
          onValueChange={(values) => {
            if (values.length > 0) setCategory(values[0] as CategoryFilter)
          }}
          variant='outline'
          size='sm'
          spacing={1}
          aria-label={t('Filter groups by category')}
          className='flex-wrap'
        >
          <ToggleGroupItem value='all' className='rounded-full px-3'>
            {t('All')}
            <span className='text-muted-foreground ml-1 tabular-nums'>
              {props.groups.length}
            </span>
          </ToggleGroupItem>
          {activeCategories.map((value) => {
            const label = CATEGORY_LABEL[value]
            return (
              <ToggleGroupItem
                key={value}
                value={value}
                className='rounded-full px-3'
              >
                {label.brand ? label.text : t(label.text)}
                <span className='text-muted-foreground ml-1 tabular-nums'>
                  {categoryCounts.get(value)}
                </span>
              </ToggleGroupItem>
            )
          })}
        </ToggleGroup>

        <div className='flex flex-wrap items-center gap-3'>
          <InputGroup className='h-8 w-full sm:w-60'>
            <InputGroupAddon>
              <Search />
            </InputGroupAddon>
            <InputGroupInput
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder={t('Search groups or models')}
              aria-label={t('Search groups or models')}
            />
          </InputGroup>
          <ToggleGroup
            value={[sortKey]}
            onValueChange={(values) => {
              if (values.length > 0) setSortKey(values[0] as SortKey)
            }}
            variant='outline'
            size='sm'
            aria-label={t('Sort groups')}
          >
            <ToggleGroupItem value='traffic'>{t('Traffic')}</ToggleGroupItem>
            <ToggleGroupItem value='availability'>
              {t('Availability')}
            </ToggleGroupItem>
            <ToggleGroupItem value='ratio'>{t('Ratio')}</ToggleGroupItem>
          </ToggleGroup>
          <label className='text-muted-foreground flex cursor-pointer items-center gap-2 text-xs'>
            <Switch
              size='sm'
              checked={hideIdle}
              onCheckedChange={setHideIdle}
            />
            {t('Hide idle groups')}
          </label>
        </div>
      </div>

      {props.isLoading && (
        <div className='grid gap-4 sm:grid-cols-2 xl:grid-cols-3'>
          {SKELETON_CARDS.map((slot) => (
            <Skeleton key={slot} className='h-[188px] rounded-lg' />
          ))}
        </div>
      )}

      {!props.isLoading && visibleGroups.length === 0 && (
        <EmptyState
          icon={Search}
          title={t('No matching groups')}
          description={t('Try another keyword or category.')}
        />
      )}

      {!props.isLoading && visibleGroups.length > 0 && (
        <motion.div layout className='grid gap-4 sm:grid-cols-2 xl:grid-cols-3'>
          <AnimatePresence initial={false} mode='popLayout'>
            {visibleGroups.map((group) => (
              <motion.div
                key={group.name}
                layout={!shouldReduceMotion}
                initial={shouldReduceMotion ? false : { opacity: 0, y: 12 }}
                animate={{ opacity: 1, y: 0 }}
                exit={
                  shouldReduceMotion ? undefined : { opacity: 0, scale: 0.97 }
                }
                transition={MOTION_TRANSITION.default}
              >
                <GroupStatusCard group={group} className='h-full' />
              </motion.div>
            ))}
          </AnimatePresence>
        </motion.div>
      )}
    </div>
  )
}
