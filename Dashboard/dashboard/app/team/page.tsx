"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { cn } from "@/lib/utils";
import { Users2, Plus, Trash2, Check, X, ShieldCheck, Eye, BarChart3 } from "lucide-react";
import { toast } from "sonner";
import { RequirePermission } from "@/components/auth/permission-guard";

const ROLE_CONFIG: Record<string, { label: string; color: string; icon: any }> = {
  admin: { label: "مشرف", color: "bg-purple-500/20 text-purple-400 border-purple-500/30", icon: ShieldCheck },
  reviewer: { label: "مراجع", color: "bg-blue-500/20 text-blue-400 border-blue-500/30", icon: Eye },
  analyst: { label: "محلل", color: "bg-green-500/20 text-green-400 border-green-500/30", icon: BarChart3 },
};

const PERM_LABELS: Record<string, string> = {
  can_approve: "الموافقة على الردود",
  can_reject: "رفض الردود",
  can_delete: "الحذف",
  can_manage_team: "إدارة الفريق",
  can_export: "تصدير البيانات",
};

function AddMemberModal({ onClose }: { onClose: () => void }) {
  const qc = useQueryClient();
  const [form, setForm] = useState({ name: "", email: "", role: "reviewer" });
  const mutation = useMutation({
    mutationFn: () => api.createTeamMember(form),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["team"] }); toast.success("تمت الإضافة"); onClose(); },
    onError: () => toast.error("فشلت الإضافة"),
  });

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60">
      <div className="w-full max-w-md rounded-xl border border-border bg-card p-6 space-y-4 shadow-xl">
        <div className="flex items-center justify-between">
          <h2 className="font-semibold text-foreground">إضافة عضو جديد</h2>
          <button onClick={onClose}><X className="h-4 w-4 text-muted-foreground" /></button>
        </div>
        {[
          { key: "name", label: "الاسم", placeholder: "أحمد خالد" },
          { key: "email", label: "البريد الإلكتروني", placeholder: "ahmed@company.com" },
        ].map(({ key, label, placeholder }) => (
          <div key={key} className="space-y-1">
            <label className="text-sm text-muted-foreground">{label}</label>
            <input
              className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50"
              placeholder={placeholder}
              value={(form as any)[key]}
              onChange={(e) => setForm(f => ({ ...f, [key]: e.target.value }))}
            />
          </div>
        ))}
        <div className="space-y-1">
          <label className="text-sm text-muted-foreground">الدور</label>
          <select
            className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none"
            value={form.role}
            onChange={(e) => setForm(f => ({ ...f, role: e.target.value }))}
          >
            <option value="admin">مشرف</option>
            <option value="reviewer">مراجع</option>
            <option value="analyst">محلل</option>
          </select>
        </div>
        <div className="flex gap-2 pt-2">
          <button
            onClick={() => mutation.mutate()}
            disabled={!form.name || !form.email || mutation.isPending}
            className="flex-1 rounded-lg bg-primary py-2 text-sm font-medium text-primary-foreground disabled:opacity-50"
          >
            {mutation.isPending ? "جاري..." : "إضافة"}
          </button>
          <button onClick={onClose} className="rounded-lg border border-border px-4 py-2 text-sm text-muted-foreground hover:bg-accent">إلغاء</button>
        </div>
      </div>
    </div>
  );
}

