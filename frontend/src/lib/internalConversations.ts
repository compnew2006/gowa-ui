import type { Contact } from '@/stores/contacts'

// Internal conversations are chats between two of the org's own WhatsApp
// numbers. Every message is stored twice — once per account — on two different
// contacts, so the backend (GET /contacts/internal-conversations) merges the two
// copies ("sides") into one conversation and the sidebar lists it once.

/** One account's copy of the conversation: `account`'s messages with
 *  `peer_account` live on `contact_id` (the peer number's contact). */
export interface InternalConversationSide {
  contact_id: string
  account: string
  peer_account: string
  last_message_at?: string
  last_message_preview: string
  unread_count: number
}

export interface InternalConversation {
  key: string
  accounts: [string, string]
  last_message_at?: string
  last_message_preview: string
  unread_count: number
  sides: InternalConversationSide[]
}

function sideTime(side: InternalConversationSide): number {
  return side.last_message_at ? new Date(side.last_message_at).getTime() : 0
}

/** The side a click opens: the one holding unread messages (the receiving
 *  account, so a reply goes back the natural way), else the most recent. */
export function defaultSide(conv: InternalConversation): InternalConversationSide {
  return [...conv.sides].sort((a, b) =>
    (b.unread_count - a.unread_count)
    || (sideTime(b) - sideTime(a))
    || a.account.localeCompare(b.account)
  )[0]
}

/** The conversation `contactId` is a side of. A contact can be a side of
 *  several conversations (one per other org account), so `account` — the
 *  account the chat is viewed through — disambiguates when known. */
export function conversationForContact(
  convs: InternalConversation[],
  contactId: string,
  account?: string | null,
): InternalConversation | undefined {
  const matches = convs.filter(conv => conv.sides.some(s => s.contact_id === contactId))
  if (account) {
    const exact = matches.find(conv =>
      conv.sides.some(s => s.contact_id === contactId && s.account === account))
    if (exact) return exact
  }
  return matches[0]
}

/** A conversation as one sidebar row, shaped as a Contact so the list template
 *  renders it unchanged. The row opens `defaultSide`; the side contacts are kept
 *  so the row stays highlighted whichever side is open. */
export function internalConversationRow(
  conv: InternalConversation,
  contactsById: Map<string, Contact>,
): Contact {
  const side = defaultSide(conv)
  const base: Contact = contactsById.get(side.contact_id) ?? {
    id: side.contact_id,
    phone_number: '',
    name: '',
    status: 'active',
    tags: [],
    metadata: {},
    unread_count: 0,
    created_at: '',
    updated_at: '',
  }
  return {
    ...base,
    name: `${conv.accounts[0]} ↔ ${conv.accounts[1]}`,
    is_internal: true,
    last_message_at: conv.last_message_at,
    last_message_preview: conv.last_message_preview,
    unread_count: conv.unread_count,
    // Both accounts are already in the row title.
    last_message_account: undefined,
    whatsapp_account: undefined,
    internal_conversation_key: conv.key,
    internal_side_contact_ids: conv.sides.map(s => s.contact_id),
    internal_open_account: side.account,
  }
}
