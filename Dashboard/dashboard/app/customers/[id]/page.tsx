"use client";

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import type { Conversation } from "@/lib/api";
import { cn, formatDateTime, formatRelativeTime } from "@/lib/utils";
import { ArrowLeft, Plus, Save, MessageSquare, MessageCircle, Clock, CheckCircle2, AlertCircle, Bot, RefreshCw } from "lucide-react";
import Link from "next/link";
import { use, useState } from "react";
import { toast } from "sonner";

type Tab = "profile" | "conversations";

const STATUS_COLOR: Record<string, string> = {
  replied: "bg-green-500/20 text-green-400",
  pending: "bg-yellow-500/20 text-yellow-400",
  resolved: "bg-blue-500/20 text-blue-400",
  escalated: "bg-red-500/20 text-red-400",
};

const SENTIMENT_ICON: Record<string, string> = {
  positive: "😊",
  negative: "😞",
  neutral: "😐",
  angry: "😠",
};

export default function CustomerDetailPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  const qc = useQueryClient();
  const [note, setNote] = useState("");
  const [editField, setEditField] = useState<string | null>(null);
  const [editValues, setEditValues] = useState<Record<string, any>>({});
  const [tab, setTab] = useState<Tab>("profile");

  const { data: customer, isLoading } = useQuery({
    queryKey: ["customer", id],
    queryFn: () => api.getCustomer(id),
  });

  const { data: conversations, isLoading: convLoading } = useQuery({
    queryKey: ["customer-conversations", id],
    queryFn: () => api.getCustomerConversations(id),
    enabled: tab === "conversations",
  });

  const updateMutation = useMutation({
    mutationFn: (data: any) => api.updateCustomer(id, data),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["customer", id] }); setEditField(null); toast.success("تم التحديث"); },
    onError: () => toast.error("فشل التحديث"),
  });

  const noteMutation = useMutation({
    mutationFn: (content: string) => api.addNote(id, content),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["customer", id] }); setNote(""); toast.success("تمت إضافة الملاحظة"); },
    onError: () => toast.error("فشل إضافة الملاحظة"),
  });

  const refreshMutation = useMutation({
    mutationFn: () => api.refreshCustomerPredictions(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["customer", id] }); toast.success("تم تحديث توقعات العميل"); },
    onError: () => toast.error("فشل تحديث التوقعات"),
  });

  if (isLoading) return <div className="flex items-center justify-center h-64 text-muted-foreground">جاري التحميل...</div>;
  if (!customer) return <div className="p-6 text-red-400">لم يُعثر على العميل</div>;

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center gap-3">
        <Link href="/customers" className="p-2 rounded-lg hover:bg-accent text-muted-foreground">
          <ArrowLeft className="h-4 w-4" />
        </Link>
        <div>
          <h1 className="text-2xl font-bold text-foreground">{customer.full_name ?? customer.username ?? "عميل"}</h1>
          <p className="text-sm text-muted-foreground">تفاصيل العميل</p>
        </div>
        <button
          onClick={() => refreshMutation.mutate()}
          disabled={refreshMutation.isPending}
          className="mr-auto flex items-center gap-2 rounded-lg border border-border px-3 py-2 text-sm text-muted-foreground hover:bg-accent hover:text-foreground disabled:opacity-50"
        >
          <RefreshCw className={cn("h-4 w-4", refreshMutation.isPending && "animate-spin")} />
          تحديث التوقعات
        </button>
      </div>

      {/* Tabs */}
      <div className="flex gap-1 border-b border-border">
        {([
          { id: "profile", label: "الملف الشخصي", icon: MessageCircle },
          { id: "conversations", label: "سجل المحادثات", icon: MessageSquare },
        ] as const).map(({ id, label, icon: Icon }) => (
          <button
            key={id}
            onClick={() => setTab(id)}
            className={cn(
              "flex items-center gap-2 px-4 py-2.5 text-sm font-medium border-b-2 -mb-px transition-colors",
              tab === id
                ? "border-primary text-primary"
                : "border-transparent text-muted-foreground hover:text-foreground"
            )}
          >
            <Icon className="h-4 w-4" />
            {label}
          </button>
        ))}
      </div>

      {tab === "profile" && (
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
          {/* Profile */}
          <div className="rounded-xl border border-border bg-card p-5 space-y-4">
            <div className="flex items-center gap-4">
              <div className="h-14 w-14 rounded-full bg-primary/20 flex items-center justify-center text-2xl font-bold text-primary">
                {(customer.full_name ?? customer.username ?? "?")[0].toUpperCase()}
              </div>
              <div>
                <p className="font-semibold text-foreground">{customer.full_name ?? "—"}</p>
                {customer.username && <p className="text-sm text-muted-foreground">@{customer.username}</p>}
              </div>
            </div>

            <div className="space-y-3 text-sm">
              {[
                { label: "نية الشراء", key: "purchase_intent", options: ["High", "Medium", "Low"] },
                { label: "حالة التحويل", key: "conversion_status", options: ["prospect", "qualified", "converted", "churned"] },
                { label: "المسؤول", key: "assigned_admin" },
                { label: "النقاط", key: "lead_score", type: "number" },
              ].map(({ label, key, options, type }) => (
                <div key={key} className="flex items-center justify-between">
                  <span className="text-muted-foreground">{label}:</span>
                  {editField === key ? (
                    <div className="flex gap-1">
                      {options ? (
                        <select
                          className="rounded bg-background border border-border px-2 py-1 text-xs text-foreground"
                          value={editValues[key] ?? (customer as any)[key]}
                          onChange={(e) => setEditValues(v => ({ ...v, [key]: e.target.value }))}
                        >
                          {options.map(o => <option key={o}>{o}</option>)}
                        </select>
                      ) : (
                        <input
                          type={type ?? "text"}
                          className="rounded bg-background border border-border px-2 py-1 text-xs text-foreground w-24"
                          value={editValues[key] ?? (customer as any)[key] ?? ""}
                          onChange={(e) => setEditValues(v => ({ ...v, [key]: type === "number" ? Number(e.target.value) : e.target.value }))}
                        />
                      )}
                      <button onClick={() => updateMutation.mutate({ [key]: editValues[key] ?? (customer as any)[key] })} className="p-1 rounded bg-primary text-white">
                        <Save className="h-3 w-3" />
                      </button>
                    </div>
                  ) : (
                    <span
                      className="font-medium text-foreground cursor-pointer hover:text-primary"
                      onClick={() => { setEditField(key); setEditValues({}); }}
                    >
                      {String((customer as any)[key] ?? "—")}
                    </span>
                  )}
                </div>
              ))}
            </div>

            <div className="pt-2 border-t border-border space-y-1.5 text-sm">
              <div className="flex justify-between"><span className="text-muted-foreground">التفاعلات:</span><span className="text-foreground">{customer.interaction_count}</span></div>
              <div className="flex justify-between"><span className="text-muted-foreground">أول تواصل:</span><span className="text-foreground">{formatDateTime(customer.first_contact_date)}</span></div>
              <div className="flex justify-between"><span className="text-muted-foreground">آخر تفاعل:</span><span className="text-foreground">{formatDateTime(customer.last_interaction)}</span></div>
              
              {customer.platform && (
                <div className="flex justify-between items-center pt-1.5 border-t border-border/40">
                  <span className="text-muted-foreground">القناة:</span>
                  <span className={cn(
                    "text-[10px] px-2 py-0.5 rounded border font-medium",
                    customer.platform === "facebook" ? "bg-blue-500/10 text-blue-400 border-blue-500/20" :
                    customer.platform === "instagram" ? "bg-pink-500/10 text-pink-400 border-pink-500/20" :
                    customer.platform === "whatsapp" ? "bg-green-500/10 text-green-400 border-green-500/20" :
                    "bg-gray-500/10 text-gray-400 border-gray-500/20"
                  )}>
                    {customer.platform === "facebook" ? "🌐 فيسبوك" :
                     customer.platform === "instagram" ? "📸 إنستغرام" :
                     customer.platform === "whatsapp" ? "💬 واتساب" : customer.platform}
                  </span>
                </div>
              )}
              {customer.page_name && (
                <div className="flex justify-between"><span className="text-muted-foreground">الصفحة:</span><span className="text-foreground font-medium">{customer.page_name}</span></div>
              )}
              {customer.facebook_id && (
                <div className="flex justify-between items-center"><span className="text-muted-foreground">مُعرف Facebook:</span><span className="text-xs font-mono bg-accent px-1.5 py-0.5 rounded select-all text-foreground/80">{customer.facebook_id}</span></div>
              )}
              {customer.whatsapp_id && (
                <div className="flex justify-between items-center"><span className="text-muted-foreground">رقم WhatsApp:</span><span className="text-xs font-mono bg-accent px-1.5 py-0.5 rounded select-all text-foreground/80">{customer.whatsapp_id}</span></div>
              )}
              {customer.instagram_id && (
                <div className="flex justify-between items-center"><span className="text-muted-foreground">مُعرف Instagram:</span><span className="text-xs font-mono bg-accent px-1.5 py-0.5 rounded select-all text-foreground/80">{customer.instagram_id}</span></div>
              )}
            </div>

            {customer.tags.length > 0 && (
              <div className="flex flex-wrap gap-1">
                {customer.tags.map(tag => (
                  <span key={tag} className="text-xs bg-accent text-foreground rounded px-2 py-0.5">{tag}</span>
                ))}
              </div>
            )}
          </div>

          {/* Notes */}
          <div className="lg:col-span-2 rounded-xl border border-border bg-card p-5 space-y-4">
            <h2 className="font-semibold text-foreground">الملاحظات</h2>
            <div className="space-y-3 max-h-80 overflow-y-auto">
              {(customer.notes ?? []).map((note: any, i: number) => (
                <div key={i} className="rounded-lg bg-accent/30 p-3 text-sm">
                  <p className="text-foreground/80">{note.content}</p>
                  <p className="text-xs text-muted-foreground mt-1">{note.author} · {formatRelativeTime(note.createdAt)}</p>
                </div>
              ))}
              {(customer.notes ?? []).length === 0 && <p className="text-sm text-muted-foreground">لا توجد ملاحظات</p>}
            </div>
            <div className="flex gap-2">
              <input
                className="flex-1 rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50"
                placeholder="إضافة ملاحظة..."
                value={note}
                onChange={(e) => setNote(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && note && noteMutation.mutate(note)}
              />
              <button
                onClick={() => noteMutation.mutate(note)}
                disabled={!note || noteMutation.isPending}
                className="rounded-lg bg-primary px-3 py-2 text-sm text-primary-foreground disabled:opacity-50"
              >
                <Plus className="h-4 w-4" />
              </button>
            </div>
          </div>
        </div>
      )}

      {tab === "conversations" && (
        <div className="space-y-4">
          {convLoading ? (
            <div className="flex items-center justify-center h-40 text-muted-foreground">جاري تحميل المحادثات...</div>
          ) : (conversations as Conversation[] | undefined)?.length === 0 ? (
            <div className="flex flex-col items-center justify-center h-48 gap-3 text-muted-foreground rounded-xl border border-border bg-card">
              <MessageSquare className="h-10 w-10 opacity-30" />
              <p className="text-sm">لا توجد محادثات مسجّلة لهذا العميل</p>
            </div>
          ) : (
            <div className="space-y-3">
              {(conversations as Conversation[] | undefined)?.map((conv) => (
                <div key={conv.id} className="rounded-xl border border-border bg-card p-5 space-y-4">
                  {/* Header */}
                  <div className="flex items-start justify-between gap-4">
                    <div className="flex-1 space-y-1">
                      <div className="flex items-center gap-2 flex-wrap">
                        <span className={cn("text-xs px-2 py-0.5 rounded-full font-medium", STATUS_COLOR[conv.status] ?? "bg-gray-500/20 text-gray-400")}>
                          {conv.status}
                        </span>
                        {conv.intent && (
                          <span className="text-xs bg-purple-500/15 text-purple-400 px-2 py-0.5 rounded-full">{conv.intent}</span>
                        )}
                        {conv.sentiment && (
                          <span className="text-xs bg-accent px-2 py-0.5 rounded-full text-muted-foreground">
                            {SENTIMENT_ICON[conv.sentiment] ?? ""} {conv.sentiment}
                          </span>
                        )}
                        {conv.confidence_score != null && (
                          <span className="text-xs text-muted-foreground font-mono">
                            ثقة: {(conv.confidence_score * 100).toFixed(0)}%
                          </span>
                        )}
                      </div>
                      <p className="text-xs text-muted-foreground flex items-center gap-1">
                        <Clock className="h-3 w-3" />
                        {formatDateTime(conv.created_at)} · {conv.page_name}
                      </p>
                    </div>
                  </div>

                  {/* Original comment */}
                  <div className="space-y-1">
                    <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide">التعليق الأصلي</p>
                    <div className="rounded-lg bg-accent/40 px-4 py-3 text-sm text-foreground">
                      {conv.original_comment}
                    </div>
                  </div>

                  {/* AI public reply */}
                  {conv.ai_reply && (
                    <div className="space-y-1">
                      <p className="text-xs font-medium text-blue-400 uppercase tracking-wide flex items-center gap-1">
                        <MessageCircle className="h-3 w-3" />
                        الرد العلني (على التعليق)
                      </p>
                      <div className="rounded-lg border border-blue-500/20 bg-blue-500/5 px-4 py-3 text-sm text-blue-200">
                        {conv.ai_reply}
                      </div>
                    </div>
                  )}

                  {/* Admin reply (DM) */}
                  {conv.admin_reply && (
                    <div className="space-y-1">
                      <p className="text-xs font-medium text-green-400 uppercase tracking-wide flex items-center gap-1">
                        <Bot className="h-3 w-3" />
                        الرد الخاص (رسالة مباشرة · AI)
                      </p>
                      <div className="rounded-lg border border-green-500/20 bg-green-500/5 px-4 py-3 text-sm text-green-200">
                        {conv.admin_reply}
                      </div>
                    </div>
                  )}

                  {/* Guardrail / escalation */}
                  {(conv.escalation_reason || conv.guardrail_triggered) && (
                    <div className="flex items-start gap-2 rounded-lg bg-red-500/10 border border-red-500/20 px-3 py-2">
                      <AlertCircle className="h-4 w-4 text-red-400 mt-0.5 shrink-0" />
                      <p className="text-xs text-red-300">
                        {conv.guardrail_triggered ? `حارس النظام: ${conv.guardrail_reason ?? "تم التفعيل"}` : conv.escalation_reason}
                      </p>
                    </div>
                  )}

                  {/* Replied at */}
                  {conv.replied_at && (
                    <div className="flex items-center gap-1 text-xs text-muted-foreground">
                      <CheckCircle2 className="h-3 w-3 text-green-400" />
                      تم الرد في {formatDateTime(conv.replied_at)}
                    </div>
                  )}
                </div>
              ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
}
