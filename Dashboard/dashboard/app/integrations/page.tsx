"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { cn } from "@/lib/utils";
import { Plug, Plus, Trash2, CheckCircle, XCircle, X, FlaskConical, AlertTriangle } from "lucide-react";
import { toast } from "sonner";
import { RequirePermission } from "@/components/auth/permission-guard";

const INTEGRATION_TYPES = [
  { value: "slack", label: "Slack", icon: "💬", desc: "إرسال إشعارات للقنوات", configFields: [{ key: "webhook_url", label: "Webhook URL", placeholder: "https://hooks.slack.com/...", type: "text" }] },
  { value: "zapier", label: "Zapier", icon: "⚡", desc: "أتمتة سير عمل مخصصة", configFields: [{ key: "webhook_url", label: "Webhook URL", placeholder: "https://hooks.zapier.com/...", type: "text" }] },
  { value: "whatsapp", label: "WhatsApp Business", icon: "📱", desc: "إشعارات واتساب للفريق", configFields: [{ key: "api_key", label: "API Key", placeholder: "whatsapp_key_...", type: "password" }, { key: "phone_number_id", label: "Phone Number ID", placeholder: "123456789", type: "text" }] },
  { value: "teams", label: "Microsoft Teams", icon: "🏢", desc: "إشعارات Teams للفريق", configFields: [{ key: "webhook_url", label: "Incoming Webhook URL", placeholder: "https://outlook.office.com/...", type: "text" }] },
];

const EVENT_LABELS: Record<string, string> = {
  escalation: "تصعيد جديد",
  reply: "رد تلقائي",
  error: "خطأ في النظام",
  shadow_approved: "موافقة وضع الظل",
  token_expiring: "رمز ينتهي قريباً",
  churn_alert: "تنبيه مغادرة عميل",
};

function IntegrationFormModal({ onClose }: { onClose: () => void }) {
  const qc = useQueryClient();
  const [type, setType] = useState("slack");
  const [form, setForm] = useState<Record<string, string>>({});
  const [name, setName] = useState("");
  const [events, setEvents] = useState(["escalation"]);

  const typeConf = INTEGRATION_TYPES.find(t => t.value === type)!;
  const mutation = useMutation({
    mutationFn: () => api.createIntegration({
      type, name: name || typeConf.label,
      config: form, trigger_events: events, is_active: true,
    }),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["integrations"] }); toast.success("تم الإنشاء"); onClose(); },
    onError: () => toast.error("فشل الإنشاء"),
  });

  const toggleEvent = (ev: string) => setEvents(es => es.includes(ev) ? es.filter(e => e !== ev) : [...es, ev]);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 overflow-y-auto py-8">
      <div className="w-full max-w-md rounded-xl border border-border bg-card p-6 space-y-4 shadow-xl m-auto">
        <div className="flex items-center justify-between">
          <h2 className="font-semibold text-foreground">تكامل جديد</h2>
          <button onClick={onClose}><X className="h-4 w-4 text-muted-foreground" /></button>
        </div>

        <div className="grid grid-cols-2 gap-2">
          {INTEGRATION_TYPES.map(t => (
            <button key={t.value} onClick={() => setType(t.value)}
              className={cn("rounded-lg border p-3 text-left transition-all", type === t.value ? "border-primary bg-primary/10" : "border-border hover:bg-accent")}>
              <span className="text-lg">{t.icon}</span>
              <p className="text-sm font-medium text-foreground mt-1">{t.label}</p>
              <p className="text-xs text-muted-foreground">{t.desc}</p>
            </button>
          ))}
        </div>

        <div className="space-y-1">
          <label className="text-xs text-muted-foreground">الاسم (اختياري)</label>
          <input className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50"
            placeholder={typeConf.label} value={name} onChange={e => setName(e.target.value)} />
        </div>

        {typeConf.configFields.map(f => (
          <div key={f.key} className="space-y-1">
            <label className="text-xs text-muted-foreground">{f.label}</label>
            <input type={f.type} className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50"
              placeholder={f.placeholder} value={form[f.key] ?? ""} onChange={e => setForm(prev => ({ ...prev, [f.key]: e.target.value }))} />
          </div>
        ))}

        <div className="space-y-2">
          <label className="text-xs text-muted-foreground">الأحداث المُفعَّلة</label>
          <div className="flex flex-wrap gap-2">
            {Object.entries(EVENT_LABELS).map(([ev, label]) => (
              <button key={ev} onClick={() => toggleEvent(ev)}
                className={cn("text-xs px-2.5 py-1 rounded-full border transition-colors", events.includes(ev) ? "border-primary bg-primary/15 text-primary" : "border-border text-muted-foreground hover:bg-accent")}>
                {label}
              </button>
            ))}
          </div>
        </div>

        <div className="flex gap-2 pt-2">
          <button onClick={() => mutation.mutate()} disabled={mutation.isPending}
            className="flex-1 rounded-lg bg-primary py-2 text-sm font-medium text-primary-foreground disabled:opacity-50">
            {mutation.isPending ? "جاري..." : "إنشاء"}
          </button>
          <button onClick={onClose} className="rounded-lg border border-border px-4 py-2 text-sm text-muted-foreground hover:bg-accent">إلغاء</button>
        </div>
      </div>
    </div>
  );
}

