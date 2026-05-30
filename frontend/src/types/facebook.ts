export interface FacebookAccount {
  id: string;
  name: string;
  account_uid: string;
  status: "active" | "inactive" | "closed";
  method: "cookies" | "credentials";
  data: Record<string, any>;
  has_cookies: boolean;
  created_at: string;
  updated_at: string;
}
