export interface ChatBackgroundSettings {
  kind: "preset" | "custom";
  preset_id?: string;
  custom_asset_id?: string;
  custom_filename?: string;
  custom_mime_type?: "image/jpeg" | "image/png" | "image/webp";
}

export interface UserSettings {
  email_notifications?: boolean;
  new_message_alerts?: boolean;
  campaign_updates?: boolean;
  notification_sound?: "notification1" | "notification2" | "notification";
  chat_background?: ChatBackgroundSettings;
  send_restrictions?: {
    enabled?: boolean;
    include_all_contacts?: boolean;
    authorized_numbers?: string[];
    allowed_instance_ids?: string[];
    allowed_instance_id?: string | null;
    prefix_agent_name?: boolean;
    allow_unclaimed_chat_view?: boolean;
    allow_unclaimed_chat_send?: boolean;
  };
}

export interface Permission {
  id?: string;
  resource: string;
  action: string;
  description?: string;
  key?: string;
}

export interface UserRole {
  id: string;
  name: string;
  description?: string;
  is_system?: boolean;
  permissions?: Permission[] | string[];
}

export interface User {
  id: string;
  email: string;
  full_name: string;
  role_id?: string;
  role?: UserRole;
  organization_id: string;
  organization_name?: string;
  settings?: UserSettings;
  is_available?: boolean;
  is_super_admin?: boolean;
  is_active?: boolean;
  is_member?: boolean;
  created_at?: string;
  updated_at?: string;
}