export default function IntegrationsPage() {
  const qc = useQueryClient();
  const [showAdd, setShowAdd] = useState(false);

  const { data: integrations = [], isLoading } = useQuery({
    queryKey: ["integrations"],
    queryFn: () => api.getIntegrations(),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => api.deleteIntegration(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["integrations"] }); toast.success("تم الحذف"); },
  });

  const toggleMutation = useMutation({
    mutationFn: ({ id, is_active }: { id: string; is_active: boolean }) => api.updateIntegration(id, { is_active }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["integrations"] }),
  });

  const testMutation = useMutation({
    mutationFn: (id: string) => api.testIntegration(id),
    onSuccess: (data) => {
      if (data.success) toast.success("✓ التكامل يعمل بشكل صحيح");
      else toast.error(`فشل الاختبار: ${data.error ?? "خطأ غير معروف"}`);
    },
    onError: () => toast.error("فشل الاختبار"),
  });

  const typeConf = (type: string) => INTEGRATION_TYPES.find(t => t.value === type) ?? { icon: "🔗", label: type, desc: "" };

  return (
    <RequirePermission permission="can_manage_settings">
    <div className="p-6 space-y-5">
      {showAdd && <IntegrationFormModal onClose={() => setShowAdd(false)} />}

      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">التكاملات</h1>
          <p className="text-sm text-muted-foreground mt-1">ربط النظام بـ Slack وZapier وWhatsApp وTeams</p>
        </div>
        <button onClick={() => setShowAdd(true)}
          className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground">
          <Plus className="h-4 w-4" />إضافة تكامل
        </button>
      </div>

      {isLoading ? (
        <div className="flex h-48 items-center justify-center text-muted-foreground">جاري التحميل...</div>
      ) : integrations.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-xl border border-border bg-card h-64 gap-3 text-muted-foreground">
          <Plug className="h-10 w-10 opacity-30" />
          <div className="text-center">
            <p className="font-medium">لا توجد تكاملات بعد</p>
            <p className="text-sm mt-1">أضف Slack أو Zapier لإشعارات آنية</p>
          </div>
        </div>
      ) : (
        <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
          {integrations.map((int: any) => {
            const conf = typeConf(int.type);
            return (
              <div key={int.id} className={cn("rounded-xl border bg-card p-5 space-y-3", int.is_active ? "border-border" : "border-border/50 opacity-60")}>
                <div className="flex items-start justify-between">
                  <div className="flex items-center gap-3">
                    <span className="text-2xl">{conf.icon}</span>
                    <div>
                      <p className="font-medium text-foreground">{int.name}</p>
                      <p className="text-xs text-muted-foreground">{conf.label}</p>
                    </div>
                  </div>
                  <div className="flex items-center gap-1">
                    <div className={cn("h-2 w-2 rounded-full", int.is_active ? "bg-green-400" : "bg-gray-400")} />
                  </div>
                </div>

                <div className="flex flex-wrap gap-1">
                  {(int.trigger_events ?? []).map((ev: string) => (
                    <span key={ev} className="text-xs bg-primary/10 text-primary border border-primary/20 rounded-full px-2 py-0.5">
                      {EVENT_LABELS[ev] ?? ev}
                    </span>
                  ))}
                </div>

                <div className="text-xs text-muted-foreground space-y-0.5">
                  <p>تشغيلات: {int.trigger_count}</p>
                  {int.last_triggered_at && <p>آخر تشغيل: {new Date(int.last_triggered_at).toLocaleDateString("ar-SA")}</p>}
                  {int.last_error && <p className="text-red-400 flex items-center gap-1"><AlertTriangle className="h-3 w-3" />{int.last_error}</p>}
                </div>

                <div className="flex gap-2 pt-1 border-t border-border">
                  <button onClick={() => testMutation.mutate(int.id)} disabled={testMutation.isPending}
                    className="flex items-center gap-1.5 text-xs rounded-lg border border-border px-3 py-1.5 text-muted-foreground hover:bg-accent transition-colors disabled:opacity-50">
                    <FlaskConical className="h-3.5 w-3.5" />اختبار
                  </button>
                  <button onClick={() => toggleMutation.mutate({ id: int.id, is_active: !int.is_active })}
                    className={cn("flex items-center gap-1.5 text-xs rounded-lg border px-3 py-1.5 transition-colors",
                      int.is_active ? "border-red-500/30 text-red-400 hover:bg-red-500/10" : "border-green-500/30 text-green-400 hover:bg-green-500/10")}>
                    {int.is_active ? <><XCircle className="h-3.5 w-3.5" />تعطيل</> : <><CheckCircle className="h-3.5 w-3.5" />تفعيل</>}
                  </button>
                  <button onClick={() => deleteMutation.mutate(int.id)}
                    className="ml-auto flex items-center gap-1.5 text-xs rounded-lg border border-border px-2 py-1.5 text-muted-foreground hover:bg-red-500/10 hover:text-red-400 transition-colors">
                    <Trash2 className="h-3.5 w-3.5" />
                  </button>
                </div>
              </div>
            );
          })}
        </div>
      )}
    </div>
    </RequirePermission>
  );
}
