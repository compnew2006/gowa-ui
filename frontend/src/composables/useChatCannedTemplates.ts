import { ref, computed } from 'vue'
import { toast } from 'vue-sonner'
import { cannedResponsesService, type CannedResponse } from '@/services/api'
import { getErrorMessage } from '@/lib/api-utils'
import { useHeaderMedia } from '@/composables/useHeaderMedia'

export interface UseChatCannedTemplatesOptions {
  /** i18n translator. */
  t: (key: string, params?: Record<string, unknown>) => string
  /** Contacts store reactive surface. */
  contactsStore: {
    currentContact: { id: string } | null
    replyingTo: { id: string } | null
    clearReplyingTo: () => void
    // Store sendMessage/sendTemplate are variadic/loosely typed; accept the
    // shapes we call them with.
    sendMessage: (...args: any[]) => Promise<void>
    sendTemplate: (...args: any[]) => Promise<void>
  }
  /** Selected account ref (multi-account). */
  selectedAccount: { value: string | null }
  /** Current contact getter — for context-token resolution. */
  getCurrentContact: () => { profile_name?: string; name?: string; phone_number?: string } | null
  /** Current signed-in user getter — for {{agent_name}}/{{user_name}} tokens. */
  getCurrentUser: () => { full_name?: string } | null
  /** Composer input ref — canned picker resets it; templates do not touch it. */
  messageInput: { value: string }
  /** Reset the composer textarea height after clearing the input. */
  resetTextareaHeight: () => void
  /** Called after a canned response lands so the room scrolls to the new bubble. */
  scrollToBottom: (instant?: boolean) => void
}

/**
 * Canned-response slash-command picker + WhatsApp template message composer
 * for the chat view. Owns the canned picker, canned param dialog, and the
 * template param dialog state. Shares the AUTO_RESOLVED_CONTEXT_TOKENS set
 * between canned and templates so context tokens (contact_name, agent_name, …)
 * resolve the same way on both paths.
 *
 * @example
 * ```ts
 * const ct = useChatCannedTemplates({ contactsStore, selectedAccount, ... })
 * ```
 */
