export interface WhatsAppInstanceSettings {
    auto_sync_history?: boolean
    auto_reject_calls?: {
        enabled?: boolean
        mode?: 'without_message' | 'with_message'
        message?: string
        reject_individual_calls?: boolean
        reject_group_calls?: boolean
        bypass_contacts?: string[]
        schedule?: {
            type?: 'always' | 'custom_hours' | 'while_in_other_calls'
            start?: string
            end?: string
            days?: number[]
            timezone?: string
        }
    }
    auto_campaign?: {
        enabled?: boolean
        name_prefix?: string
        message?: string
        interval_days?: number
        min_delay_minutes?: number
        max_delay_minutes?: number
        target_status?: 'draft' | 'run'
        media_local_path?: string
        media_mime_type?: string
        media_filename?: string
        last_generated_at?: string
    }
    chat_tag_custom_label?: string
    chat_tag_color?: string
    chat_tag_display_mode?: 'name' | 'phone' | 'custom'
    [key: string]: any
}

export interface WhatsAppInstance {
    id: string
    name: string
    status: string
    jid?: string
    phone_number?: string
    is_default: boolean
    auto_read_receipt: boolean
    organization_id: string
    settings?: WhatsAppInstanceSettings
    health?: InstanceHealth
    created_at: string
    updated_at: string
}

export interface InstanceHealth {
    uptime_seconds: number
    messages_sent_today: number
    messages_received_today: number
    messages_failed_today: number
    error_rate_percent: number
    queue_depth: number
}

export interface InstanceNotification {
    id: string
    organization_id: string
    instance_id: string
    event_type: string
    message: string
    is_dismissed: boolean
    created_at: string
    updated_at: string
    instance?: {
        id: string
        name: string
    }
}
