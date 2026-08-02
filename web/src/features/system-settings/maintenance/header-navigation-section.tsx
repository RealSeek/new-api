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
import { Add01Icon, Delete02Icon, Link01Icon } from '@hugeicons/core-free-icons'
import { HugeiconsIcon } from '@hugeicons/react'
import { useEffect, useMemo } from 'react'
import { useFieldArray, useForm } from 'react-hook-form'
import { useTranslation } from 'react-i18next'
import * as z from 'zod'

import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from '@/components/ui/accordion'
import { Button } from '@/components/ui/button'
import {
  Empty,
  EmptyDescription,
  EmptyHeader,
  EmptyMedia,
  EmptyTitle,
} from '@/components/ui/empty'
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
import {
  INTERFACE_LANGUAGE_OPTIONS,
  normalizeInterfaceLanguage,
  type InterfaceLanguageCode,
} from '@/i18n/languages'
import {
  DEFAULT_HEADER_NAV_CUSTOM_LINKS,
  createEmptyHeaderNavCustomLinkTitles,
  serializeHeaderNavCustomLinks,
  type HeaderNavCustomLink,
} from '@/lib/header-nav-custom-links'

import {
  SettingsControlChildren,
  SettingsForm,
  SettingsSwitchContent,
  SettingsControlGroup,
  SettingsSwitchItem,
} from '../components/settings-form-layout'
import { SettingsPageFormActions } from '../components/settings-page-context'
import { SettingsSection } from '../components/settings-section'
import { useUpdateOption } from '../hooks/use-update-option'
import {
  HEADER_NAV_DEFAULT,
  type HeaderNavModulesConfig,
  serializeHeaderNavModules,
} from './config'

const createHeaderNavSchema = (
  t: (key: string) => string,
  currentLocale: InterfaceLanguageCode
) =>
  z.object({
    home: z.boolean(),
    console: z.boolean(),
    pricingEnabled: z.boolean(),
    pricingRequireAuth: z.boolean(),
    rankingsEnabled: z.boolean(),
    rankingsRequireAuth: z.boolean(),
    docs: z.boolean(),
    about: z.boolean(),
    customLinks: z.array(
      z
        .object({
          titles: z.object({
            zhCN: z.string().trim(),
            en: z.string().trim(),
            fr: z.string().trim(),
            ru: z.string().trim(),
            ja: z.string().trim(),
            vi: z.string().trim(),
            zhTW: z.string().trim(),
          }),
          url: z
            .string()
            .trim()
            .min(1, t('URL is required'))
            .url(t('Must be a valid URL'))
            .refine(
              (value) => /^https?:\/\//i.test(value),
              t('Must be a valid URL')
            ),
        })
        .superRefine((link, context) => {
          if (Object.values(link.titles).some((title) => title !== '')) return
          context.addIssue({
            code: 'custom',
            message: t('At least one language name is required'),
            path: ['titles', currentLocale],
          })
        })
    ),
  })

type HeaderNavFormValues = z.infer<ReturnType<typeof createHeaderNavSchema>>

type HeaderNavigationSectionProps = {
  config: HeaderNavModulesConfig
  initialSerialized: string
  customLinks: HeaderNavCustomLink[]
  initialCustomLinksSerialized: string
}

