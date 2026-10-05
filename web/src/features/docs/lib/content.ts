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
import anthropicMessages from '../content/anthropic-messages.md?raw'
import apiKeys from '../content/api-keys.md?raw'
import apiOverview from '../content/api-overview.md?raw'
import billing from '../content/billing.md?raw'
import ccSwitch from '../content/cc-switch.md?raw'
import chatClients from '../content/chat-clients.md?raw'
import chatCompletions from '../content/chat-completions.md?raw'
import claudeCode from '../content/claude-code.md?raw'
import codex from '../content/codex.md?raw'
import errors from '../content/errors.md?raw'
import faq from '../content/faq.md?raw'
import firstRequest from '../content/first-request.md?raw'
import geminiCli from '../content/gemini-cli.md?raw'
import geminiNative from '../content/gemini-native.md?raw'
import groups from '../content/groups.md?raw'
import images from '../content/images.md?raw'
import introduction from '../content/introduction.md?raw'
import models from '../content/models.md?raw'
import register from '../content/register.md?raw'
import responses from '../content/responses.md?raw'
import videos from '../content/videos.md?raw'
import workbenchFeatures from '../content/workbench-features.md?raw'
import workbench from '../content/workbench.md?raw'

/** Markdown source per doc slug; every slug in `DOC_SECTIONS` has an entry. */
export const DOC_CONTENT: Record<string, string> = {
  introduction,
  register,
  billing,
  'api-keys': apiKeys,
  'first-request': firstRequest,
  groups,
  models,
  'cc-switch': ccSwitch,
  'claude-code': claudeCode,
  codex,
  'gemini-cli': geminiCli,
  'chat-clients': chatClients,
  'api-overview': apiOverview,
  'chat-completions': chatCompletions,
  responses,
  'anthropic-messages': anthropicMessages,
  'gemini-native': geminiNative,
  images,
  videos,
  errors,
  workbench,
  'workbench-features': workbenchFeatures,
  faq,
}
