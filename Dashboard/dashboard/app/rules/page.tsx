"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { usePageContext } from "@/lib/page-context";
import { cn } from "@/lib/utils";
import { Zap, Plus, Trash2, ChevronRight, X } from "lucide-react";
import { toast } from "sonner";
import { RequirePermission } from "@/components/auth/permission-guard";

const ACTION_LABELS: Record<string, { label: string; color: string }> = {
  escalate: { label: "تصعيد", color: "text-red-400 bg-red-500/10" },
  tag: { label: "وسم", color: "text-blue-400 bg-blue-500/10" },
  assign: { label: "تعيين", color: "text-purple-400 bg-purple-500/10" },
  notify: { label: "إشعار", color: "text-yellow-400 bg-yellow-500/10" },
  skip: { label: "تخطي", color: "text-gray-400 bg-gray-500/10" },
  custom_reply: { label: "رد مخصص", color: "text-green-400 bg-green-500/10" },
};

const CONDITION_FIELDS = [
  { value: "intent", label: "النية" },
  { value: "sentiment", label: "المشاعر" },
  { value: "language", label: "اللغة" },
  { value: "confidence", label: "الثقة" },
  { value: "platform", label: "المنصة" },
  { value: "priority", label: "الأولوية" },
  { value: "customer_lead_score", label: "نقاط العميل" },
];

const OPERATORS = [
  { value: "eq", label: "يساوي" },
  { value: "neq", label: "لا يساوي" },
  { value: "contains", label: "يحتوي على" },
  { value: "gt", label: "أكبر من" },
  { value: "lt", label: "أصغر من" },
  { value: "in", label: "ضمن" },
];

const EMPTY_RULE = {
  name: "", description: "", conditions: [{ field: "intent", op: "eq", value: "" }],
  condition_logic: "AND", action: "escalate", action_config: {}, priority: 10, is_active: true,
};