export function useChatCannedTemplates(options: UseChatCannedTemplatesOptions) {
  const { t, contactsStore, selectedAccount, messageInput } = options

  // Tokens that the chat already knows how to fill from the current contact /
  // signed-in agent. Shared by canned responses (resolved client-side into the
  // outgoing message) and templates (pre-filled into the param payload so the
  // backend forwards the resolved value to Meta).
  const AUTO_RESOLVED_CONTEXT_TOKENS = new Set(['contact_name', 'phone_number', 'user_name', 'agent_name'])

  // ─── Canned responses slash command state ───
  const cannedPickerOpen = ref(false)
  const cannedSearchQuery = ref('')

  // Canned response preview dialog state
  const cannedDialogOpen = ref(false)
  const selectedCannedResponse = ref<CannedResponse | null>(null)
  const cannedParamNames = ref<string[]>([])
  const cannedParamValues = ref<Record<string, string>>({})
  const isSendingCanned = ref(false)

  // ─── Template picker state ───
  const templateDialogOpen = ref(false)
  const selectedTemplate = ref<any>(null)
  const templateParamNames = ref<string[]>([])
  const templateParamValues = ref<Record<string, string>>({})
  // Name of the TEXT-header variable (max 1 per Meta) and its value. Kept in
  // its own ref so a positional {{1}} in the header doesn't collide with a
  // {{1}} body parameter — both can be filled independently.
  const templateHeaderParamName = ref<string | null>(null)
  const templateHeaderParamValue = ref('')
  const templateButtonUrlParams = ref<{ index: number; text: string; value: string; type: string }[]>([])
  const isSendingTemplate = ref(false)
  const templateHeaderType = computed(() => selectedTemplate.value?.header_type)
  const {
    file: templateHeaderFile,
    previewUrl: templateHeaderPreview,
    needsMedia: templateNeedsHeaderMedia,
    acceptTypes: templateHeaderAccept,
    handleFileChange: handleTemplateHeaderFile,
    clear: clearTemplateHeaderMedia,
  } = useHeaderMedia(templateHeaderType)

  // ─── Token resolution (shared by canned + template) ───
  function extractCannedTokens(content: string): string[] {
    const seen = new Set<string>()
    const matches = content.matchAll(/\{\{\s*([\w.-]+)\s*\}\}/g)
    for (const m of matches) seen.add(m[1])
    return Array.from(seen)
  }

  // Collect tokens from the message body AND every button field, so the param
  // dialog prompts for custom tokens used anywhere on the response.
  function extractCannedTokensFromResponse(r: CannedResponse): string[] {
    const seen = new Set<string>(extractCannedTokens(r.content))
    for (const btn of r.buttons || []) {
      for (const tk of extractCannedTokens(btn.title || '')) seen.add(tk)
      for (const tk of extractCannedTokens(btn.url || '')) seen.add(tk)
      for (const tk of extractCannedTokens(btn.phone_number || '')) seen.add(tk)
    }
    return Array.from(seen)
  }

  // Resolve a single context token (contact_name / phone_number / user_name /
  // agent_name) against the current chat. Returns null for any key that isn't
  // in AUTO_RESOLVED_CONTEXT_TOKENS so callers can fall back to their own param
  // dict.
  function resolveContextToken(key: string): string | null {
    const contact = options.getCurrentContact()
    if (key === 'contact_name') return contact?.profile_name || contact?.name || 'there'
    if (key === 'phone_number') return contact?.phone_number || ''
    if (key === 'user_name' || key === 'agent_name') return options.getCurrentUser()?.full_name || ''
    return null
  }

  // Shared {{...}} resolver used by the body preview and the button fields, so
  // `{{phone_number}}` works inside a button URL the same way it does in content.
  function resolveCannedTokens(text: string): string {
    if (!text) return text
    return text.replace(/\{\{\s*([\w.-]+)\s*\}\}/g, (_match, key: string) => {
      const ctx = resolveContextToken(key)
      if (ctx !== null) return ctx
      const value = cannedParamValues.value[key]
      return value ? value : `{{${key}}}`
    })
  }

  const cannedPreview = computed(() =>
    selectedCannedResponse.value ? resolveCannedTokens(selectedCannedResponse.value.content) : '',
  )

  // Resolved buttons (with {{...}} substitution applied) for the dialog preview.
  // Empty array when no response is selected or it has no buttons.
  const cannedPreviewButtons = computed(() => {
    const raw = selectedCannedResponse.value?.buttons || []
    return raw.map(b => ({
      ...b,
      title: resolveCannedTokens(b.title),
      ...(b.url !== undefined ? { url: resolveCannedTokens(b.url) } : {}),
      ...(b.phone_number !== undefined ? { phone_number: resolveCannedTokens(b.phone_number) } : {}),
    }))
  })

  function handleCannedSelect(response: CannedResponse) {
    selectedCannedResponse.value = response
    const tokens = extractCannedTokensFromResponse(response).filter(
      tk => !AUTO_RESOLVED_CONTEXT_TOKENS.has(tk)
    )
    cannedParamNames.value = tokens
    cannedParamValues.value = Object.fromEntries(tokens.map(tk => [tk, '']))
    // Drop the slash command (or any stray text) so the textarea starts clean.
    messageInput.value = ''
    options.resetTextareaHeight()
    cannedPickerOpen.value = false
    cannedSearchQuery.value = ''
    cannedDialogOpen.value = true
  }

  async function sendCannedResponse() {
    if (!contactsStore.currentContact || !selectedCannedResponse.value) return

    const missing = cannedParamNames.value.some(n => !cannedParamValues.value[n]?.trim())
    if (missing) {
      toast.error(t('chat.parameterRequired'))
      return
    }

    const body = cannedPreview.value
    const responseId = selectedCannedResponse.value.id
    // Substitute {{...}} tokens in every button field — same rules as the body —
    // so URLs like https://x.com/u/{{phone_number}} resolve at send time.
    const buttons = (selectedCannedResponse.value.buttons || []).map(b => ({
      ...b,
      title: resolveCannedTokens(b.title),
      ...(b.url !== undefined ? { url: resolveCannedTokens(b.url) } : {}),
      ...(b.phone_number !== undefined ? { phone_number: resolveCannedTokens(b.phone_number) } : {}),
    }))
    const replyButtons = buttons.filter(b => !b.type || b.type === 'reply')
    const urlButtons = buttons.filter(b => b.type === 'url')

    let sendType: 'text' | 'interactive' = 'text'
    let interactive: {
      type: 'button' | 'list' | 'cta_url'
      body: string
      buttons?: Array<{ id: string; title: string }>
      button_text?: string
      url?: string
    } | undefined

    if (buttons.length > 0 && replyButtons.length === buttons.length && replyButtons.length <= 10) {
      sendType = 'interactive'
      interactive = {
        type: replyButtons.length <= 3 ? 'button' : 'list',
        body,
        buttons: replyButtons.map(b => ({ id: b.id, title: b.title })),
      }
    } else if (buttons.length === 1 && urlButtons.length === 1) {
      sendType = 'interactive'
      interactive = {
        type: 'cta_url',
        body,
        button_text: urlButtons[0].title,
        url: urlButtons[0].url || '',
      }
    }

    isSendingCanned.value = true
    try {
      await contactsStore.sendMessage(
        contactsStore.currentContact.id,
        sendType,
        sendType === 'interactive' ? { body } : { body },
        contactsStore.replyingTo?.id,
        selectedAccount.value || undefined,
        interactive ? { interactive } : undefined,
      )
      cannedResponsesService.use(responseId).catch(() => {})
      contactsStore.clearReplyingTo()
      cannedDialogOpen.value = false
      selectedCannedResponse.value = null
      cannedParamNames.value = []
      cannedParamValues.value = {}
      await new Promise(resolve => setTimeout(resolve, 0))
      options.scrollToBottom()
    } catch (error) {
      toast.error(getErrorMessage(error, t('chat.sendMessageFailed')))
    } finally {
      isSendingCanned.value = false
    }
  }

  function closeCannedPicker() {
    cannedPickerOpen.value = false
    cannedSearchQuery.value = ''
  }

  // ─── Template handling ───
  function getTemplateBodyContent(tpl: any): string {
    return tpl.body_content || ''
  }

  const templatePreview = computed(() => {
    if (!selectedTemplate.value) return ''
    const body = getTemplateBodyContent(selectedTemplate.value)
    return body.replace(/\{\{\s*([\w.-]+)\s*\}\}/g, (_match, key: string) => {
      const supplied = templateParamValues.value[key]
      if (supplied) return supplied
      const ctx = resolveContextToken(key)
      if (ctx !== null) return ctx
      return `{{${key}}}`
    })
  })

  // Show the header input only when the user has to fill it. Context-token
  // names (contact_name, phone_number, …) auto-resolve and stay hidden — same
  // rule body params follow via templateParamNames filtering.
  const showHeaderParamInput = computed(() =>
    !!templateHeaderParamName.value &&
    !AUTO_RESOLVED_CONTEXT_TOKENS.has(templateHeaderParamName.value)
  )

  function extractButtonUrlParams(buttons: any[]): { index: number; text: string; value: string; type: string }[] {
    if (!buttons?.length) return []
    return buttons
      .map((btn: any, index: number) => {
        if (btn.type === 'COPY_CODE') {
          return { index, text: btn.text || 'Copy Code', value: btn.example?.[0] || '', type: 'COPY_CODE' }
        }
        if (btn.type !== 'URL' || !btn.url) return null
        const hasParams = /\{\{[^}]+\}\}/.test(btn.url)
        if (!hasParams) return null
        return { index, text: btn.text || 'URL Button', value: '', type: 'URL' }
      })
      .filter((b): b is { index: number; text: string; value: string; type: string } => b !== null)
  }

  function handleTemplateWithParams(template: any, paramNames: string[]) {
    selectedTemplate.value = template
    // Pre-fill body context tokens from the conversation; keep them in the
    // payload dict so the backend forwards them to Meta, but hide them from
    // the dialog so the agent doesn't have to type values we already know —
    // same pattern as canned responses (see handleCannedSelect).
    const initial: Record<string, string> = {}
    for (const name of paramNames) {
      const resolved = resolveContextToken(name)
      initial[name] = resolved ?? ''
    }
    templateParamValues.value = initial
    templateParamNames.value = paramNames.filter(n => !AUTO_RESOLVED_CONTEXT_TOKENS.has(n))

    // Identify the TEXT-header variable (max 1) and pre-fill from context.
    // Context-token names (contact_name / phone_number / agent_name / user_name)
    // resolve automatically and stay hidden from the dialog — same convention
    // as body params.
    templateHeaderParamName.value = null
    templateHeaderParamValue.value = ''
    if (template.header_type === 'TEXT' && template.header_content) {
      const m = template.header_content.match(/\{\{([^}]+)\}\}/)
      if (m) {
        const name = m[1].trim()
        templateHeaderParamName.value = name
        templateHeaderParamValue.value = resolveContextToken(name) ?? ''
      }
    }

    clearTemplateHeaderMedia()
    templateButtonUrlParams.value = extractButtonUrlParams(template.buttons)
    templateDialogOpen.value = true
  }

  async function sendTemplateMessage() {
    if (!contactsStore.currentContact || !selectedTemplate.value) return

    // Validate header param (separate ref so it can hold its own value even
    // when the body has a {{1}} that would otherwise collide). Auto-resolved
    // context tokens are exempt — their value comes from the conversation.
    if (showHeaderParamInput.value && !templateHeaderParamValue.value.trim()) {
      toast.error(t('chat.parameterRequired'))
      return
    }

    // Validate all body params are filled
    const missingBody = templateParamNames.value.some(n => !templateParamValues.value[n]?.trim())
    if (missingBody) {
      toast.error(t('chat.parameterRequired'))
      return
    }

    // Validate header media if required
    if (templateNeedsHeaderMedia.value && !templateHeaderFile.value) {
      toast.error(t('chat.headerMediaRequired'))
      return
    }

    // Validate all button URL params are filled
    const missingButton = templateButtonUrlParams.value.some(b => !b.value?.trim())
    if (missingButton) {
      toast.error(t('chat.parameterRequired'))
      return
    }

    // Build button params map: button index -> value
    const buttonParams: Record<string, string> | undefined =
      templateButtonUrlParams.value.length > 0
        ? Object.fromEntries(templateButtonUrlParams.value.map(b => [String(b.index), b.value]))
        : undefined

    // Header value goes in its own payload field so a positional {{1}} header
    // doesn't overwrite a positional {{1}} body parameter in the flat map.
    const headerParams: Record<string, string> | undefined =
      templateHeaderParamName.value && templateHeaderParamValue.value
        ? { [templateHeaderParamName.value]: templateHeaderParamValue.value }
        : undefined

    isSendingTemplate.value = true
    try {
      await contactsStore.sendTemplate(
        contactsStore.currentContact.id,
        selectedTemplate.value.name,
        templateParamValues.value,
        selectedAccount.value || undefined,
        templateHeaderFile.value || undefined,
        buttonParams,
        headerParams
      )
      toast.success(t('chat.templateSent'))
      templateDialogOpen.value = false
      selectedTemplate.value = null
      templateParamNames.value = []
      templateParamValues.value = {}
      templateHeaderParamName.value = null
      templateHeaderParamValue.value = ''
      clearTemplateHeaderMedia()
      templateButtonUrlParams.value = []
    } catch (error: any) {
      const message = error.response?.data?.message || t('chat.templateSendFailed')
      toast.error(message)
    } finally {
      isSendingTemplate.value = false
    }
  }

  return {
    // Canned picker
    cannedPickerOpen,
    cannedSearchQuery,
    cannedDialogOpen,
    selectedCannedResponse,
    cannedParamNames,
    cannedParamValues,
    isSendingCanned,
    cannedPreview,
    cannedPreviewButtons,
    handleCannedSelect,
    sendCannedResponse,
    closeCannedPicker,
    // Template composer
    templateDialogOpen,
    selectedTemplate,
    templateParamNames,
    templateParamValues,
    templateHeaderParamName,
    templateHeaderParamValue,
    templateButtonUrlParams,
    isSendingTemplate,
    templatePreview,
    templateHeaderType,
    templateHeaderFile,
    templateHeaderPreview,
    templateNeedsHeaderMedia,
    templateHeaderAccept,
    showHeaderParamInput,
    handleTemplateWithParams,
    handleTemplateHeaderFile,
    clearTemplateHeaderMedia,
    sendTemplateMessage,
  }
}
