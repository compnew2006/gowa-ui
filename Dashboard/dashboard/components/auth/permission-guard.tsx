"use client";

import { useEffect, useState } from "react";
import { ShieldAlert } from "lucide-react";
import { can, denyMessage, type PermissionKey } from "@/lib/permissions";
import { getStoredUser, type StoredUser } from "@/lib/session";

export function useCurrentUser() {
  const [user, setUser] = useState<StoredUser | null | undefined>(undefined);

  useEffect(() => {
    setUser(getStoredUser());
  }, []);

  return user;
}

export function useCan(permission: PermissionKey) {
  const user = useCurrentUser();
  if (user === undefined) return undefined;
  if (user?.role === "admin") return true;
  return can(permission, user?.permissions);
}

export function RequirePermission({
  permission,
  children,
}: {
  permission: PermissionKey;
  children: React.ReactNode;
}) {
  const allowed = useCan(permission);

  if (allowed === undefined) {
    return (
      <div className="flex min-h-screen items-center justify-center p-6 text-sm text-muted-foreground">
        جاري التحقق من الصلاحيات...
      </div>
    );
  }

  if (!allowed) {
    return (
      <div className="flex min-h-screen items-center justify-center p-6">
        <div className="max-w-md rounded-xl border border-border bg-card p-6 text-center shadow-sm">
          <ShieldAlert className="mx-auto h-8 w-8 text-yellow-400" />
          <h1 className="mt-3 text-lg font-semibold text-foreground">غير مصرح</h1>
          <p className="mt-2 text-sm text-muted-foreground">{denyMessage(permission)}</p>
        </div>
      </div>
    );
  }

  return <>{children}</>;
}