function RuleFormModal({ initial, onClose }: { initial?: any; onClose: () => void }) {
  const qc = useQueryClient();
  const [form, setForm] = useState(initial ?? EMPTY_RULE);

  const mutation = useMutation({
    mutationFn: () => initial ? api.updateRule(initial.id, form) : api.createRule(form),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["rules"] }); toast.success(initial ? "تم التحديث" : "تم الإنشاء"); onClose(); },
    onError: () => toast.error("فشلت العملية"),
  });

  const addCondition = () => setForm((f: any) => ({ ...f, conditions: [...f.conditions, { field: "intent", op: "eq", value: "" }] }));
  const removeCondition = (i: number) => setForm((f: any) => ({ ...f, conditions: f.conditions.filter((_: any, idx: number) => idx !== i) }));
  const updateCondition = (i: number, key: string, value: string) => setForm((f: any) => ({
    ...f,
    conditions: f.conditions.map((c: any, idx: number) => idx === i ? { ...c, [key]: value } : c),
  }));

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 overflow-y-auto py-8">
      <div className="w-full max-w-xl rounded-xl border border-border bg-card p-6 space-y-4 shadow-xl m-auto">
        <div className="flex items-center justify-between">
          <h2 className="font-semibold text-foreground">{initial ? "تعديل القاعدة" : "قاعدة جديدة"}</h2>
          <button onClick={onClose}><X className="h-4 w-4 text-muted-foreground" /></button>
        </div>

        <div className="space-y-1">
          <label className="text-xs text-muted-foreground">اسم القاعدة</label>
          <input className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm focus:outline-none focus:ring-2 focus:ring-primary/50 text-foreground"
            value={form.name} onChange={e => setForm((f: any) => ({ ...f, name: e.target.value }))} placeholder="مثال: تصعيد الشكاوى الغاضبة" />
        </div>

        <div className="space-y-2">
          <div className="flex items-center justify-between">
            <label className="text-xs text-muted-foreground">الشروط</label>
            <div className="flex items-center gap-2">
              <span className="text-xs text-muted-foreground">المنطق:</span>
              {["AND", "OR"].map(l => (
                <button key={l} onClick={() => setForm((f: any) => ({ ...f, condition_logic: l }))}
                  className={cn("text-xs px-2 py-0.5 rounded border transition-colors", form.condition_logic === l ? "border-primary bg-primary/15 text-primary" : "border-border text-muted-foreground")}>
                  {l}
                </button>
              ))}
            </div>
          </div>
          {form.conditions.map((c: any, i: number) => (
            <div key={i} className="flex gap-2 items-center">
              <select className="flex-1 rounded-lg border border-border bg-background px-2 py-1.5 text-xs text-foreground focus:outline-none"
                value={c.field} onChange={e => updateCondition(i, "field", e.target.value)}>
                {CONDITION_FIELDS.map(f => <option key={f.value} value={f.value}>{f.label}</option>)}
              </select>
              <select className="flex-1 rounded-lg border border-border bg-background px-2 py-1.5 text-xs text-foreground focus:outline-none"
                value={c.op} onChange={e => updateCondition(i, "op", e.target.value)}>
                {OPERATORS.map(o => <option key={o.value} value={o.value}>{o.label}</option>)}
              </select>
              <input className="flex-1 rounded-lg border border-border bg-background px-2 py-1.5 text-xs text-foreground focus:outline-none"
                value={c.value} onChange={e => updateCondition(i, "value", e.target.value)} placeholder="القيمة" />
              <button onClick={() => removeCondition(i)} className="p-1 text-muted-foreground hover:text-red-400"><X className="h-3.5 w-3.5" /></button>
            </div>
          ))}
          <button onClick={addCondition} className="text-xs text-primary hover:underline">+ إضافة شرط</button>
        </div>

        <div className="grid grid-cols-2 gap-3">
          <div className="space-y-1">
            <label className="text-xs text-muted-foreground">الإجراء</label>
            <select className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none"
              value={form.action} onChange={e => setForm((f: any) => ({ ...f, action: e.target.value }))}>
              {Object.entries(ACTION_LABELS).map(([k, v]) => <option key={k} value={k}>{v.label}</option>)}
            </select>
          </div>
          <div className="space-y-1">
            <label className="text-xs text-muted-foreground">الأولوية (1=عليا)</label>
            <input type="number" min={1} max={100} className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none"
              value={form.priority} onChange={e => setForm((f: any) => ({ ...f, priority: parseInt(e.target.value) || 10 }))} />
          </div>
        </div>

        {form.action === "custom_reply" && (
          <div className="space-y-1">
            <label className="text-xs text-muted-foreground">نص الرد المخصص</label>
            <textarea className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none h-20 resize-none"
              value={form.action_config?.template ?? ""} onChange={e => setForm((f: any) => ({ ...f, action_config: { ...f.action_config, template: e.target.value } }))} />
          </div>
        )}
        {form.action === "tag" && (
          <div className="space-y-1">
            <label className="text-xs text-muted-foreground">الوسم</label>
            <input className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none"
              value={form.action_config?.tag ?? ""} onChange={e => setForm((f: any) => ({ ...f, action_config: { ...f.action_config, tag: e.target.value } }))} placeholder="hot-lead" />
          </div>
        )}

        <div className="flex gap-2 pt-2">
          <button onClick={() => mutation.mutate()} disabled={!form.name || mutation.isPending}
            className="flex-1 rounded-lg bg-primary py-2 text-sm font-medium text-primary-foreground disabled:opacity-50">
            {mutation.isPending ? "جاري..." : initial ? "حفظ التعديلات" : "إنشاء القاعدة"}
          </button>
          <button onClick={onClose} className="rounded-lg border border-border px-4 py-2 text-sm text-muted-foreground hover:bg-accent">إلغاء</button>
        </div>
      </div>
    </div>
  );
}