export default function TeamPage() {
  const qc = useQueryClient();
  const [showAdd, setShowAdd] = useState(false);
  const [auditPage, setAuditPage] = useState(1);

  const { data: members = [], isLoading } = useQuery({
    queryKey: ["team"],
    queryFn: () => api.getTeamMembers(),
  });

  const { data: auditData } = useQuery({
    queryKey: ["audit-log", auditPage],
    queryFn: () => api.getAuditLog({ page: auditPage }),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.deleteTeamMember(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["team"] }); toast.success("تم الحذف"); },
    onError: () => toast.error("فشل الحذف"),
  });

  const toggleActiveMutation = useMutation({
    mutationFn: ({ id, is_active }: { id: string; is_active: boolean }) =>
      api.updateTeamMember(id, { is_active }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["team"] }),
  });

  return (
    <RequirePermission permission="can_manage_team">
    <div className="p-6 space-y-6">
      {showAdd && <AddMemberModal onClose={() => setShowAdd(false)} />}

      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">الفريق</h1>
          <p className="text-sm text-muted-foreground mt-1">إدارة أعضاء الفريق والصلاحيات</p>
        </div>
        <button
          onClick={() => setShowAdd(true)}
          className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground"
        >
          <Plus className="h-4 w-4" />إضافة عضو
        </button>
      </div>

      {/* Members grid */}
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
        {isLoading ? (
          <div className="col-span-3 flex h-48 items-center justify-center text-muted-foreground">جاري التحميل...</div>
        ) : members.length === 0 ? (
          <div className="col-span-3 flex flex-col items-center justify-center h-48 gap-2 text-muted-foreground">
            <Users2 className="h-8 w-8 opacity-40" />
            <p className="text-sm">لا يوجد أعضاء حتى الآن</p>
          </div>
        ) : (
          members.map((m: any) => {
            const roleConf = ROLE_CONFIG[m.role] ?? ROLE_CONFIG.analyst;
            const RoleIcon = roleConf.icon;
            return (
              <div key={m.id} className="rounded-xl border border-border bg-card p-5 space-y-3">
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-3">
                    <div className="h-10 w-10 rounded-full bg-primary/20 flex items-center justify-center text-sm font-bold text-primary">
                      {m.name[0].toUpperCase()}
                    </div>
                    <div>
                      <p className="font-medium text-foreground">{m.name}</p>
                      <p className="text-xs text-muted-foreground">{m.email}</p>
                    </div>
                  </div>
                  <div className="flex gap-1">
                    <button
                      onClick={() => toggleActiveMutation.mutate({ id: m.id, is_active: !m.is_active })}
                      className={cn("p-1.5 rounded-lg transition-colors", m.is_active ? "text-green-400 hover:bg-green-500/10" : "text-muted-foreground hover:bg-accent")}
                      title={m.is_active ? "تعطيل" : "تفعيل"}
                    >
                      {m.is_active ? <Check className="h-4 w-4" /> : <X className="h-4 w-4" />}
                    </button>
                    <button
                      onClick={() => deleteMutation.mutate(m.id)}
                      className="p-1.5 rounded-lg text-muted-foreground hover:bg-red-500/10 hover:text-red-400 transition-colors"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  </div>
                </div>

                <div className="flex items-center justify-between">
                  <span className={cn("flex items-center gap-1.5 text-xs px-2.5 py-1 rounded-full border", roleConf.color)}>
                    <RoleIcon className="h-3 w-3" />{roleConf.label}
                  </span>
                  <span className={cn("text-xs px-2 py-0.5 rounded-full", m.is_active ? "bg-green-500/15 text-green-400" : "bg-gray-500/15 text-gray-400")}>
                    {m.is_active ? "نشط" : "موقوف"}
                  </span>
                </div>

                <div className="pt-2 border-t border-border space-y-1">
                  {Object.entries(m.permissions ?? {}).map(([key, val]) => (
                    <div key={key} className="flex items-center justify-between text-xs">
                      <span className="text-muted-foreground">{PERM_LABELS[key] ?? key}</span>
                      <span className={val ? "text-green-400" : "text-red-400"}>{val ? "✓" : "✗"}</span>
                    </div>
                  ))}
                </div>
              </div>
            );
          })
        )}
      </div>

      {/* Audit Log */}
      <div className="rounded-xl border border-border bg-card overflow-hidden">
        <div className="p-5 border-b border-border flex items-center gap-2">
          <ShieldCheck className="h-4 w-4 text-muted-foreground" />
          <h2 className="font-semibold text-foreground">سجل المراجعة</h2>
          {auditData && <span className="ml-2 text-xs text-muted-foreground">({auditData.total} إجمالاً)</span>}
        </div>
        <div className="divide-y divide-border">
          {(auditData?.data ?? []).map((log: any) => (
            <div key={log.id} className="flex items-center gap-4 px-5 py-3 hover:bg-accent/20 transition-colors">
              <div className="text-xs text-muted-foreground w-36 shrink-0">{new Date(log.created_at).toLocaleString("ar-SA")}</div>
              <div className="flex-1 min-w-0">
                <span className="text-sm font-medium text-foreground">{log.admin_name}</span>
                <span className="mx-1 text-muted-foreground">·</span>
                <span className="text-sm text-muted-foreground">{log.action}</span>
                <span className="mx-1 text-muted-foreground">·</span>
                <span className="text-xs text-muted-foreground">{log.entity_type} {log.entity_id ? `(${log.entity_id.slice(0, 8)}...)` : ""}</span>
              </div>
            </div>
          ))}
          {(auditData?.data ?? []).length === 0 && (
            <div className="flex h-24 items-center justify-center text-muted-foreground text-sm">لا يوجد سجل</div>
          )}
        </div>
        {auditData && auditData.total > 50 && (
          <div className="flex justify-center gap-2 p-3 border-t border-border">
            <button onClick={() => setAuditPage(p => Math.max(1, p - 1))} disabled={auditPage === 1} className="px-3 py-1 text-xs rounded border border-border disabled:opacity-40">السابق</button>
            <span className="text-xs text-muted-foreground self-center">صفحة {auditPage}</span>
            <button onClick={() => setAuditPage(p => p + 1)} disabled={auditPage * 50 >= auditData.total} className="px-3 py-1 text-xs rounded border border-border disabled:opacity-40">التالي</button>
          </div>
        )}
      </div>
    </div>
    </RequirePermission>
  );
}
