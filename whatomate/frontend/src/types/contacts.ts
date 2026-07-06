export type ChatStatus = 'pending' | 'open' | 'closed'
export type ChatBucketTab = 'pending' | 'assigned'
export type ChatTypeFilter = 'private' | 'group' | 'channel'

export interface Contact {
  id: string
  phone_number: string
  instance_id?: string
  conversation_id?: string
  is_group_chat?: boolean
  name: string
  profile_name?: string
  avatar_url?: string
  status: ChatStatus
  tags: string[]
  metadata: Record<string, unknown>
  last_message_at?: string
  last_message_preview?: string
  last_inbound_at?: string
  service_window_open?: boolean
  unread_count: number
  assigned_user_id?: string
  assigned_user_name?: string
  is_public?: boolean
  is_collaborator?: boolean
  closed_at?: string
  closed_by_user_id?: string
  closed_by_name?: string
  whatsapp_account?: string
  created_at: string
  updated_at: string
}

export interface ReplyPreview {
  id: string
  content: unknown
  message_type: string
  direction: 'incoming' | 'outgoing'
  sender_phone?: string
  media_url?: string
  media_mime_type?: string
  media_filename?: string
}

export interface Reaction {
  emoji: string
  from_phone?: string
  from_user?: string
}

export interface Message {
  id: string
  contact_id: string
  conversation_id?: string
  is_group_chat?: boolean
  sender_phone?: string
  sender_push_name?: string
  direction: 'incoming' | 'outgoing'
  message_type: string
  content: unknown
  media_url?: string
  media_mime_type?: string
  media_filename?: string
  interactive_data?: {
    type?: string
    body?: string
    question?: string
    name?: string
    title?: string
    options?: unknown[]
    poll_options?: unknown[]
    max_selections?: number | string
    selectable_options_count?: number | string
    votes?: Record<string, number>
    vote_counts?: Record<string, number>
    total_votes?: number | string
    selected_options?: string[]
    last_selected_options?: string[]
    voters?: Record<string, string[]>
    poll?: Record<string, unknown>
    buttons?: Array<{
      type?: string
      reply?: { id: string; title: string }
      id?: string
      title?: string
    }>
    rows?: Array<{
      id?: string
      title?: string
    }>
  }
  status: string
  wamid?: string
  error_message?: string
  is_reply?: boolean
  reply_to_message_id?: string
  reply_to_message?: ReplyPreview
  reactions?: Reaction[]
  instance_id?: string
  metadata?: Record<string, unknown>
  whatsapp_account?: string
  created_at: string
  updated_at: string
}