const toFormValues = (
  config: HeaderNavModulesConfig,
  customLinks: HeaderNavCustomLink[]
): HeaderNavFormValues => ({
  home:
    config.home === undefined ? HEADER_NAV_DEFAULT.home : Boolean(config.home),
  console:
    config.console === undefined
      ? HEADER_NAV_DEFAULT.console
      : Boolean(config.console),
  pricingEnabled:
    config.pricing?.enabled === undefined
      ? HEADER_NAV_DEFAULT.pricing.enabled
      : Boolean(config.pricing.enabled),
  pricingRequireAuth:
    config.pricing?.requireAuth === undefined
      ? HEADER_NAV_DEFAULT.pricing.requireAuth
      : Boolean(config.pricing.requireAuth),
  rankingsEnabled:
    config.rankings?.enabled === undefined
      ? HEADER_NAV_DEFAULT.rankings.enabled
      : Boolean(config.rankings.enabled),
  rankingsRequireAuth:
    config.rankings?.requireAuth === undefined
      ? HEADER_NAV_DEFAULT.rankings.requireAuth
      : Boolean(config.rankings.requireAuth),
  docs:
    config.docs === undefined ? HEADER_NAV_DEFAULT.docs : Boolean(config.docs),
  about:
    config.about === undefined
      ? HEADER_NAV_DEFAULT.about
      : Boolean(config.about),
  customLinks: customLinks.map((link) => ({
    ...link,
    titles: { ...link.titles },
  })),
})