export default function RulesPage() {
  const qc = useQueryClient();
  const { selectedPageId } = usePageContext();
  const [showForm, setShowForm] = useState(false);
  const [editRule, setEditRule] = useState<any>(null);

  const { data: rules = [], isLoading } = useQuery({
    queryKey: ["rules", selectedPageId],
    queryFn: () => api.getRules(selectedPageId),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.deleteRule(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["rules"] }); toast.success("تم الحذف"); },
  });

  const toggleMutation = useMutation({
    mutationFn: ({ id, is_active }: { id: string; is_active: boolean }) => api.updateRule(id, { is_active }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["rules"] }),
  });

  return (
    <RequirePermission permission="can_manage_settings">
    <div className="p-6 space-y-5">
      {(showForm || editRule) && (
        <RuleFormModal initial={editRule} onClose={() => { setShowForm(false); setEditRule(null); }} />
      )}

      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">قواعد الأتمتة</h1>
          <p className="text-sm text-muted-foreground mt-1">إذا تحقق الشرط ← نفّذ الإجراء</p>
        </div>
        <button onClick={() => setShowForm(true)}
          className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground">
          <Plus className="h-4 w-4" />قاعدة جديدة
        </button>
      </div>

      <div className="rounded-xl border border-border bg-card overflow-hidden">
        {isLoading ? (
          <div className="flex h-48 items-center justify-center text-muted-foreground">جاري التحميل...</div>
        ) : rules.length === 0 ? (
          <div className="flex flex-col items-center justify-center h-48 gap-2 text-muted-foreground">
            <Zap className="h-8 w-8 opacity-40" />
            <p className="text-sm">لا توجد قواعد بعد</p>
          </div>
        ) : (
          <div className="divide-y divide-border">
            {rules.map((rule: any) => {
              const actionConf = ACTION_LABELS[rule.action] ?? { label: rule.action, color: "text-gray-400 bg-gray-500/10" };
              return (
                <div key={rule.id} className={cn("p-4 transition-colors", rule.is_active ? "hover:bg-accent/20" : "opacity-50 hover:bg-accent/10")}>
                  <div className="flex items-start gap-4">
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2 mb-1.5">
                        <span className="font-medium text-sm text-foreground">{rule.name}</span>
                        <span className={cn("text-xs px-2 py-0.5 rounded-full font-medium", actionConf.color)}>
                          {actionConf.label}
                        </span>
                        <span className="text-xs text-muted-foreground bg-accent/50 px-1.5 py-0.5 rounded">أولوية {rule.priority}</span>
                      </div>
                      <div className="flex flex-wrap gap-1 mb-1">
                        {(rule.conditions ?? []).map((c: any, i: number) => (
                          <span key={i} className="text-xs bg-primary/10 text-primary border border-primary/20 rounded px-1.5 py-0.5">
                            {c.field} {c.op} "{c.value}"
                          </span>
                        ))}
                        {rule.conditions?.length > 1 && (
                          <span className="text-xs text-muted-foreground self-center">{rule.condition_logic}</span>
                        )}
                      </div>
                      <p className="text-xs text-muted-foreground">
                        {rule.trigger_count} مرة · {rule.last_triggered_at ? `آخر تشغيل: ${new Date(rule.last_triggered_at).toLocaleDateString("ar-SA")}` : "لم يُشغَّل بعد"}
                      </p>
                    </div>
                    <div className="flex gap-1 shrink-0">
                      <button onClick={() => setEditRule(rule)} className="p-1.5 rounded-lg hover:bg-accent text-muted-foreground hover:text-foreground"><Zap className="h-4 w-4" /></button>
                      <button
                        onClick={() => toggleMutation.mutate({ id: rule.id, is_active: !rule.is_active })}
                        className={cn("p-1.5 rounded-lg transition-colors", rule.is_active ? "text-green-400 hover:bg-green-500/10" : "text-muted-foreground hover:bg-accent")}>
                        <ChevronRight className="h-4 w-4" />
                      </button>
                      <button onClick={() => deleteMutation.mutate(rule.id)} className="p-1.5 rounded-lg text-muted-foreground hover:bg-red-500/10 hover:text-red-400"><Trash2 className="h-4 w-4" /></button>
                    </div>
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </div>
    </div>
    </RequirePermission>
  );
}
