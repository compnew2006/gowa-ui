import { describe, expect, it } from 'vitest'

import type { Contact } from '@/stores/contacts'
import { ChatSidebarUnifier } from './chat-sidebar-unifier'

function createContact(overrides: Partial<Contact> & { id: string; phone_number: string; whatsapp_account?: string }): Contact {
  return {
    id: overrides.id,
    phone_number: overrides.phone_number,
    name: overrides.name || overrides.phone_number,
    status: overrides.status || 'open',
    tags: overrides.tags || [],
    metadata: overrides.metadata || {},
    unread_count: overrides.unread_count || 0,
    whatsapp_account: overrides.whatsapp_account,
    instance_id: overrides.instance_id,
    conversation_id: overrides.conversation_id,
    is_group_chat: overrides.is_group_chat,
    profile_name: overrides.profile_name,
    avatar_url: overrides.avatar_url,
    last_message_at: overrides.last_message_at,
    last_message_preview: overrides.last_message_preview,
    last_inbound_at: overrides.last_inbound_at,
    service_window_open: overrides.service_window_open,
    assigned_user_id: overrides.assigned_user_id,
    assigned_user_name: overrides.assigned_user_name,
    closed_at: overrides.closed_at,
    closed_by_user_id: overrides.closed_by_user_id,
    closed_by_name: overrides.closed_by_name,
    created_at: overrides.created_at || '2026-02-01T00:00:00Z',
    updated_at: overrides.updated_at || '2026-02-01T00:00:00Z'
  }
}

describe('ChatSidebarUnifier', () => {
  const unifier = new ChatSidebarUnifier()

  it('keeps private chats separate when view mode is separate', () => {
    const contactA = createContact({
      id: 'c-1',
      phone_number: '+15550001111',
      whatsapp_account: 'account-a',
      unread_count: 1,
      last_message_at: '2026-02-21T10:00:00Z'
    })
    const contactB = createContact({
      id: 'c-2',
      phone_number: '+15550001111',
      whatsapp_account: 'account-b',
      unread_count: 2,
      last_message_at: '2026-02-21T11:00:00Z'
    })

    const entries = unifier.buildEntries([contactA, contactB], 'separate')

    expect(entries).toHaveLength(2)
    expect(entries.map((entry) => entry.key)).toEqual(['contact:c-2', 'contact:c-1'])
  })

  it('merges private chats with the same phone across accounts in unified mode', () => {
    const contactA = createContact({
      id: 'c-1',
      phone_number: '+15550001111',
      whatsapp_account: 'account-a',
      unread_count: 1,
      tags: ['vip'],
      last_message_at: '2026-02-21T10:00:00Z',
      name: 'Alice'
    })
    const contactB = createContact({
      id: 'c-2',
      phone_number: '+15550001111',
      whatsapp_account: 'account-b',
      unread_count: 2,
      tags: ['priority'],
      last_message_at: '2026-02-21T11:00:00Z',
      name: 'Alice B'
    })

    const entries = unifier.buildEntries([contactA, contactB], 'unified')

    expect(entries).toHaveLength(1)
    expect(entries[0].key).toBe('phone:+15550001111')
    expect(entries[0].isUnified).toBe(true)
    expect(entries[0].sourceContactIDs).toEqual(['c-1', 'c-2'])
    expect(entries[0].accountNames).toEqual(['account-a', 'account-b'])
    expect(entries[0].displayContact.unread_count).toBe(3)
    expect(entries[0].displayContact.tags.sort()).toEqual(['priority', 'vip'])
    expect(entries[0].displayContact.name).toBe('Alice B')
  })

  it('merges same group conversation across accounts in unified mode', () => {
    const groupA = createContact({
      id: 'g-1',
      phone_number: '1203630@g.us',
      conversation_id: '1203630@g.us',
      is_group_chat: true,
      instance_id: 'instance-a',
      whatsapp_account: 'account-a',
      unread_count: 1
    })
    const groupB = createContact({
      id: 'g-2',
      phone_number: '1203630@g.us',
      conversation_id: '1203630@g.us',
      is_group_chat: true,
      instance_id: 'instance-b',
      whatsapp_account: 'account-b',
      unread_count: 2
    })

    const entries = unifier.buildEntries([groupA, groupB], 'unified')

    expect(entries).toHaveLength(1)
    expect(entries[0].key).toBe('conversation:1203630@g.us')
    expect(entries[0].key).not.toContain('phone:')
    expect(entries[0].sourceContactIDs).toEqual(['g-1', 'g-2'])
    expect(entries[0].accountNames).toEqual(['account-a', 'account-b'])
    expect(entries[0].displayContact.unread_count).toBe(3)
  })

  it('keeps same group conversation separate by instance in separate mode', () => {
    const groupA = createContact({
      id: 'g-1',
      phone_number: '1203630@g.us',
      conversation_id: '1203630@g.us',
      is_group_chat: true,
      instance_id: 'instance-a',
      whatsapp_account: 'account-a'
    })
    const groupB = createContact({
      id: 'g-2',
      phone_number: '1203630@g.us',
      conversation_id: '1203630@g.us',
      is_group_chat: true,
      instance_id: 'instance-b',
      whatsapp_account: 'account-b'
    })

    const entries = unifier.buildEntries([groupA, groupB], 'separate')

    expect(entries).toHaveLength(2)
    expect(entries[0].key).toContain('conversation:1203630@g.us:instance-')
    expect(entries[1].key).toContain('conversation:1203630@g.us:instance-')
  })

  it('finds a grouped entry by source contact id', () => {
    const contactA = createContact({
      id: 'c-1',
      phone_number: '+15550001111',
      whatsapp_account: 'account-a'
    })
    const contactB = createContact({
      id: 'c-2',
      phone_number: '+15550001111',
      whatsapp_account: 'account-b'
    })

    const entries = unifier.buildEntries([contactA, contactB], 'unified')
    const entry = unifier.findEntryByContactID(entries, 'c-2')

    expect(entry).toBeTruthy()
    expect(entry?.sourceContactIDs).toContain('c-2')
    expect(entry?.sourceContactIDs).toContain('c-1')
  })
})
