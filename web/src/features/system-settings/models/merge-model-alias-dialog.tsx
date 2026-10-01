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
import { toast } from 'sonner'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Field, FieldDescription, FieldLabel } from '@/components/ui/field'
import { Input } from '@/components/ui/input'

export type ResolutionAliasConfig = {
  default_resolution?: string
  resolutions: Record<string, string>
}

type MergeModelAliasDialogProps = {
  open: boolean
  targets: string[]
  onOpenChange: (open: boolean) => void
  onConfirm: (aliasName: string, config: ResolutionAliasConfig) => void
}

const RESOLUTION_SUFFIX_PATTERN = /-(\d+p|\d+k(?:-pro)?)$/i

function guessResolution(modelName: string): string {
  const match = RESOLUTION_SUFFIX_PATTERN.exec(modelName)
  return match ? match[1].toLowerCase() : ''
}

function guessAliasName(modelName: string): string {
  const match = RESOLUTION_SUFFIX_PATTERN.exec(modelName)
  return match ? modelName.slice(0, match.index) : modelName
}

/**
 * 把选中的多个分辨率变体模型合并成一个别名模型：
 * 客户端只请求别名 + resolution，平台按档位解析到具体变体。
 */
export function MergeModelAliasDialog(props: MergeModelAliasDialogProps) {
  const { t } = useTranslation()
  const [aliasName, setAliasName] = useState('')
  const [defaultResolution, setDefaultResolution] = useState('720p')
  const [rows, setRows] = useState<
    Array<{ model: string; resolution: string }>
  >([])

  useEffect(() => {
    if (!props.open) return
    const nextRows = props.targets.map((model) => ({
      model,
      resolution: guessResolution(model),
    }))
    setRows(nextRows)
    setAliasName(
      props.targets.length > 0 ? guessAliasName(props.targets[0]) : ''
    )
    const labels = nextRows.map((row) => row.resolution).filter(Boolean)
    setDefaultResolution(labels.includes('720p') ? '720p' : labels[0] || '720p')
  }, [props.open, props.targets])

  const handleConfirm = () => {
    const resolutions: Record<string, string> = {}
    for (const row of rows) {
      const label = row.resolution.trim().toLowerCase()
      if (label === '') continue
      resolutions[label] = row.model
    }
    const trimmedAlias = aliasName.trim()
    if (trimmedAlias === '' || Object.keys(resolutions).length === 0) {
      toast.error(t('Alias name and at least one resolution are required'))
      return
    }
    const normalizedDefault = defaultResolution.trim().toLowerCase()
    if (!(normalizedDefault in resolutions)) {
      toast.error(t('Default resolution must be one of the merged resolutions'))
      return
    }
    props.onConfirm(trimmedAlias, {
      default_resolution: normalizedDefault,
      resolutions,
    })
  }

  return (
    <Dialog
      open={props.open}
      onOpenChange={props.onOpenChange}
      title={t('Merge into one model')}
      description={t(
        'Clients request the alias with a resolution; the matching variant is used for billing and forwarding.'
      )}
      contentClassName='sm:max-w-2xl'
      footer={
        <div className='flex justify-end gap-2'>
          <Button
            type='button'
            variant='outline'
            onClick={() => props.onOpenChange(false)}
          >
            {t('Cancel')}
          </Button>
          <Button type='button' onClick={handleConfirm}>
            {t('Merge')}
          </Button>
        </div>
      }
    >
      <div className='space-y-4'>
        <div className='grid gap-3 sm:grid-cols-2'>
          <Field>
            <FieldLabel>{t('Alias model name')}</FieldLabel>
            <Input
              value={aliasName}
              onChange={(event) => setAliasName(event.target.value)}
              placeholder='MiniMaxH3'
            />
          </Field>
          <Field>
            <FieldLabel>{t('Default resolution')}</FieldLabel>
            <Input
              value={defaultResolution}
              onChange={(event) => setDefaultResolution(event.target.value)}
              placeholder='720p'
            />
          </Field>
        </div>

        <div className='space-y-2'>
          <FieldLabel>{t('Resolution variants')}</FieldLabel>
          {rows.map((row, index) => (
            <div
              key={row.model}
              className='grid grid-cols-[minmax(0,1fr)_minmax(0,2fr)] items-center gap-2'
            >
              <Input
                value={row.resolution}
                placeholder='720p'
                onChange={(event) =>
                  setRows((previous) =>
                    previous.map((item, itemIndex) =>
                      itemIndex === index
                        ? { ...item, resolution: event.target.value }
                        : item
                    )
                  )
                }
              />
              <span className='text-muted-foreground truncate font-mono text-xs'>
                {row.model}
              </span>
            </div>
          ))}
        </div>

        <FieldDescription>
          {t(
            'Confirm, then save the pricing page to apply. Only the selected models are merged.'
          )}
        </FieldDescription>
      </div>
    </Dialog>
  )
}
