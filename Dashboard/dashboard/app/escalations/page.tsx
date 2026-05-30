"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { usePageContext } from "@/lib/page-context";
import { cn, formatRelativeTime, getStatusColor, getPriorityColor } from "@/lib/utils";
import { AlertTriangle, CheckCircle } from "lucide-react";
import { toast } from "sonner";

export default function EscalationsPage() {
  const qc = useQueryClient();
  const { selectedPageId } = usePageContext();
  const [page, setPage] = useState(1);
  const [status, setStatus] = useState("open");
  const [priority, setPriority] = useState("");
  const [resolveId, setResolveId] = useState<string | null>(null);
  const [notes, setNotes] = useState("");

  const params: Record<string, string> = { page: String(page), limit: "20" };
  if (status) params.status = status;
  if (priority) params.priority = priority;
  if (selectedPageId) params.page_id = selectedPageId;

  const { data, isLoading } = useQuery({
    queryKey: ["escalations", params],
    queryFn: () => api.getEscalations(params),
    refetchInterval: 15_000,
  });

  const resolveMutation = useMutation({
    mutationFn: ({ id, notes }: { id: string; notes: string }) => api.resolveEscalation(id, notes),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["escalations"] });
      qc.invalidateQueries({ queryKey: ["dashboard-stats"] });
      setResolveId(null);
      setNotes("");
      toast.success("تم حل التصعيد");
    },
    onError: () => toast.error("فشل الحل"),
  });

  return (
    <div className="p-6 space-y-5">
      <div>
        <h1 className="text-2xl font-bold text-foreground">التصعيدات</h1>
        <p className="text-sm text-muted-foreground mt-1">{data?.total ?? 0} تصعيد</p>
      </div>

      <div className="flex gap-3 flex-wrap">
        <select
          className="rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50"
          value={status}
          onChange={(e) => { setStatus(e.target.value); setPage(1); }}
        >
          <option value="">كل الحالات</option>
          <option value="open">مفتوح</option>
          <option value="resolved">مُحلّ</option>
        </select>
        <select
          className="rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50"
          value={priority}
          onChange={(e) => { setPriority(e.target.value); setPage(1); }}
        >
          <option value="">كل الأولويات</option>
          <option value="critical">حرج</option>
          <option value="high">عالي</option>
          <option value="medium">متوسط</option>
          <option value="low">منخفض</option>
        </select>
      </div>

      <div className="rounded-xl border border-border bg-card overflow-hidden">
        {isLoading ? (
          <div className="flex items-center justify-center h-48 text-muted-foreground">جاري التحميل...</div>
        ) : (
          <div className="divide-y divide-border">
            {(data?.data ?? []).map((esc) => (
              <div key={esc.id} className="p-4 hover:bg-accent/20 transition-colors">
                <div className="flex items-start justify-between gap-4">
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1 flex-wrap">
                      <span className="font-medium text-sm text-foreground">{esc.customer_name}</span>
                      <span className={cn("text-xs px-2 py-0.5 rounded-full border", getPriorityColor(esc.priority))}>{esc.priority}</span>
                      <span className={cn("text-xs px-2 py-0.5 rounded-full border", getStatusColor(esc.status))}>{esc.status}</span>
                      <span className="text-xs text-muted-foreground">{esc.page_name}</span>
                    </div>
                    <p className="text-sm text-foreground/80 mb-1">{esc.original_comment}</p>
                    <p className="text-xs text-orange-400/80 bg-orange-500/10 rounded px-2 py-1 inline-block">{esc.reason}</p>
                    <p className="text-xs text-muted-foreground mt-2">{formatRelativeTime(esc.created_at)}</p>
                    {esc.admin_notes && (
                      <p className="text-xs text-blue-400/80 mt-1 bg-blue-500/10 rounded px-2 py-1">ملاحظة: {esc.admin_notes}</p>
                    )}
                  </div>
                  {esc.status === "open" && (
                    <button
                      onClick={() => setResolveId(resolveId === esc.id ? null : esc.id)}
                      className="p-1.5 rounded-lg hover:bg-accent text-muted-foreground hover:text-green-400 transition-colors shrink-0"
                      title="حل"
                    >
                      <CheckCircle className="h-5 w-5" />
                    </button>
                  )}
                </div>
                {resolveId === esc.id && (
                  <div className="mt-3 flex gap-2">
                    <input
                      className="flex-1 rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50"
                      placeholder="ملاحظات الإدارة (اختياري)..."
                      value={notes}
                      onChange={(e) => setNotes(e.target.value)}
                    />
                    <button
                      onClick={() => resolveMutation.mutate({ id: esc.id, notes })}
                      disabled={resolveMutation.isPending}
                      className="rounded-lg bg-green-600 px-4 py-2 text-sm font-medium text-white disabled:opacity-50"
                    >
                      تأكيد
                    </button>
                  </div>
                )}
              </div>
            ))}
            {data?.data.length === 0 && (
              <div className="flex flex-col items-center justify-center h-48 gap-2 text-muted-foreground">
                <AlertTriangle className="h-8 w-8 opacity-40" />
                <p className="text-sm">لا توجد تصعيدات</p>
              </div>
            )}
          </div>
        )}
      </div>

      {data && data.total > 20 && (
        <div className="flex items-center justify-between">
          <p className="text-sm text-muted-foreground">صفحة {page} من {Math.ceil(data.total / 20)}</p>
          <div className="flex gap-2">
            <button disabled={page === 1} onClick={() => setPage(p => p - 1)} className="px-3 py-1.5 text-sm rounded-lg border border-border disabled:opacity-40 hover:bg-accent">السابق</button>
            <button disabled={page * 20 >= data.total} onClick={() => setPage(p => p + 1)} className="px-3 py-1.5 text-sm rounded-lg border border-border disabled:opacity-40 hover:bg-accent">التالي</button>
          </div>
        </div>
      )}
    </div>
  );
}
