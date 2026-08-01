/**
 * WhatsApp Status (story) — virtual "conversation" in the sidebar.
 *
 * The Status feature posts to the well-known `status@broadcast` JID via a
 * dedicated backend path (see internal/handlers/status.go). It is not a real
 * conversation tied to a Contact row, so it is rendered in the sidebar as a
 * synthetic, always-present entry. Inbound status media is still hidden
 * (its download is unsupported by GOWA — see stores/contacts.ts
 * isNonChatContact), so this conversation is *send-only*: it shows a
 * session-local log of what the user posted.
 */
import type { Contact } from '@/stores/contacts'

/** Sentinel id for the virtual Status conversation. */
export const STATUS_CONTACT_ID = '__status__'

/**
 * The virtual Status entry shown at the top of the sidebar on every tab.
 * Fields are filled to satisfy the Contact type's non-optional members;
 * timestamps are stable ISO strings (the entry never represents a real row).
 */
export const STATUS_VIRTUAL_CONTACT: Contact = {
  id: STATUS_CONTACT_ID,
  phone_number: 'status@broadcast',
  name: 'Status',
  status: 'open',
  tags: [],
  metadata: { is_status: true },
  unread_count: 0,
  created_at: '1970-01-01T00:00:00.000Z',
  updated_at: '1970-01-01T00:00:00.000Z',
}

/** True when the id is the virtual Status conversation sentinel. */
export function isStatusContact(id: string | undefined | null): boolean {
  return id === STATUS_CONTACT_ID
}
