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
import {
  Code,
  MessagesSquare,
  RefreshCw,
  Terminal,
  type LucideIcon,
} from 'lucide-react'
import {
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  type CSSProperties,
} from 'react'
import { useTranslation } from 'react-i18next'

import { GroupMultiplierBadge } from '@/components/group-badge'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { useGroupStatusBoard } from '@/features/group-status/api'
import { formatGroupRate } from '@/features/group-status/lib/format'
import {
  GROUP_HEALTH_COLOR,
  getGroupHealth,
} from '@/features/group-status/lib/status'
import type { GroupStatusItem } from '@/features/group-status/types'
import {
  formatLatency,
  formatThroughput,
} from '@/features/performance-metrics/lib/format'
import { useSystemConfig } from '@/hooks/use-system-config'
import { toIntlLocale } from '@/i18n/languages'
import { formatCompactNumber, formatTimestampRelative } from '@/lib/format'
import { cn } from '@/lib/utils'

const ROW_HEIGHT = 60
const PADDING_Y = 28
const MIN_CONTENT_HEIGHT = 300
const GATEWAY_RADIUS = 36
const NODE_DOT_RADIUS = 10
const CLIENT_SPACING = 78
const MAX_TOPOLOGY_GROUPS = 8
const PLACEHOLDER_ROWS = [0, 1, 2, 3, 4, 5]
const PREVIEW_MODEL_COUNT = 3

const CLIENTS: { key: string; label: string; icon: LucideIcon }[] = [
  { key: 'chat', label: 'Chat clients', icon: MessagesSquare },
  { key: 'ide', label: 'IDE plugins', icon: Terminal },
  { key: 'sdk', label: 'SDK calls', icon: Code },
]

/** A cubic link that leaves and enters horizontally, like a wiring diagram. */
function linkPath(x1: number, y1: number, x2: number, y2: number): string {
  const bend = (x2 - x1) * 0.5
  return `M ${x1} ${y1} C ${x1 + bend} ${y1}, ${x2 - bend} ${y2}, ${x2} ${y2}`
}

/** Dashes move faster on busier groups: ~4s at one request per hour, ~1s at thousands. */
function getFlowDuration(group: GroupStatusItem): number {
  const recent = group.status?.recent
  if (!recent || recent.hours <= 0) return 4
  const perHour = recent.request_count / recent.hours
  return Math.min(4, Math.max(1, 4 - Math.log10(perHour + 1) * 0.9))
}

function UpdatedAgo(props: { updatedAt: number }) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const [, setTick] = useState(0)

  useEffect(() => {
    const timer = window.setInterval(() => setTick((tick) => tick + 1), 5000)
    return () => window.clearInterval(timer)
  }, [])

  if (!props.updatedAt) return null
  return (
    <span className='tabular-nums'>
      {t('Updated {{time}}', {
        time: formatTimestampRelative(props.updatedAt, 'milliseconds', locale),
      })}
    </span>
  )
}

function GroupNodeText(props: { group: GroupStatusItem; locale?: string }) {
  const { t } = useTranslation()
  const group = props.group
  const status = group.status
  const previewModels = group.models.slice(0, PREVIEW_MODEL_COUNT)
  const hiddenModels = group.models.length - previewModels.length
  const modelLine =
    previewModels.join(' · ') + (hiddenModels > 0 ? ` +${hiddenModels}` : '')
  const metricsLine = status
    ? [
        `${status.recent.hours}h ${formatGroupRate(status.recent.success_rate, props.locale)}`,
        formatLatency(status.avg_latency_ms),
        formatThroughput(status.avg_tps),
        `24h ${formatCompactNumber(status.request_count, props.locale)}`,
      ].join(' · ')
    : t('No traffic in the last 24 hours')

  return (
    <span className='min-w-0 flex-1'>
      <span className='flex min-w-0 items-center gap-2'>
        <span className='truncate text-sm font-semibold'>{group.name}</span>
        <GroupMultiplierBadge
          ratio={group.ratio}
          className='h-[18px] min-w-0 shrink-0 px-1.5 text-[11px]'
        />
      </span>
      <span className='text-muted-foreground block truncate font-mono text-[11px] leading-4'>
        {modelLine || t('No models')}
      </span>
      <span className='text-muted-foreground/75 block truncate font-mono text-[10.5px] leading-4 tabular-nums'>
        {metricsLine}
      </span>
    </span>
  )
}

