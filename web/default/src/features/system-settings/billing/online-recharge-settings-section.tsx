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
import { zodResolver } from '@hookform/resolvers/zod'
import type { Resolver } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import * as z from 'zod'

import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from '@/components/ui/form'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'

import { FormDirtyIndicator } from '../components/form-dirty-indicator'
import { FormNavigationGuard } from '../components/form-navigation-guard'
import {
  SettingsForm,
  SettingsSwitchContent,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useSettingsForm } from '../hooks/use-settings-form'
import { useUpdateOption } from '../hooks/use-update-option'

const createOnlineRechargeSchema = (t: (key: string) => string) =>
  z.object({
    OnlineRechargeEnabled: z.boolean(),
    OnlineRechargeUrl: z
      .string()
      .trim()
      .url(t('Must be a valid URL'))
      .refine((value) => /^https?:\/\//i.test(value), t('Must be a valid URL')),
  })

type OnlineRechargeFormValues = z.infer<
  ReturnType<typeof createOnlineRechargeSchema>
>

type OnlineRechargeSettingsSectionProps = {
  defaultValues: OnlineRechargeFormValues
}

const UPDATE_ORDER: (keyof OnlineRechargeFormValues)[] = [
  'OnlineRechargeUrl',
  'OnlineRechargeEnabled',
]

export function OnlineRechargeSettingsSection(
  props: OnlineRechargeSettingsSectionProps
) {
  const { t } = useTranslation()
  const updateOption = useUpdateOption()
  const schema = createOnlineRechargeSchema(t)

  const { form, handleSubmit, isDirty, isSubmitting } =
    useSettingsForm<OnlineRechargeFormValues>({
      resolver: zodResolver(schema) as Resolver<
        OnlineRechargeFormValues,
        unknown,
        OnlineRechargeFormValues
      >,
      defaultValues: props.defaultValues,
      onSubmit: async (data, changedFields) => {
        for (const key of UPDATE_ORDER) {
          if (!Object.hasOwn(changedFields, key)) {
            continue
          }
          await updateOption.mutateAsync({ key, value: data[key] })
        }
      },
    })

  return (
    <SettingsSection title={t('Online Recharge')}>
      <FormNavigationGuard when={isDirty} />
      <Form {...form}>
        <SettingsForm onSubmit={handleSubmit}>
          <SettingsPageFormActions
            onSave={handleSubmit}
            isSaving={updateOption.isPending || isSubmitting}
          />
          <FormDirtyIndicator isDirty={isDirty} />

          <FormField
            control={form.control}
            name='OnlineRechargeEnabled'
            render={({ field }) => (
              <SettingsSwitchItem>
                <SettingsSwitchContent>
                  <FormLabel>{t('Enable Online Recharge')}</FormLabel>
                  <FormDescription>
                    {t(
                      'Show the online recharge page below Wallet in the personal sidebar.'
                    )}
                  </FormDescription>
                </SettingsSwitchContent>
                <FormControl>
                  <Switch
                    checked={field.value}
                    onCheckedChange={field.onChange}
                    disabled={updateOption.isPending}
                  />
                </FormControl>
              </SettingsSwitchItem>
            )}
          />

          <FormField
            control={form.control}
            name='OnlineRechargeUrl'
            render={({ field }) => (
              <FormItem className='lg:col-span-2'>
                <FormLabel>{t('Online Recharge URL')}</FormLabel>
                <FormControl>
                  <Input
                    type='url'
                    placeholder='https://pay.example.com/shop'
                    {...field}
                  />
                </FormControl>
                <FormDescription>
                  {t(
                    'This URL is embedded as an iframe; HTTPS is recommended for production.'
                  )}
                </FormDescription>
                <FormMessage />
              </FormItem>
            )}
          />
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
