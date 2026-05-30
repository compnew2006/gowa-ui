"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api, type KnowledgeBaseEntry } from "@/lib/api";
import { usePageContext } from "@/lib/page-context";
import { cn } from "@/lib/utils";
import { BookOpen, Plus, Pencil, Trash2, Check, X } from "lucide-react";
import { toast } from "sonner";
import { ConfirmDialog } from "@/components/ui/confirm-dialog";
import { RequirePermission } from "@/components/auth/permission-guard";

const CATEGORIES = ["pricing", "shipping", "returns", "products", "support", "general"];

function EntryForm({ initial, onSave, onCancel, saving }: {
  initial?: Partial<KnowledgeBaseEntry>; onSave: (data: any) => void; onCancel: () => void; saving: boolean;
}) {
  const [form, setForm] = useState({
    category: initial?.category ?? "general",
    question: initial?.question ?? "",
    answer: initial?.answer ?? "",
    language: initial?.language ?? "ar",
    is_active: initial?.is_active ?? true,
    intent_tags: (initial?.intent_tags ?? []).join(", "),
  });

  return (
    <div className="rounded-xl border border-primary/30 bg-primary/5 p-4 space-y-3">
      <div className="grid grid-cols-2 gap-3">
        <div>
          <label className="text-xs text-muted-foreground mb-1 block">الفئة</label>
          <select className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground" value={form.category} onChange={(e) => setForm(f => ({ ...f, category: e.target.value }))}>
            {CATEGORIES.map(c => <option key={c}>{c}</option>)}
          </select>
        </div>
        <div>
          <label className="text-xs text-muted-foreground mb-1 block">اللغة</label>
          <select className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground" value={form.language} onChange={(e) => setForm(f => ({ ...f, language: e.target.value }))}>
            <option value="ar">العربية</option>
            <option value="en">الإنجليزية</option>
          </select>
        </div>
      </div>
      <div>
        <label className="text-xs text-muted-foreground mb-1 block">السؤال</label>
        <input className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground" value={form.question} onChange={(e) => setForm(f => ({ ...f, question: e.target.value }))} placeholder="السؤال أو الموضوع..." />
      </div>
      <div>
        <label className="text-xs text-muted-foreground mb-1 block">الإجابة</label>
        <textarea className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground resize-none" rows={3} value={form.answer} onChange={(e) => setForm(f => ({ ...f, answer: e.target.value }))} placeholder="الإجابة..." />
      </div>
      <div>
        <label className="text-xs text-muted-foreground mb-1 block">علامات النية (مفصولة بفاصلة)</label>
        <input className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground" value={form.intent_tags} onChange={(e) => setForm(f => ({ ...f, intent_tags: e.target.value }))} placeholder="price_inquiry, general..." />
      </div>
      <div className="flex gap-2 justify-end">
        <button onClick={onCancel} className="px-3 py-1.5 text-sm rounded-lg border border-border hover:bg-accent text-muted-foreground"><X className="h-4 w-4 inline ml-1" />إلغاء</button>
        <button
          onClick={() => onSave({ ...form, intent_tags: form.intent_tags.split(",").map(t => t.trim()).filter(Boolean) })}
          disabled={!form.question || !form.answer || saving}
          className="px-3 py-1.5 text-sm rounded-lg bg-primary text-primary-foreground disabled:opacity-50"
        >
          <Check className="h-4 w-4 inline ml-1" />حفظ
        </button>
      </div>
    </div>
  );
}

