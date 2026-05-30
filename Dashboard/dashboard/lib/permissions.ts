export type PermissionKey =
  | "can_approve"
  | "can_reject"
  | "can_delete"
  | "can_manage_team"
  | "can_export"
  | "can_manage_settings"
  | "can_manage_campaigns";

export type PermissionMap = Partial<Record<PermissionKey | string, boolean>>;

const DEFAULT_PERMISSIONS: Record<PermissionKey, boolean> = {
  can_approve: false,
  can_reject: false,
  can_delete: false,
  can_manage_team: false,
  can_export: false,
  can_manage_settings: false,
  can_manage_campaigns: false,
};

export function can(permission: PermissionKey, permissions?: PermissionMap | null): boolean {
  if (!permissions) return DEFAULT_PERMISSIONS[permission];
  return permissions[permission] === true;
}

export function denyMessage(permission: PermissionKey): string {
  const labels: Record<PermissionKey, string> = {
    can_approve: "ليست لديك صلاحية الموافقة",
    can_reject: "ليست لديك صلاحية الرفض",
    can_delete: "ليست لديك صلاحية الحذف",
    can_manage_team: "ليست لديك صلاحية إدارة الفريق",
    can_export: "ليست لديك صلاحية التصدير",
    can_manage_settings: "ليست لديك صلاحية إدارة الإعدادات",
    can_manage_campaigns: "ليست لديك صلاحية إدارة الحملات",
  };
  return labels[permission];
}
