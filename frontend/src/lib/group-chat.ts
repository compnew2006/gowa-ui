import type { Contact, Message } from '@/stores/contacts'

const GROUP_JID_SUFFIX = '@g.us'

export function isGroupContact(contact: Pick<Contact, 'phone_number' | 'metadata' | 'is_group_chat'> | null | undefined): boolean {
  if (!contact) {
    return false
  }

  if (typeof contact.is_group_chat === 'boolean') {
    return contact.is_group_chat
  }

  if (contact.metadata && typeof contact.metadata.is_group_chat === 'boolean') {
    return contact.metadata.is_group_chat
  }

  return typeof contact.phone_number === 'string' && contact.phone_number.endsWith(GROUP_JID_SUFFIX)
}

export function getMessageSenderPhone(message: Pick<Message, 'sender_phone' | 'metadata'>): string {
  if (typeof message.sender_phone === 'string' && message.sender_phone.trim() !== '') {
    return message.sender_phone.trim()
  }

  const metadataSender = message.metadata?.sender_phone
  if (typeof metadataSender === 'string' && metadataSender.trim() !== '') {
    return metadataSender.trim()
  }

  return ''
}