function HealthDot(props: { group: GroupStatusItem; className?: string }) {
  const health = getGroupHealth(props.group)
  const color = GROUP_HEALTH_COLOR[health]
  return (
    <span
      className={cn(
        'relative flex size-5 shrink-0 items-center justify-center',
        props.className
      )}
    >
      <span className={cn('absolute inset-0 rounded-full', color.ring)} />
      {health !== 'idle' && (
        <span
          aria-hidden
          className={cn(
            'topology-ripple absolute inset-0 rounded-full',
            color.ring
          )}
        />
      )}
      <span className={cn('relative size-2.5 rounded-full', color.dot)} />
    </span>
  )
}

function scrollToGroupBoard(event: React.MouseEvent<HTMLAnchorElement>) {
  event.preventDefault()
  document
    .getElementById('group-status')
    ?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

type LiveTopologyProps = {
  className?: string
}

export function LiveTopology(props: LiveTopologyProps) {
  const { t, i18n } = useTranslation()
  const locale = toIntlLocale(i18n.resolvedLanguage || i18n.language)
  const { systemName, logo } = useSystemConfig()
  const board = useGroupStatusBoard()
  const canvasRef = useRef<HTMLDivElement>(null)
  const [width, setWidth] = useState(0)
  const [activeName, setActiveName] = useState<string | null>(null)

  // Links are drawn in pixels, so the canvas width drives the whole layout.
  useLayoutEffect(() => {
    const canvas = canvasRef.current
    if (!canvas) return
    setWidth(canvas.getBoundingClientRect().width)
    const observer = new ResizeObserver(([entry]) => {
      setWidth(entry.contentRect.width)
    })
    observer.observe(canvas)
    return () => observer.disconnect()
  }, [])

  const groups = board.data?.groups ?? []
  const activeGroups = groups.filter((group) => group.status)
  const shownGroups = (activeGroups.length > 0 ? activeGroups : groups).slice(
    0,
    MAX_TOPOLOGY_GROUPS
  )
  const hiddenCount = groups.length - shownGroups.length
  const showPlaceholders = board.isLoading || !board.canView || board.isError
  const rowCount = showPlaceholders
    ? PLACEHOLDER_ROWS.length
    : shownGroups.length + (hiddenCount > 0 ? 1 : 0)

  const contentHeight = Math.max(rowCount * ROW_HEIGHT, MIN_CONTENT_HEIGHT)
  const height = contentHeight + PADDING_Y * 2
  const centerY = height / 2
  const rowTop = (height - rowCount * ROW_HEIGHT) / 2
  const clientX = width * 0.15
  const gatewayX = width * 0.37
  const groupX = width * 0.57
  const nodeWidth = Math.max(width - groupX + NODE_DOT_RADIUS - 12, 0)
  const rowCenter = (index: number) => rowTop + ROW_HEIGHT * (index + 0.5)

  let statusNote: string | null = null
  if (!board.canView) {
    statusNote = t('Sign in to view live group status.')
  } else if (board.isError) {
    statusNote = t('Live status is temporarily unavailable.')
  }

  return (
    <div className={cn('relative overflow-hidden border-y', props.className)}>
      <div
        aria-hidden
        className='pointer-events-none absolute inset-0 bg-[linear-gradient(to_right,var(--border)_1px,transparent_1px),linear-gradient(to_bottom,var(--border)_1px,transparent_1px)] [mask-image:radial-gradient(ellipse_70%_60%_at_45%_50%,black_25%,transparent_100%)] bg-[size:3rem_3rem] opacity-40'
      />

      <div className='border-border/50 relative flex flex-wrap items-center justify-between gap-2 border-b px-5 py-3 text-xs'>
        <span className='flex items-center gap-2 font-medium'>
          <span className='relative flex size-2'>
            <span className='absolute inset-0 animate-ping rounded-full bg-emerald-500/60' />
            <span className='relative size-2 rounded-full bg-emerald-500' />
          </span>
          {t('Live traffic')}
          <span className='text-muted-foreground font-normal'>
            {t('Last 24 hours · {{active}} of {{total}} groups active', {
              active: activeGroups.length,
              total: groups.length,
            })}
          </span>
        </span>
        <span className='text-muted-foreground flex items-center gap-2'>
          <UpdatedAgo updatedAt={board.dataUpdatedAt} />
          <Button
            type='button'
            variant='ghost'
            size='icon-sm'
            aria-label={t('Refresh')}
            disabled={!board.canView || board.isFetching}
            onClick={() => void board.refetch()}
          >
            <RefreshCw
              className={cn('size-3.5', board.isFetching && 'animate-spin')}
            />
          </Button>
        </span>
      </div>

      {/* Desktop: wiring diagram from clients through the gateway to groups. */}
      <div
        ref={canvasRef}
        className='relative hidden lg:block'
        style={{ height }}
      >
        {width > 0 && (
          <>
            <svg
              aria-hidden
              className='pointer-events-none absolute inset-0'
              width={width}
              height={height}
            >
              {CLIENTS.map((client, index) => {
                const y = centerY + (index - 1) * CLIENT_SPACING
                const d = linkPath(
                  clientX + 6,
                  y,
                  gatewayX - GATEWAY_RADIUS - 4,
                  centerY
                )
                return (
                  <g key={client.key}>
                    <path
                      d={d}
                      pathLength={1}
                      className='topology-draw stroke-border fill-none'
                      strokeWidth={1.25}
                    />
                    <path
                      d={d}
                      className='topology-flow stroke-primary fill-none'
                      strokeWidth={1.75}
                      strokeLinecap='round'
                      style={{ '--flow-duration': '2.2s' } as CSSProperties}
                    />
                  </g>
                )
              })}
              {!showPlaceholders &&
                shownGroups.map((group, index) => {
                  const health = getGroupHealth(group)
                  const d = linkPath(
                    gatewayX + GATEWAY_RADIUS + 4,
                    centerY,
                    groupX - NODE_DOT_RADIUS,
                    rowCenter(index)
                  )
                  const dimmed =
                    activeName !== null && activeName !== group.name
                  return (
                    <g
                      key={group.name}
                      className={cn(
                        'transition-opacity duration-300',
                        dimmed && 'opacity-15'
                      )}
                    >
                      <path
                        d={d}
                        pathLength={1}
                        className='topology-draw stroke-border fill-none'
                        strokeWidth={activeName === group.name ? 2 : 1.25}
                        style={{ animationDelay: `${index * 70}ms` }}
                      />
                      {health !== 'idle' && (
                        <path
                          d={d}
                          className={cn(
                            'topology-flow fill-none',
                            GROUP_HEALTH_COLOR[health].stroke
                          )}
                          strokeWidth={activeName === group.name ? 2.5 : 1.75}
                          strokeLinecap='round'
                          style={
                            {
                              '--flow-duration': `${getFlowDuration(group).toFixed(2)}s`,
                              animationDelay: `${index * 70}ms`,
                            } as CSSProperties
                          }
                        />
                      )}
                    </g>
                  )
                })}
              {!showPlaceholders && hiddenCount > 0 && (
                <path
                  d={linkPath(
                    gatewayX + GATEWAY_RADIUS + 4,
                    centerY,
                    groupX - NODE_DOT_RADIUS,
                    rowCenter(shownGroups.length)
                  )}
                  className='stroke-border fill-none'
                  strokeWidth={1.25}
                  strokeDasharray='3 5'
                />
              )}
            </svg>

            {CLIENTS.map((client, index) => {
              const Icon = client.icon
              return (
                <div
                  key={client.key}
                  className='landing-animate-fade-right absolute flex items-center gap-2.5'
                  style={{
                    top: centerY + (index - 1) * CLIENT_SPACING - 14,
                    right: width - clientX - 10,
                    animationDelay: `${index * 80}ms`,
                  }}
                >
                  <span className='text-muted-foreground flex items-center gap-1.5 text-xs font-medium whitespace-nowrap'>
                    <Icon className='size-3.5' />
                    {t(client.label)}
                  </span>
                  <span className='bg-primary/15 flex size-5 items-center justify-center rounded-full'>
                    <span className='bg-primary size-2 rounded-full' />
                  </span>
                </div>
              )
            })}

            <div
              className='landing-animate-scale-in absolute'
              style={{
                left: gatewayX - GATEWAY_RADIUS,
                top: centerY - GATEWAY_RADIUS,
              }}
            >
              <span
                aria-hidden
                className='topology-ripple bg-primary/20 absolute inset-0 rounded-full'
              />
              <span
                aria-hidden
                className='topology-ripple bg-primary/15 absolute inset-0 rounded-full'
                style={{ animationDelay: '1.4s' }}
              />
              <div
                className='border-primary/30 bg-background relative flex items-center justify-center rounded-full border shadow-[0_0_0_6px_color-mix(in_oklch,var(--primary)_10%,transparent)]'
                style={{
                  width: GATEWAY_RADIUS * 2,
                  height: GATEWAY_RADIUS * 2,
                }}
              >
                <img
                  src={logo}
                  alt=''
                  className='size-10 rounded-xl object-contain'
                />
              </div>
            </div>
            <div
              className='absolute w-40 -translate-x-1/2 text-center'
              style={{ left: gatewayX, top: centerY + GATEWAY_RADIUS + 14 }}
            >
              <div className='truncate text-base font-semibold tracking-tight'>
                {systemName}
              </div>
              <div className='text-muted-foreground text-xs'>
                {t('Unified AI gateway')}
              </div>
            </div>

            {showPlaceholders &&
              PLACEHOLDER_ROWS.map((slot) => (
                <div
                  key={slot}
                  className='absolute flex items-center gap-3'
                  style={{
                    left: groupX - NODE_DOT_RADIUS,
                    top: rowCenter(slot) - ROW_HEIGHT / 2,
                    width: nodeWidth,
                    height: ROW_HEIGHT,
                  }}
                >
                  <Skeleton className='size-5 shrink-0 rounded-full' />
                  <span className='flex flex-1 flex-col gap-1.5'>
                    <Skeleton className='h-3.5 w-32' />
                    <Skeleton className='h-3 w-56' />
                  </span>
                </div>
              ))}
            {statusNote && (
              <div
                className='bg-background/80 border-border/60 text-muted-foreground absolute max-w-xs rounded-xl border px-4 py-3 text-sm backdrop-blur'
                style={{ left: groupX + 24, top: centerY - 28 }}
              >
                {statusNote}
              </div>
            )}

            {!showPlaceholders &&
              shownGroups.map((group, index) => (
                <Link
                  key={group.name}
                  to='/pricing'
                  search={{ group: group.name }}
                  title={group.description || group.name}
                  onMouseEnter={() => setActiveName(group.name)}
                  onMouseLeave={() => setActiveName(null)}
                  onFocus={() => setActiveName(group.name)}
                  onBlur={() => setActiveName(null)}
                  className={cn(
                    'landing-animate-fade-left focus-visible:ring-ring/50 absolute flex items-center gap-3 rounded-xl pr-2 outline-none transition-opacity duration-300 focus-visible:ring-3',
                    activeName !== null &&
                      activeName !== group.name &&
                      'opacity-40'
                  )}
                  style={{
                    left: groupX - NODE_DOT_RADIUS,
                    top: rowCenter(index) - ROW_HEIGHT / 2,
                    width: nodeWidth,
                    height: ROW_HEIGHT,
                    animationDelay: `${index * 70 + 120}ms`,
                  }}
                >
                  <HealthDot group={group} />
                  <GroupNodeText group={group} locale={locale} />
                </Link>
              ))}
            {!showPlaceholders && hiddenCount > 0 && (
              <a
                href='#group-status'
                onClick={scrollToGroupBoard}
                className='landing-animate-fade-left text-muted-foreground hover:text-foreground focus-visible:ring-ring/50 absolute flex items-center gap-3 rounded-xl text-sm transition-colors outline-none focus-visible:ring-3'
                style={{
                  left: groupX - NODE_DOT_RADIUS,
                  top: rowCenter(shownGroups.length) - ROW_HEIGHT / 2,
                  height: ROW_HEIGHT,
                  animationDelay: `${shownGroups.length * 70 + 120}ms`,
                }}
              >
                <span className='border-muted-foreground/40 flex size-5 items-center justify-center rounded-full border border-dashed text-[10px] font-semibold'>
                  +
                </span>
                {t('View all {{count}} groups', { count: groups.length })}
              </a>
            )}
          </>
        )}
      </div>

      {/* Mobile: the same flow as a vertical pipeline. */}
      <div className='relative flex flex-col items-center gap-4 px-4 py-6 lg:hidden'>
        <div className='flex flex-wrap justify-center gap-2'>
          {CLIENTS.map((client) => {
            const Icon = client.icon
            return (
              <span
                key={client.key}
                className='border-border/60 bg-background/70 text-muted-foreground flex items-center gap-1.5 rounded-full border px-3 py-1 text-xs'
              >
                <Icon className='size-3.5' />
                {t(client.label)}
              </span>
            )
          })}
        </div>
        <svg aria-hidden width='2' height='28' className='overflow-visible'>
          <path
            d='M 1 0 L 1 28'
            className='topology-flow stroke-primary fill-none'
            strokeWidth={2}
          />
        </svg>
        <div className='flex items-center gap-3'>
          <span className='border-primary/30 bg-background flex size-12 items-center justify-center rounded-full border'>
            <img
              src={logo}
              alt=''
              className='size-8 rounded-lg object-contain'
            />
          </span>
          <span>
            <span className='block text-base font-semibold'>{systemName}</span>
            <span className='text-muted-foreground block text-xs'>
              {t('Unified AI gateway')}
            </span>
          </span>
        </div>
        <svg aria-hidden width='2' height='28' className='overflow-visible'>
          <path
            d='M 1 0 L 1 28'
            className='topology-flow fill-none stroke-emerald-500'
            strokeWidth={2}
          />
        </svg>
        <div className='flex w-full flex-col gap-1'>
          {showPlaceholders &&
            PLACEHOLDER_ROWS.slice(0, 4).map((slot) => (
              <Skeleton key={slot} className='h-14 w-full rounded-xl' />
            ))}
          {statusNote && (
            <p className='text-muted-foreground text-center text-sm'>
              {statusNote}
            </p>
          )}
          {!showPlaceholders &&
            shownGroups.map((group) => (
              <Link
                key={group.name}
                to='/pricing'
                search={{ group: group.name }}
                className='hover:bg-muted/40 flex items-center gap-3 rounded-xl px-2 py-2 transition-colors'
              >
                <HealthDot group={group} />
                <GroupNodeText group={group} locale={locale} />
              </Link>
            ))}
          {!showPlaceholders && hiddenCount > 0 && (
            <a
              href='#group-status'
              onClick={scrollToGroupBoard}
              className='text-primary px-2 py-2 text-center text-sm font-medium'
            >
              {t('View all {{count}} groups', { count: groups.length })}
            </a>
          )}
        </div>
      </div>
    </div>
  )
}