export function HeaderNavigationSection(props: HeaderNavigationSectionProps) {
  const { t, i18n } = useTranslation()
  const updateOption = useUpdateOption()
  const currentLocale = normalizeInterfaceLanguage(
    i18n.resolvedLanguage ?? i18n.language
  ) as InterfaceLanguageCode
  const currentLanguageLabel =
    INTERFACE_LANGUAGE_OPTIONS.find(
      (language) => language.code === currentLocale
    )?.label ?? 'English'
  const otherLanguages = INTERFACE_LANGUAGE_OPTIONS.filter(
    (language) => language.code !== currentLocale
  )
  const schema = useMemo(
    () => createHeaderNavSchema(t, currentLocale),
    [t, currentLocale]
  )
  const formDefaults = useMemo(
    () => toFormValues(props.config, props.customLinks),
    [props.config, props.customLinks]
  )

  const form = useForm<HeaderNavFormValues>({
    resolver: zodResolver(schema),
    defaultValues: formDefaults,
  })
  const customLinksFieldArray = useFieldArray({
    control: form.control,
    name: 'customLinks',
  })

  useEffect(() => {
    form.reset(formDefaults)
  }, [formDefaults, form])

  const onSubmit = async (values: HeaderNavFormValues) => {
    const payload: HeaderNavModulesConfig = {
      ...props.config,
      home: values.home,
      console: values.console,
      docs: values.docs,
      about: values.about,
      pricing: {
        ...(props.config.pricing ?? HEADER_NAV_DEFAULT.pricing),
        enabled: values.pricingEnabled,
        requireAuth: values.pricingRequireAuth,
      },
      rankings: {
        ...(props.config.rankings ?? HEADER_NAV_DEFAULT.rankings),
        enabled: values.rankingsEnabled,
        requireAuth: values.rankingsRequireAuth,
      },
    }

    const serialized = serializeHeaderNavModules(payload)
    if (serialized !== props.initialSerialized) {
      await updateOption.mutateAsync({
        key: 'HeaderNavModules',
        value: serialized,
      })
    }

    const customLinksSerialized = serializeHeaderNavCustomLinks(
      values.customLinks
    )
    if (customLinksSerialized !== props.initialCustomLinksSerialized) {
      await updateOption.mutateAsync({
        key: 'HeaderNavCustomLinks',
        value: customLinksSerialized,
      })
    }
  }

  const resetToDefault = () => {
    form.reset(
      toFormValues(HEADER_NAV_DEFAULT, DEFAULT_HEADER_NAV_CUSTOM_LINKS)
    )
  }

  const simpleModules: Array<{
    key: 'home' | 'console' | 'docs' | 'about'
    title: string
    description: string
  }> = [
    {
      key: 'home',
      title: t('Home'),
      description: t('Landing page with system overview.'),
    },
    {
      key: 'console',
      title: t('Console'),
      description: t('User dashboard and quota controls.'),
    },
    {
      key: 'docs',
      title: t('Docs'),
      description: t('Documentation or external knowledge base.'),
    },
    {
      key: 'about',
      title: t('About'),
      description: t('Static page describing the platform.'),
    },
  ]

  const accessModules: Array<{
    enabledKey: 'pricingEnabled' | 'rankingsEnabled'
    requireAuthKey: 'pricingRequireAuth' | 'rankingsRequireAuth'
    requireAuthDependsOn: 'pricingEnabled' | 'rankingsEnabled'
    title: string
    description: string
    requireAuthTitle: string
    requireAuthDescription: string
  }> = [
    {
      enabledKey: 'pricingEnabled',
      requireAuthKey: 'pricingRequireAuth',
      requireAuthDependsOn: 'pricingEnabled',
      title: t('Model Square'),
      description: t('Public model catalog and pricing page.'),
      requireAuthTitle: t('Require login to view models'),
      requireAuthDescription: t(
        'Visitors must authenticate before accessing the pricing directory.'
      ),
    },
    {
      enabledKey: 'rankingsEnabled',
      requireAuthKey: 'rankingsRequireAuth',
      requireAuthDependsOn: 'rankingsEnabled',
      title: t('Rankings'),
      description: t('Public rankings page based on live usage data.'),
      requireAuthTitle: t('Require login to view rankings'),
      requireAuthDescription: t(
        'Visitors must authenticate before accessing the rankings page.'
      ),
    },
  ]

  return (
    <SettingsSection title={t('Header navigation')}>
      <Form {...form}>
        <SettingsForm onSubmit={form.handleSubmit(onSubmit)}>
          <SettingsPageFormActions
            onSave={form.handleSubmit(onSubmit)}
            onReset={resetToDefault}
            isSaving={updateOption.isPending}
            resetLabel='Reset to default'
            saveLabel='Save navigation'
          />
          <div className='grid gap-4 md:grid-cols-2'>
            {simpleModules.map((module) => (
              <FormField
                key={module.key}
                control={form.control}
                name={module.key}
                render={({ field }) => (
                  <SettingsSwitchItem>
                    <SettingsSwitchContent>
                      <FormLabel>{module.title}</FormLabel>
                      <FormDescription>{module.description}</FormDescription>
                    </SettingsSwitchContent>
                    <FormControl>
                      <Switch
                        checked={field.value}
                        onCheckedChange={field.onChange}
                      />
                    </FormControl>
                    <FormMessage />
                  </SettingsSwitchItem>
                )}
              />
            ))}
          </div>

          <SettingsControlGroup>
            <div className='flex flex-col gap-3 sm:flex-row sm:items-start sm:justify-between'>
              <div className='min-w-0'>
                <p className='text-sm font-medium'>{t('Custom links')}</p>
                <p className='text-muted-foreground text-xs'>
                  {t(
                    'Custom links open in a new browser window and appear after the model square.'
                  )}
                </p>
              </div>
              <Button
                type='button'
                variant='outline'
                size='sm'
                onClick={() =>
                  customLinksFieldArray.append({
                    titles: createEmptyHeaderNavCustomLinkTitles(),
                    url: '',
                  })
                }
              >
                <HugeiconsIcon
                  icon={Add01Icon}
                  data-icon='inline-start'
                  aria-hidden='true'
                />
                {t('Add link')}
              </Button>
            </div>

            {customLinksFieldArray.fields.length === 0 ? (
              <Empty className='border py-5'>
                <EmptyHeader>
                  <EmptyMedia variant='icon'>
                    <HugeiconsIcon icon={Link01Icon} aria-hidden='true' />
                  </EmptyMedia>
                  <EmptyTitle>{t('No custom links configured')}</EmptyTitle>
                  <EmptyDescription>
                    {t('Use Add link to create a new top navigation entry.')}
                  </EmptyDescription>
                </EmptyHeader>
              </Empty>
            ) : (
              <div className='flex flex-col gap-3'>
                {customLinksFieldArray.fields.map((customLink, index) => (
                  <div
                    key={customLink.id}
                    className='bg-background grid min-w-0 gap-3 rounded-lg border p-3 md:grid-cols-[minmax(0,1fr)_minmax(0,1.5fr)_auto] md:items-start'
                  >
                    <FormField
                      control={form.control}
                      name={`customLinks.${index}.titles.${currentLocale}`}
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>
                            {t('Name ({{language}})', {
                              language: currentLanguageLabel,
                            })}
                          </FormLabel>
                          <FormControl>
                            <Input
                              placeholder={t('Image Generation')}
                              {...field}
                            />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name={`customLinks.${index}.url`}
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>{t('URL')}</FormLabel>
                          <FormControl>
                            <Input
                              type='url'
                              placeholder='https://example.com/'
                              {...field}
                            />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                    <div className='flex md:pt-6'>
                      <Button
                        type='button'
                        variant='destructive'
                        size='icon'
                        aria-label={`${t('Delete')} ${index + 1}`}
                        onClick={() => customLinksFieldArray.remove(index)}
                      >
                        <HugeiconsIcon icon={Delete02Icon} aria-hidden='true' />
                      </Button>
                    </div>
                    <div className='md:col-span-3'>
                      <Accordion>
                        <AccordionItem value='other-language-names'>
                          <AccordionTrigger className='py-1.5 hover:no-underline'>
                            {t('Other language names')}
                          </AccordionTrigger>
                          <AccordionContent className='pt-3'>
                            <div className='flex flex-col gap-3'>
                              <p className='text-muted-foreground text-xs'>
                                {t(
                                  'Leave a language blank to use another configured name as fallback.'
                                )}
                              </p>
                              <div className='grid gap-3 sm:grid-cols-2 lg:grid-cols-3'>
                                {otherLanguages.map((language) => (
                                  <FormField
                                    key={language.code}
                                    control={form.control}
                                    name={`customLinks.${index}.titles.${language.code}`}
                                    render={({ field }) => (
                                      <FormItem>
                                        <FormLabel>{language.label}</FormLabel>
                                        <FormControl>
                                          <Input {...field} />
                                        </FormControl>
                                        <FormMessage />
                                      </FormItem>
                                    )}
                                  />
                                ))}
                              </div>
                            </div>
                          </AccordionContent>
                        </AccordionItem>
                      </Accordion>
                    </div>
                  </div>
                ))}
              </div>
            )}
          </SettingsControlGroup>

          <div className='grid gap-4 lg:grid-cols-2'>
            {accessModules.map((module) => (
              <SettingsControlGroup key={module.enabledKey}>
                <FormField
                  control={form.control}
                  name={module.enabledKey}
                  render={({ field }) => (
                    <SettingsSwitchItem>
                      <SettingsSwitchContent>
                        <FormLabel>{module.title}</FormLabel>
                        <FormDescription>{module.description}</FormDescription>
                      </SettingsSwitchContent>
                      <FormControl>
                        <Switch
                          checked={field.value}
                          onCheckedChange={field.onChange}
                        />
                      </FormControl>
                      <FormMessage />
                    </SettingsSwitchItem>
                  )}
                />

                <FormField
                  control={form.control}
                  name={module.requireAuthKey}
                  render={({ field }) => (
                    <SettingsControlChildren>
                      <SettingsSwitchItem className='py-2'>
                        <SettingsSwitchContent>
                          <FormLabel>{module.requireAuthTitle}</FormLabel>
                          <FormDescription>
                            {module.requireAuthDescription}
                          </FormDescription>
                        </SettingsSwitchContent>
                        <FormControl>
                          <Switch
                            checked={field.value}
                            onCheckedChange={field.onChange}
                            disabled={!form.watch(module.requireAuthDependsOn)}
                          />
                        </FormControl>
                        <FormMessage />
                      </SettingsSwitchItem>
                    </SettingsControlChildren>
                  )}
                />
              </SettingsControlGroup>
            ))}
          </div>
        </SettingsForm>
      </Form>
    </SettingsSection>
  )
}
