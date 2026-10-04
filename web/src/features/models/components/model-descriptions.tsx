import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { ComboboxInput } from '@/components/ui/combobox-input'
import { Textarea } from '@/components/ui/textarea'
import { getChannels } from '@/features/channels/api'

import {
  getChannelModelDescriptions,
  saveChannelModelDescription,
} from '../api'

export function ModelDescriptions() {
  const { t } = useTranslation()
  const client = useQueryClient()
  const [channelId, setChannelId] = useState('')
  const [modelName, setModelName] = useState('')
  const [description, setDescription] = useState('')
  const rows = useQuery({
    queryKey: ['model-descriptions'],
    queryFn: getChannelModelDescriptions,
  })
  const channels = useQuery({
    queryKey: ['channels', 'description-editor'],
    queryFn: () => getChannels({ page_size: 1000 }),
  })
  const channelItems = channels.data?.data?.items
  const channelOptions = useMemo(
    () =>
      (channelItems ?? []).map((channel) => ({
        value: String(channel.id),
        label: `${channel.name} (#${channel.id})`,
      })),
    [channelItems]
  )
  // Only models the selected channel actually exposes can carry a channel description.
  const modelOptions = useMemo(() => {
    const channel = channelItems?.find((item) => String(item.id) === channelId)
    const names = (channel?.models ?? '')
      .split(',')
      .map((name) => name.trim())
      .filter(Boolean)
    return [...new Set(names)].map((name) => ({ value: name, label: name }))
  }, [channelItems, channelId])
  const save = useMutation({
    mutationFn: saveChannelModelDescription,
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: ['model-descriptions'] })
      setDescription('')
    },
  })
  return (
    <div className='space-y-4'>
      <div className='grid gap-3 rounded-lg border p-4 md:grid-cols-4'>
        <ComboboxInput
          aria-label={t('Channel')}
          options={channelOptions}
          value={channelId}
          onValueChange={(value) => {
            setChannelId(value)
            setModelName('')
          }}
          placeholder={t('Select channel')}
          emptyText={t('No matching items')}
        />
        <ComboboxInput
          aria-label={t('Model name')}
          options={modelOptions}
          value={modelName}
          onValueChange={setModelName}
          placeholder={t('Select model')}
          emptyText={t('No matching items')}
          disabled={!channelId}
        />
        <Textarea
          className='md:col-span-2'
          aria-label={t('Model Description')}
          placeholder={t('Model Description')}
          value={description}
          onChange={(e) => setDescription(e.target.value)}
        />
        <Button
          className='md:col-start-4'
          disabled={!channelId || !modelName || save.isPending}
          onClick={() =>
            save.mutate({
              channel_id: Number(channelId),
              model_name: modelName,
              description,
              enabled: true,
            })
          }
        >
          {t('Save')}
        </Button>
      </div>
      <div className='divide-y rounded-lg border'>
        {rows.data?.data?.map((row) => (
          <div
            className='grid gap-2 p-4 md:grid-cols-[180px_220px_1fr]'
            key={row.id}
          >
            <span>#{row.channel_id}</span>
            <span className='font-mono'>{row.model_name}</span>
            <span>{row.description}</span>
          </div>
        ))}
      </div>
    </div>
  )
}
