import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Textarea } from '@/components/ui/textarea'
import { getChannels } from '@/features/channels/api'

import { getChannelModelDescriptions, saveChannelModelDescription } from '../api'

export function ModelDescriptions() {
  const { t } = useTranslation()
  const client = useQueryClient()
  const [channelId, setChannelId] = useState('')
  const [modelName, setModelName] = useState('')
  const [description, setDescription] = useState('')
  const rows = useQuery({ queryKey: ['model-descriptions'], queryFn: getChannelModelDescriptions })
  const channels = useQuery({ queryKey: ['channels', 'description-editor'], queryFn: () => getChannels({ page_size: 1000 }) })
  const save = useMutation({
    mutationFn: saveChannelModelDescription,
    onSuccess: () => { void client.invalidateQueries({ queryKey: ['model-descriptions'] }); setDescription('') },
  })
  return (
    <div className='space-y-4'>
      <div className='grid gap-3 rounded-lg border p-4 md:grid-cols-4'>
        <select aria-label={t('Channel')} className='h-9 rounded-md border bg-background px-3 text-sm' value={channelId} onChange={(e) => setChannelId(e.target.value)}>
          <option value=''>{t('Select channel')}</option>
          {channels.data?.data?.items?.map((channel) => <option key={channel.id} value={channel.id}>{channel.name} (#{channel.id})</option>)}
        </select>
        <Input placeholder={t('Model name')} value={modelName} onChange={(e) => setModelName(e.target.value)} />
        <Textarea className='md:col-span-2' placeholder={t('Model description')} value={description} onChange={(e) => setDescription(e.target.value)} />
        <Button className='md:col-start-4' disabled={!channelId || !modelName.trim() || save.isPending} onClick={() => save.mutate({ channel_id: Number(channelId), model_name: modelName.trim(), description, enabled: true })}>{t('Save')}</Button>
      </div>
      <div className='divide-y rounded-lg border'>
        {rows.data?.data?.map((row) => <div className='grid gap-2 p-4 md:grid-cols-[180px_220px_1fr]' key={row.id}><span>#{row.channel_id}</span><span className='font-mono'>{row.model_name}</span><span>{row.description}</span></div>)}
      </div>
    </div>
  )
}
