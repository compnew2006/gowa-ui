import type { Contact } from '@/stores/contacts'

export type ChatSidebarViewMode = 'separate' | 'unified'

export interface SidebarContactEntry {
  key: string
  displayContact: Contact
  sourceContacts: Contact[]
  sourceContactIDs: string[]
  accountNames: string[]
  contactsByAccount: Record<string, Contact>
  isUnified: boolean
}

interface MutableSidebarContactEntry {
  key: string
  displayContact: Contact
  sourceContacts: Contact[]
  contactsByAccount: Record<string, Contact>
  isUnified: boolean
}

const GROUP_CHAT_SUFFIX = '@g.us'
const CHANNEL_CHAT_SUFFIX = '@newsletter'

function getTimestamp(isoDate?: string): number {
  if (!isoDate) return 0
  const parsed = new Date(isoDate).getTime()
  return Number.isFinite(parsed) ? parsed : 0
}

function pickLatestContact(a: Contact, b: Contact): Contact {
  return getTimestamp(b.last_message_at) >= getTimestamp(a.last_message_at) ? b : a
}

function mergeDisplayContact(existing: Contact, incoming: Contact): Contact {
  const latest = pickLatestContact(existing, incoming)
  return {
    ...existing,
    ...latest,
    unread_count: (existing.unread_count || 0) + (incoming.unread_count || 0),
    tags: Array.from(new Set([...(existing.tags || []), ...(incoming.tags || [])]))
  }
}

function readConversationID(contact: Contact): string {
  const explicitConversationID = typeof contact.conversation_id === 'string'
    ? contact.conversation_id.trim()
    : ''
  if (explicitConversationID) {
    return explicitConversationID
  }

  const groupJID = typeof contact.metadata?.group_jid === 'string'
    ? contact.metadata.group_jid.trim()
    : ''
  if (groupJID) {
    return groupJID
  }

  const channelJID = typeof contact.metadata?.channel_jid === 'string'
    ? contact.metadata.channel_jid.trim()
    : ''
  if (channelJID) {
    return channelJID
  }

  const phone = typeof contact.phone_number === 'string'
    ? contact.phone_number.trim()
    : ''
  if (phone.endsWith(GROUP_CHAT_SUFFIX) || phone.endsWith(CHANNEL_CHAT_SUFFIX)) {
    return phone
  }

  return ''
}

function isGroupOrChannelContact(contact: Contact): boolean {
  if (contact.is_group_chat === true || contact.metadata?.is_group_chat === true) {
    return true
  }

  if (contact.metadata?.is_channel_chat === true) {
    return true
  }

  const conversationID = readConversationID(contact)
  return conversationID.endsWith(GROUP_CHAT_SUFFIX) || conversationID.endsWith(CHANNEL_CHAT_SUFFIX)
}

function normalizePhoneNumber(phoneNumber: string): string {
  return phoneNumber.trim().toLowerCase()
}

export class ChatSidebarUnifier {
  static readonly VIEW_MODE_STORAGE_KEY = 'chat.sidebarViewMode'
  static readonly DEFAULT_VIEW_MODE: ChatSidebarViewMode = 'separate'

  static normalizeViewMode(value: unknown): ChatSidebarViewMode {
    return value === 'unified' ? 'unified' : 'separate'
  }

  static readViewMode(): ChatSidebarViewMode {
    try {
      return ChatSidebarUnifier.normalizeViewMode(localStorage.getItem(ChatSidebarUnifier.VIEW_MODE_STORAGE_KEY))
    } catch {
      return ChatSidebarUnifier.DEFAULT_VIEW_MODE
    }
  }

  static saveViewMode(mode: ChatSidebarViewMode) {
    try {
      localStorage.setItem(ChatSidebarUnifier.VIEW_MODE_STORAGE_KEY, mode)
    } catch {
      // Ignore localStorage errors
    }
  }

  buildEntries(contacts: Contact[], mode: ChatSidebarViewMode): SidebarContactEntry[] {
    const groupedEntries = new Map<string, MutableSidebarContactEntry>()

    for (const contact of contacts) {
      const key = this.getEntryKey(contact, mode)
      const existing = groupedEntries.get(key)

      if (!existing) {
        groupedEntries.set(key, {
          key,
          displayContact: { ...contact },
          sourceContacts: [contact],
          contactsByAccount: this.createAccountContactMap(contact),
          isUnified: mode === 'unified' && this.canUnifyByPhone(contact)
        })
        continue
      }

      existing.displayContact = mergeDisplayContact(existing.displayContact, contact)
      existing.sourceContacts.push(contact)

      const accountName = this.getAccountName(contact)
      if (accountName) {
        const existingAccountContact = existing.contactsByAccount[accountName]
        existing.contactsByAccount[accountName] = existingAccountContact
          ? pickLatestContact(existingAccountContact, contact)
          : contact
      }
    }

    return Array.from(groupedEntries.values())
      .map((entry) => {
        const accountNames = Object.keys(entry.contactsByAccount).sort((a, b) => a.localeCompare(b))
        const sourceContactIDs = Array.from(new Set(entry.sourceContacts.map((contact) => contact.id)))

        return {
          key: entry.key,
          displayContact: entry.displayContact,
          sourceContacts: entry.sourceContacts,
          sourceContactIDs,
          accountNames,
          contactsByAccount: entry.contactsByAccount,
          isUnified: entry.isUnified
        }
      })
      .sort((a, b) => getTimestamp(b.displayContact.last_message_at) - getTimestamp(a.displayContact.last_message_at))
  }

  findEntryByContactID(entries: SidebarContactEntry[], contactID: string): SidebarContactEntry | undefined {
    return entries.find((entry) => entry.sourceContactIDs.includes(contactID))
  }

  private createAccountContactMap(contact: Contact): Record<string, Contact> {
    const accountName = this.getAccountName(contact)
    if (!accountName) {
      return {}
    }

    return {
      [accountName]: contact
    }
  }

  private getAccountName(contact: Contact): string {
    return typeof contact.whatsapp_account === 'string' ? contact.whatsapp_account.trim() : ''
  }

  private canUnifyByPhone(contact: Contact): boolean {
    const normalizedPhone = normalizePhoneNumber(contact.phone_number || '')
    if (!normalizedPhone) {
      return false
    }

    if (isGroupOrChannelContact(contact)) {
      return false
    }

    return true
  }

  private getEntryKey(contact: Contact, mode: ChatSidebarViewMode): string {
    if (mode === 'unified' && this.canUnifyByPhone(contact)) {
      return `phone:${normalizePhoneNumber(contact.phone_number)}`
    }

    if (isGroupOrChannelContact(contact)) {
      const conversationID = readConversationID(contact)
      const instanceID = contact.instance_id || 'no-instance'
      return `conversation:${conversationID || contact.id}:${instanceID}`
    }

    return `contact:${contact.id}`
  }
}
