"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api, type Page } from "@/lib/api";
import { cn, getStatusColor } from "@/lib/utils";
import { Globe, Plus, Pencil, Trash2, Facebook, Instagram } from "lucide-react";
import { toast } from "sonner";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { SecretInput } from "@/components/ui/secret-input";
import { RequirePermission } from "@/components/auth/permission-guard";
import { DatePicker } from "@/components/ui/date-picker";

function PageForm({ initial, onSave, onCancel, saving }: {
  initial?: Partial<Page>; onSave: (d: Parameters<typeof api.createPage>[0]) => void; onCancel: () => void; saving: boolean;
}) {
  const [form, setForm] = useState({
    platform: initial?.platform ?? "facebook",
    page_id: initial?.page_id ?? "",
    name: initial?.name ?? "",
    avatar_url: initial?.avatar_url ?? "",
    is_active: initial?.is_active ?? true,
    auto_reply_enabled: initial?.auto_reply_enabled ?? false,
    shadow_mode: initial?.shadow_mode ?? true,
    access_token_encrypted: "",
    auto_reply_end_date: initial?.auto_reply_end_date ?? "",
  });

  const handleSave = () => {
    if (form.auto_reply_enabled && !form.auto_reply_end_date) {
      toast.error("التاريخ غير محدد");
      return;
    }
    const payload = {
      ...form,
      auto_reply_end_date: form.auto_reply_enabled ? form.auto_reply_end_date : null
    };
    onSave(payload as any);
  };

  return (
    <div className="rounded-xl border border-primary/30 bg-primary/5 p-5 space-y-4">
      <h3 className="font-semibold text-foreground">{initial ? "تعديل الصفحة" : "إضافة صفحة جديدة"}</h3>
      <div className="grid grid-cols-2 gap-4">
        <div>
          <label className="text-xs text-muted-foreground mb-1 block">المنصة</label>
          <select className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground" value={form.platform} onChange={e => setForm(f => ({ ...f, platform: e.target.value }))}>
            <option value="facebook">Facebook</option>
            <option value="instagram">Instagram</option>
          </select>
        </div>
        <div>
          <label className="text-xs text-muted-foreground mb-1 block">Page ID</label>
          <input className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground" value={form.page_id} onChange={e => setForm(f => ({ ...f, page_id: e.target.value }))} placeholder="12345678..." />
        </div>
        <div>
          <label className="text-xs text-muted-foreground mb-1 block">اسم الصفحة</label>
          <input className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground" value={form.name} onChange={e => setForm(f => ({ ...f, name: e.target.value }))} placeholder="اسم الصفحة..." />
        </div>
        <div>
          <label className="text-xs text-muted-foreground mb-1 block">رمز الوصول</label>
          <SecretInput
            saved={Boolean(initial)}
            className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground"
            value={form.access_token_encrypted}
            onChange={value => setForm(f => ({ ...f, access_token_encrypted: value }))}
            placeholder="EAABxx..."
          />
        </div>
      </div>
      <div className="flex gap-6">
        {[
          { key: "is_active", label: "نشط" },
          { key: "auto_reply_enabled", label: "رد تلقائي" },
          { key: "shadow_mode", label: "وضع الظل" },
        ].map(({ key, label }) => (
          <label key={key} className="flex items-center gap-2 cursor-pointer">
            <input type="checkbox" checked={(form as any)[key]} onChange={e => setForm(f => ({ ...f, [key]: e.target.checked }))} className="rounded" />
            <span className="text-sm text-foreground">{label}</span>
          </label>
        ))}
      </div>
      {form.auto_reply_enabled && (
        <div className="rounded-lg border border-indigo-500/20 bg-indigo-500/5 p-4 space-y-2 animate-in slide-in-from-top-2 duration-200">
          <div className="flex flex-col sm:flex-row sm:items-center justify-between gap-2">
            <div>
              <span className="text-xs text-indigo-400 font-semibold block">صلاحية الرد التلقائي</span>
              <span className="text-xs text-muted-foreground">يتم تفعيل البوت من تاريخ اليوم كبداية تلقائية. يجب تحديد تاريخ انتهاء الصلاحية.</span>
            </div>
          </div>
          <div className="pt-1">
            <label className="text-xs text-muted-foreground mb-1 block font-semibold">تاريخ النهاية (إلى) <span className="text-red-500">*</span></label>
            <DatePicker 
              value={form.auto_reply_end_date} 
              onChange={date => setForm(f => ({ ...f, auto_reply_end_date: date ?? "" }))} 
              placeholder="اختر تاريخ انتهاء الصلاحية..." 
            />
          </div>
        </div>
      )}
      <div className="flex gap-2 justify-end">
        <button onClick={onCancel} className="px-4 py-2 text-sm rounded-lg border border-border hover:bg-accent text-muted-foreground">إلغاء</button>
        <button onClick={handleSave} disabled={!form.name || !form.page_id || saving} className="px-4 py-2 text-sm rounded-lg bg-primary text-primary-foreground disabled:opacity-50">حفظ</button>
      </div>
    </div>
  );
}

