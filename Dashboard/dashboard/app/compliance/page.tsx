"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { cn } from "@/lib/utils";
import { ShieldCheck, Search, Download, Trash2, Eye, AlertTriangle, Check, FileText, Database } from "lucide-react";
import { toast } from "sonner";
import { RequirePermission } from "@/components/auth/permission-guard";

export default function CompliancePage() {
  const qc = useQueryClient();
  const [convId, setConvId] = useState("");
  const [customerId, setCustomerId] = useState("");
  const [piiResult, setPiiResult] = useState<any>(null);
  const [exportResult, setExportResult] = useState<any>(null);
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null);

  const { data: retention } = useQuery({
    queryKey: ["compliance-retention"],
    queryFn: () => api.getDataRetention(),
  });

  const { data: auditSummary } = useQuery({
    queryKey: ["compliance-audit-summary"],
    queryFn: () => api.getAuditSummary(),
  });

  const piiScanMutation = useMutation({
    mutationFn: (id: string) => api.scanPII(id),
    onSuccess: (data) => setPiiResult(data),
    onError: () => toast.error("فشل الفحص"),
  });

  const exportMutation = useMutation({
    mutationFn: (id: string) => api.gdprExport(id),
    onSuccess: (data) => {
      setExportResult(data);
      toast.success("تم تصدير البيانات");
    },
    onError: () => toast.error("فشل التصدير"),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.gdprDelete(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["compliance-retention"] }); toast.success("تم حذف بيانات العميل (GDPR)"); setDeleteConfirm(null); },
    onError: () => toast.error("فشل الحذف"),
  });

  const downloadExport = () => {
    if (!exportResult) return;
    const blob = new Blob([JSON.stringify(exportResult, null, 2)], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const a = document.createElement("a"); a.href = url; a.download = `gdpr-export-${customerId}.json`; a.click();
    URL.revokeObjectURL(url);
  };

  return (
    <RequirePermission permission="can_export">
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">الامتثال والأمان</h1>
        <p className="text-sm text-muted-foreground mt-1">GDPR، كشف PII، الاحتفاظ بالبيانات، سجل المراجعة</p>
      </div>

      {/* Data Retention Stats */}
      <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        {[
          { title: "سياسة الاحتفاظ", value: `${retention?.retention_policy_days ?? 90} يوم`, icon: Database, color: "bg-blue-500/20 text-blue-400" },
          { title: "محادثات قديمة +90 يوم", value: retention?.resolved_older_than_90d ?? "—", icon: AlertTriangle, color: "bg-yellow-500/20 text-yellow-400" },
          { title: "إجمالي العملاء", value: retention?.total_customers ?? "—", icon: ShieldCheck, color: "bg-purple-500/20 text-purple-400" },
          { title: "محذوف (GDPR)", value: retention?.gdpr_deleted_customers ?? "—", icon: Trash2, color: "bg-red-500/20 text-red-400" },
        ].map((item) => (
          <div key={item.title} className="rounded-xl border border-border bg-card p-4">
            <div className="flex items-start justify-between">
              <div>
                <p className="text-xs text-muted-foreground">{item.title}</p>
                <p className="mt-1 text-2xl font-bold text-foreground">{item.value}</p>
              </div>
              <div className={cn("rounded-lg p-2", item.color)}>
                <item.icon className="h-4 w-4" />
              </div>
            </div>
          </div>
        ))}
      </div>

      {retention?.recommendation && retention.recommendation !== "OK" && (
        <div className="flex items-center gap-3 rounded-lg border border-yellow-500/30 bg-yellow-500/10 px-4 py-3">
          <AlertTriangle className="h-4 w-4 text-yellow-400 shrink-0" />
          <p className="text-sm text-yellow-300">{retention.recommendation}</p>
        </div>
      )}

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        {/* PII Scanner */}
        <div className="rounded-xl border border-border bg-card p-5 space-y-4">
          <div className="flex items-center gap-2">
            <Eye className="h-4 w-4 text-muted-foreground" />
            <h2 className="font-semibold text-foreground">فاحص PII</h2>
          </div>
          <p className="text-xs text-muted-foreground">كشف الهواتف والبريد الإلكتروني وأرقام البطاقات في المحادثات</p>
          <div className="flex gap-2">
            <input
              className="flex-1 rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50"
              placeholder="معرف المحادثة..."
              value={convId}
              onChange={e => setConvId(e.target.value)}
            />
            <button
              onClick={() => convId && piiScanMutation.mutate(convId)}
              disabled={!convId || piiScanMutation.isPending}
              className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground disabled:opacity-50"
            >
              <Search className="h-4 w-4" />
            </button>
          </div>

          {piiResult && (
            <div className="space-y-3 p-3 rounded-lg bg-muted/30 border border-border">
              {[
                { label: "التعليق", detected: piiResult.comment_pii.detected, types: piiResult.comment_pii.types, masked: piiResult.masked_comment },
                { label: "الرد", detected: piiResult.reply_pii.detected, types: piiResult.reply_pii.types, masked: piiResult.masked_reply },
              ].map((r) => (
                <div key={r.label} className="space-y-1">
                  <div className="flex items-center gap-2">
                    <span className="text-xs text-muted-foreground">{r.label}:</span>
                    {r.detected ? (
                      <span className="text-xs text-red-400 flex items-center gap-1">
                        <AlertTriangle className="h-3 w-3" />PII مكتشف: {r.types.join(", ")}
                      </span>
                    ) : (
                      <span className="text-xs text-green-400 flex items-center gap-1"><Check className="h-3 w-3" />لا يوجد PII</span>
                    )}
                  </div>
                  {r.masked && <p className="text-xs bg-accent/50 rounded p-2 text-muted-foreground break-all">{r.masked}</p>}
                </div>
              ))}
            </div>
          )}
        </div>

        {/* GDPR Export/Delete */}
        <div className="rounded-xl border border-border bg-card p-5 space-y-4">
          <div className="flex items-center gap-2">
            <FileText className="h-4 w-4 text-muted-foreground" />
            <h2 className="font-semibold text-foreground">GDPR — حقوق البيانات</h2>
          </div>
          <p className="text-xs text-muted-foreground">تصدير أو حذف بيانات العميل وفق اللائحة الأوروبية</p>
          <div className="flex gap-2">
            <input
              className="flex-1 rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50"
              placeholder="معرف العميل..."
              value={customerId}
              onChange={e => setCustomerId(e.target.value)}
            />
          </div>
          <div className="flex gap-2">
            <button
              onClick={() => customerId && exportMutation.mutate(customerId)}
              disabled={!customerId || exportMutation.isPending}
              className="flex items-center gap-2 rounded-lg bg-blue-500/15 border border-blue-500/30 px-4 py-2 text-sm font-medium text-blue-400 hover:bg-blue-500/25 disabled:opacity-50 transition-colors"
            >
              <Download className="h-4 w-4" />تصدير البيانات
            </button>
            <button
              onClick={() => customerId && setDeleteConfirm(customerId)}
              disabled={!customerId}
              className="flex items-center gap-2 rounded-lg bg-red-500/15 border border-red-500/30 px-4 py-2 text-sm font-medium text-red-400 hover:bg-red-500/25 disabled:opacity-50 transition-colors"
            >
              <Trash2 className="h-4 w-4" />حذف (حق الطيّ)
            </button>
          </div>
          {exportResult && (
            <div className="p-3 rounded-lg bg-blue-500/5 border border-blue-500/20 space-y-2">
              <p className="text-xs text-blue-400 font-medium">تم التصدير بنجاح</p>
              <p className="text-xs text-muted-foreground">{exportResult.conversations?.length ?? 0} محادثة · {exportResult.escalations?.length ?? 0} تصعيد</p>
              <button onClick={downloadExport} className="flex items-center gap-1.5 text-xs text-blue-400 hover:underline">
                <Download className="h-3 w-3" />تحميل JSON
              </button>
            </div>
          )}
          {deleteConfirm && (
            <div className="p-3 rounded-lg bg-red-500/10 border border-red-500/30 space-y-2">
              <p className="text-sm text-red-400 font-medium">هل أنت متأكد من حذف بيانات العميل؟</p>
              <p className="text-xs text-muted-foreground">لا يمكن التراجع عن هذه العملية.</p>
              <div className="flex gap-2">
                <button onClick={() => deleteMutation.mutate(deleteConfirm)} disabled={deleteMutation.isPending}
                  className="rounded-lg bg-red-500 text-white px-3 py-1.5 text-xs font-medium disabled:opacity-50">
                  {deleteMutation.isPending ? "جاري..." : "تأكيد الحذف"}
                </button>
                <button onClick={() => setDeleteConfirm(null)} className="rounded-lg border border-border px-3 py-1.5 text-xs text-muted-foreground hover:bg-accent">إلغاء</button>
              </div>
            </div>
          )}
        </div>
      </div>

      {/* Audit Action Summary */}
      <div className="rounded-xl border border-border bg-card p-5">
        <div className="flex items-center gap-2 mb-4">
          <ShieldCheck className="h-4 w-4 text-muted-foreground" />
          <h2 className="font-semibold text-foreground">ملخص سجل المراجعة</h2>
          {auditSummary && <span className="ml-2 text-xs text-muted-foreground">({auditSummary.total_audit_events} حدث)</span>}
        </div>
        <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-5">
          {(auditSummary?.by_action ?? []).slice(0, 10).map((item: any) => (
            <div key={item.action} className="rounded-lg bg-accent/30 p-3 text-center">
              <p className="text-xl font-bold text-foreground">{item.count}</p>
              <p className="text-xs text-muted-foreground mt-1 break-all">{item.action}</p>
            </div>
          ))}
          {(auditSummary?.by_action ?? []).length === 0 && (
            <div className="col-span-5 flex h-24 items-center justify-center text-muted-foreground text-sm">لا توجد أحداث</div>
          )}
        </div>
      </div>
    </div>
    </RequirePermission>
  );
}
