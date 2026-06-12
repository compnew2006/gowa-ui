export interface FacebookAccountPage {
  id: string;
  name?: string;
  category?: string;
  picture?: {
    data?: {
      url?: string;
    };
  };
  connected?: boolean;
  [key: string]: any;
}

export interface FacebookAccountData {
  pages?: FacebookAccountPage[];
  page_count?: number;
  [key: string]: any;
}

export interface FacebookAccount {
  id: string;
  name: string;
  account_uid: string;
  platform: string;
  email?: string;
  avatar_url?: string;
  status: "active" | "inactive" | "closed" | "expired" | "revoked";
  method: "cookies" | "credentials" | "oauth";
  data: FacebookAccountData;
  has_cookies: boolean;
  oauth_connected: boolean;
  token_expires_at?: string;
  connected_at?: string;
  last_renewed_at?: string;
  page_count: number;
  created_at: string;
  updated_at: string;
}