export default function KnowledgeBasePage() {
  const qc = useQueryClient();
  const { selectedPageId, selectedPage } = usePageContext();
  const [showCreate, setShowCreate] = useState(false);
  const [editId, setEditId] = useState<string | null>(null);
  const [filterCat, setFilterCat] = useState("");
  const [deleteId, setDeleteId] = useState<string | null>(null);

  const kbParams: Record<string, string> = {};
  if (filterCat) kbParams.category = filterCat;
  if (selectedPageId) kbParams.page_id = selectedPageId;

  const { data: entries, isLoading } = useQuery({
    queryKey: ["knowledge-base", selectedPageId, filterCat],
    queryFn: () => api.getKnowledgeBase(Object.keys(kbParams).length ? kbParams : undefined),
  });

  const createMutation = useMutation({
    mutationFn: (data: any) => api.createKBEntry({ ...data, page_id: selectedPageId }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["knowledge-base"] }); setShowCreate(false); toast.success("تمت الإضافة"); },
    onError: () => toast.error("فشل الحفظ"),
  });

  const updateMutation = useMutation({
    mutationFn: ({ id, data }: { id: string; data: any }) => api.updateKBEntry(id, data),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["knowledge-base"] }); setEditId(null); toast.success("تم التحديث"); },
    onError: () => toast.error("فشل التحديث"),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.deleteKBEntry(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["knowledge-base"] }); setDeleteId(null); toast.success("تم الحذف"); },
    onError: () => toast.error("فشل الحذف"),
  });

  const grouped = (entries ?? []).reduce((acc, e) => {
    (acc[e.category] = acc[e.category] ?? []).push(e);
    return acc;
  }, {} as Record<string, KnowledgeBaseEntry[]>);

  return (
    <RequirePermission permission="can_manage_settings">
    <div className="p-6 space-y-5">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">قاعدة المعرفة</h1>
          <p className="text-sm text-muted-foreground mt-1">
            {selectedPage ? `${selectedPage.name} — ` : ""}{entries?.length ?? 0} مدخلة
          </p>
        </div>
        <button onClick={() => setShowCreate(true)} className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground hover:bg-primary/90">
          <Plus className="h-4 w-4" />إضافة مدخلة
        </button>
      </div>

      <div className="flex gap-2 flex-wrap">
        <button onClick={() => setFilterCat("")} className={cn("text-xs px-3 py-1.5 rounded-full border transition-colors", !filterCat ? "bg-primary/20 border-primary/40 text-primary" : "border-border text-muted-foreground hover:bg-accent")}>الكل</button>
        {CATEGORIES.map(c => (
          <button key={c} onClick={() => setFilterCat(c)} className={cn("text-xs px-3 py-1.5 rounded-full border transition-colors", filterCat === c ? "bg-primary/20 border-primary/40 text-primary" : "border-border text-muted-foreground hover:bg-accent")}>{c}</button>
        ))}
      </div>

      {showCreate && <EntryForm onSave={(d) => createMutation.mutate(d)} onCancel={() => setShowCreate(false)} saving={createMutation.isPending} />}

      {isLoading ? (
        <div className="text-center h-48 flex items-center justify-center text-muted-foreground">جاري التحميل...</div>
      ) : (
        <div className="space-y-6">
          {Object.entries(grouped).map(([cat, items]) => (
            <div key={cat}>
              <h2 className="text-sm font-semibold text-muted-foreground uppercase tracking-wider mb-3">{cat}</h2>
              <div className="space-y-3">
                {items.map(entry => (
                  <div key={entry.id}>
                    {editId === entry.id ? (
                      <EntryForm
                        initial={entry}
                        onSave={(d) => updateMutation.mutate({ id: entry.id, data: d })}
                        onCancel={() => setEditId(null)}
                        saving={updateMutation.isPending}
                      />
                    ) : (
                      <div className={cn("rounded-xl border bg-card p-4", !entry.is_active && "opacity-50")}>
                        <div className="flex items-start justify-between gap-3">
                          <div className="flex-1">
                            <p className="font-medium text-sm text-foreground mb-1">{entry.question}</p>
                            <p className="text-sm text-muted-foreground">{entry.answer}</p>
                            <div className="flex items-center gap-2 mt-2">
                              <span className="text-xs text-muted-foreground">{entry.language.toUpperCase()}</span>
                              <span className="text-xs text-muted-foreground">·</span>
                              <span className="text-xs text-muted-foreground">استخدم {entry.usage_count}x</span>
                              {entry.intent_tags.map(t => <span key={t} className="text-xs bg-blue-500/10 text-blue-400 rounded px-1.5">{t}</span>)}
                            </div>
                          </div>
                          <div className="flex gap-1 shrink-0">
                            <button onClick={() => setEditId(entry.id)} className="p-1.5 rounded hover:bg-accent text-muted-foreground"><Pencil className="h-3.5 w-3.5" /></button>
                            <button onClick={() => setDeleteId(entry.id)} className="p-1.5 rounded hover:bg-accent text-muted-foreground hover:text-red-400"><Trash2 className="h-3.5 w-3.5" /></button>
                          </div>
                        </div>
                      </div>
                    )}
                  </div>
                ))}
              </div>
            </div>
          ))}
          {Object.keys(grouped).length === 0 && (
            <div className="flex flex-col items-center justify-center h-48 gap-2 text-muted-foreground">
              <BookOpen className="h-8 w-8 opacity-40" />
              <p className="text-sm">لا توجد مدخلات</p>
            </div>
          )}
        </div>
      )}
      <ConfirmDialog
        open={Boolean(deleteId)}
        title="حذف مدخلة قاعدة المعرفة"
        description="سيتم حذف هذه الإجابة من قاعدة المعرفة ولن تستخدمها الأتمتة لاحقاً."
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
