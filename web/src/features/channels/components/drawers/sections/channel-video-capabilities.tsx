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
import { useState } from 'react'
import type { UseFormReturn } from 'react-hook-form'
import { useTranslation } from 'react-i18next'

import { ModelSelector } from '@/components/model-group-selector'
import {
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'

import type { ChannelFormValues } from '../../../lib/channel-form'
import type { VideoModelCapabilities } from '../../../types'

export function ChannelVideoCapabilities(props: {
  form: UseFormReturn<ChannelFormValues>
  models: string[]
  disabled: boolean
}) {
  const { t } = useTranslation()
  const [selectedModel, setSelectedModel] = useState('')
  const model = props.models.includes(selectedModel)
    ? selectedModel
    : props.models[0]
  const materials: {
    key: Exclude<keyof VideoModelCapabilities, 'face_supported'>
    label: string
    inputLabel: string
    max?: number
  }[] = [
    {
      key: 'reference_images',
      label: t('Reference images'),
      inputLabel: t('Reference images material limit'),
    },
    {
      key: 'first_last_frames',
      label: t('First and last frames'),
      inputLabel: t('First and last frames material limit'),
      max: 2,
    },
    {
      key: 'reference_videos',
      label: t('Reference videos'),
      inputLabel: t('Reference videos material limit'),
    },
    {
      key: 'reference_audios',
      label: t('Reference audio'),
      inputLabel: t('Reference audio material limit'),
    },
  ]
  if (!model) return null

  return (
    <FormField
      control={props.form.control}
      name='video_model_capabilities'
      render={({ field }) => {
        const configured = field.value?.[model]
        return (
          <FormItem className='border-border/60 rounded-lg border p-4'>
            <FormLabel>{t('Video input capabilities')}</FormLabel>
            <FormDescription>
              {t(
                'Configure support and material limits for this model on this channel. These are displayed in model details.'
              )}
            </FormDescription>
            <ModelSelector
              selectedModel={model}
              models={props.models.map((name) => ({
                label: name,
                value: name,
              }))}
              onModelChange={setSelectedModel}
              disabled={props.disabled}
            />
            <div className='flex items-center justify-between gap-3'>
              <span className='text-sm'>
                {t('Configure video capabilities')}
              </span>
              <Switch
                aria-label={t('Configure video capabilities')}
                disabled={props.disabled}
                checked={configured !== undefined}
                onCheckedChange={(checked) => {
                  const next = { ...field.value }
                  if (checked) {
                    next[model] = {
                      reference_images: 0,
                      first_last_frames: 0,
                      reference_videos: 0,
                      reference_audios: 0,
                    }
                  } else {
                    delete next[model]
                  }
                  field.onChange(next)
                }}
              />
            </div>
            {configured ? (
              <div className='space-y-3'>
                {materials.map((material) => (
                  <div key={material.key} className='flex items-center gap-3'>
                    <span className='min-w-0 flex-1 text-sm'>
                      {material.label}
                    </span>
                    <Switch
                      aria-label={material.label}
                      disabled={props.disabled}
                      checked={configured[material.key] > 0}
                      onCheckedChange={(checked) =>
                        field.onChange({
                          ...field.value,
                          [model]: {
                            ...configured,
                            [material.key]: checked ? (material.max ?? 1) : 0,
                          },
                        })
                      }
                    />
                    <Input
                      className='w-24'
                      type='number'
                      min={0}
                      max={material.max}
                      step={1}
                      aria-label={material.inputLabel}
                      disabled={props.disabled}
                      value={configured[material.key]}
                      onChange={(event) =>
                        field.onChange({
                          ...field.value,
                          [model]: {
                            ...configured,
                            [material.key]: Number(event.target.value),
                          },
                        })
                      }
                    />
                  </div>
                ))}
                <div className='flex items-center gap-3'>
                  <span className='min-w-0 flex-1 text-sm'>
                    {t('Face support')}
                  </span>
                  {configured.face_supported === undefined && (
                    <span className='text-muted-foreground text-xs'>
                      {t('Not configured')}
                    </span>
                  )}
                  <Switch
                    aria-label={t('Face support')}
                    disabled={props.disabled}
                    checked={configured.face_supported === true}
                    onCheckedChange={(checked) =>
                      field.onChange({
                        ...field.value,
                        [model]: { ...configured, face_supported: checked },
                      })
                    }
                  />
                </div>
              </div>
            ) : (
              <FormDescription>{t('Not configured')}</FormDescription>
            )}
            <FormMessage />
          </FormItem>
        )
      }}
    />
  )
}