export default function PagesPage() {
  const qc = useQueryClient();
  const [showCreate, setShowCreate] = useState(false);
  const [editId, setEditId] = useState<string | null>(null);
  const [deleteId, setDeleteId] = useState<string | null>(null);

  const { data: pages, isLoading } = useQuery({
    queryKey: ["pages"],
    queryFn: () => api.getPages(),
  });

  const createMutation = useMutation({
    mutationFn: (d: any) => api.createPage(d),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["pages"] }); setShowCreate(false); toast.success("تمت إضافة الصفحة"); },
    onError: (err: any) => toast.error(err.message || "فشل الإضافة"),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: any) => api.updatePage(id, data),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["pages"] }); setEditId(null); toast.success("تم التحديث"); },
    onError: (err: any) => toast.error(err.message || "فشل التحديث"),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.deletePage(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["pages"] }); setDeleteId(null); toast.success("تم الحذف"); },
    onError: () => toast.error("فشل الحذف"),
  });

  return (
    <RequirePermission permission="can_manage_settings">
    <div className="p-6 space-y-5">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">الصفحات</h1>
          <p className="text-sm text-muted-foreground mt-1">{pages?.length ?? 0} صفحة</p>
        </div>
        <button onClick={() => setShowCreate(true)} className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90">
          <Plus className="h-4 w-4" />إضافة صفحة
        </button>
      </div>

      {showCreate && <PageForm onSave={(d) => createMutation.mutate(d)} onCancel={() => setShowCreate(false)} saving={createMutation.isPending} />}

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        {isLoading ? (
          <div className="col-span-2 flex items-center justify-center h-48 text-muted-foreground">جاري التحميل...</div>
        ) : (
          (pages ?? []).map(page => (
            <div key={page.id}>
              {editId === page.id ? (
                <PageForm
                  initial={page}
                  onSave={(d) => updateMutation.mutate({ id: page.id, data: d })}
                  onCancel={() => setEditId(null)}
                  saving={updateMutation.isPending}
                />
              ) : (
                <div className="rounded-xl border border-border bg-card p-5">
                  <div className="flex items-start justify-between mb-3">
                    <div className="flex items-center gap-3">
                      <div className={cn("h-10 w-10 rounded-lg flex items-center justify-center", page.platform === "facebook" ? "bg-blue-500/20" : "bg-pink-500/20")}>
                        {page.platform === "facebook" ? <Facebook className="h-5 w-5 text-blue-400" /> : <Instagram className="h-5 w-5 text-pink-400" />}
                      </div>
                      <div>
                        <p className="font-semibold text-foreground">{page.name}</p>
                        <p className="text-xs text-muted-foreground font-mono">{page.page_id}</p>
                      </div>
                    </div>
                    <div className="flex gap-1">
                      <button onClick={() => setEditId(page.id)} className="p-1.5 rounded hover:bg-accent text-muted-foreground"><Pencil className="h-3.5 w-3.5" /></button>
                      <button onClick={() => setDeleteId(page.id)} className="p-1.5 rounded hover:bg-accent text-muted-foreground hover:text-red-400"><Trash2 className="h-3.5 w-3.5" /></button>
                    </div>
                  </div>
                  <div className="flex flex-wrap gap-2">
                    {[
                      { label: "نشط", value: page.is_active },
                      { label: "رد تلقائي", value: page.auto_reply_enabled },
                      { label: "وضع الظل", value: page.shadow_mode },
                    ].map(({ label, value }) => (
                      <span key={label} className={cn("text-xs px-2 py-0.5 rounded-full border", value ? "bg-green-500/20 text-green-400 border-green-500/30" : "bg-gray-500/20 text-gray-400 border-gray-500/30")}>
                        {label}: {value ? "نعم" : "لا"}
                      </span>
                    ))}
                    {page.auto_reply_enabled && page.auto_reply_end_date && (
                      <span className={cn(
                        "text-xs px-2 py-0.5 rounded-full border flex items-center gap-1",
                        new Date(page.auto_reply_end_date) < new Date() 
                          ? "bg-red-500/20 text-red-400 border-red-500/30" 
                          : "bg-indigo-500/20 text-indigo-400 border-indigo-500/30"
                      )}>
                        الصلاحية: {new Date(page.auto_reply_end_date) < new Date() ? "منتهية" : "إلى " + new Date(page.auto_reply_end_date).toLocaleDateString("ar-EG")}
                      </span>
                    )}
                    <span className={cn("text-xs px-2 py-0.5 rounded-full border", getStatusColor(page.token_status ?? "valid"))}>
                      الرمز: {page.token_status ?? "—"}
                    </span>
                  </div>
                </div>
              )}
            </div>
          ))
        )}
        {!isLoading && (pages ?? []).length === 0 && (
          <div className="col-span-2 flex flex-col items-center justify-center h-48 gap-2 text-muted-foreground">
            <Globe className="h-8 w-8 opacity-40" />
            <p className="text-sm">لا توجد صفحات</p>
          </div>
        )}
      </div>
      <ConfirmDialog
        open={Boolean(deleteId)}
        title="حذف الصفحة"
        description="سيتم حذف الصفحة وإعداداتها من لوحة التحكم. لا يمكن التراجع عن هذا الإجراء."
        destructive
        confirmLabel="حذف"
        pending={deleteMutation.isPending}
        onCancel={() => setDeleteId(null)}
        onConfirm={() => deleteId && deleteMutation.mutate(deleteId)}
      />
    </div>
    </RequirePermission>
  );
}
