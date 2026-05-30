"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { cn, formatDateTime, getStatusColor } from "@/lib/utils";
import { Key, RefreshCw, CheckCircle, AlertTriangle, XCircle, Facebook, Instagram } from "lucide-react";
import { toast } from "sonner";
import { RequirePermission } from "@/components/auth/permission-guard";

const STATUS_ICONS: Record<string, React.ReactNode> = {
  valid: <CheckCircle className="h-4 w-4 text-green-400" />,
  expiring_soon: <AlertTriangle className="h-4 w-4 text-yellow-400" />,
  expired: <XCircle className="h-4 w-4 text-red-400" />,
  error: <XCircle className="h-4 w-4 text-red-400" />,
};

export default function TokensPage() {
  const qc = useQueryClient();

  const { data: tokens, isLoading } = useQuery({
    queryKey: ["tokens"],
    queryFn: () => api.getTokens(),
    refetchInterval: 60_000,
  });

  const refreshMutation = useMutation({
    mutationFn: (id: string) => api.refreshToken(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["tokens"] }); toast.success("تم تحديث الرمز"); },
    onError: (e: any) => toast.error(e.message ?? "فشل التحديث"),
  });

  const healthy = tokens?.filter(t => t.token_status === "valid").length ?? 0;
  const expiring = tokens?.filter(t => t.token_status === "expiring_soon").length ?? 0;
  const expired = tokens?.filter(t => ["expired", "error"].includes(t.token_status ?? "")).length ?? 0;

  return (
    <RequirePermission permission="can_manage_settings">
    <div className="p-6 space-y-5">
      <div>
        <h1 className="text-2xl font-bold text-foreground">صحة الرموز</h1>
        <p className="text-sm text-muted-foreground mt-1">مراقبة رموز الوصول لصفحات Meta</p>
      </div>

      {/* Summary */}
      <div className="grid grid-cols-3 gap-4">
        <div className="rounded-xl border border-green-500/20 bg-green-500/5 p-4 text-center">
          <CheckCircle className="h-6 w-6 text-green-400 mx-auto mb-2" />
          <p className="text-2xl font-bold text-green-400">{healthy}</p>
          <p className="text-xs text-muted-foreground mt-1">صالح</p>
        </div>
        <div className="rounded-xl border border-yellow-500/20 bg-yellow-500/5 p-4 text-center">
          <AlertTriangle className="h-6 w-6 text-yellow-400 mx-auto mb-2" />
          <p className="text-2xl font-bold text-yellow-400">{expiring}</p>
          <p className="text-xs text-muted-foreground mt-1">تنتهي قريباً</p>
        </div>
        <div className="rounded-xl border border-red-500/20 bg-red-500/5 p-4 text-center">
          <XCircle className="h-6 w-6 text-red-400 mx-auto mb-2" />
          <p className="text-2xl font-bold text-red-400">{expired}</p>
          <p className="text-xs text-muted-foreground mt-1">منتهي / خطأ</p>
        </div>
      </div>

      {/* Token list */}
      <div className="rounded-xl border border-border bg-card overflow-hidden">
        {isLoading ? (
          <div className="flex items-center justify-center h-48 text-muted-foreground">جاري التحميل...</div>
        ) : (
          <div className="divide-y divide-border">
            {(tokens ?? []).map(token => (
              <div key={token.id} className="flex items-center gap-4 p-4 hover:bg-accent/20 transition-colors">
                <div className={cn("h-9 w-9 rounded-lg flex items-center justify-center shrink-0", token.platform === "facebook" ? "bg-blue-500/20" : "bg-pink-500/20")}>
                  {token.platform === "facebook" ? <Facebook className="h-4 w-4 text-blue-400" /> : <Instagram className="h-4 w-4 text-pink-400" />}
                </div>
                <div className="flex-1 min-w-0">
                  <p className="font-medium text-sm text-foreground">{token.name}</p>
                  <p className="text-xs text-muted-foreground">
                    {token.token_expires_at ? `تنتهي: ${formatDateTime(token.token_expires_at)}` : "لا يوجد تاريخ انتهاء"}
                  </p>
                  {token.token_last_error && (
                    <p className="text-xs text-red-400 mt-0.5">{token.token_last_error}</p>
                  )}
                </div>
                <div className="flex items-center gap-3 shrink-0">
                  <div className="flex items-center gap-1.5">
                    {STATUS_ICONS[token.token_status ?? "valid"] ?? <Key className="h-4 w-4 text-muted-foreground" />}
                    <span className={cn("text-xs px-2 py-0.5 rounded-full border", getStatusColor(token.token_status ?? "valid"))}>
                      {token.token_status ?? "غير محدد"}
                    </span>
                  </div>
                  <button
                    onClick={() => refreshMutation.mutate(token.id)}
                    disabled={refreshMutation.isPending}
                    className="p-1.5 rounded-lg hover:bg-accent text-muted-foreground hover:text-primary transition-colors disabled:opacity-40"
                    title="تحديث الرمز"
                  >
                    <RefreshCw className={cn("h-4 w-4", refreshMutation.isPending && "animate-spin")} />
                  </button>
                </div>
              </div>
            ))}
            {(tokens ?? []).length === 0 && (
              <div className="flex flex-col items-center justify-center h-48 gap-2 text-muted-foreground">
                <Key className="h-8 w-8 opacity-40" />
                <p className="text-sm">لا توجد صفحات مكوّنة</p>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
    </RequirePermission>
  );
}
