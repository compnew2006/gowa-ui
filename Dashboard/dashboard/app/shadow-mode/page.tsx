"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { usePageContext } from "@/lib/page-context";
import { cn } from "@/lib/utils";
import { Eye, CheckCircle, XCircle, Bot, Clock, MessageSquare, Facebook, Smartphone } from "lucide-react";
import { toast } from "sonner";
import { useCan } from "@/components/auth/permission-guard";

export default function ShadowModePage() {
  const qc = useQueryClient();
  const { selectedPageId } = usePageContext();
  const [page, setPage] = useState(1);
  const [expandedId, setExpandedId] = useState<string | null>(null);
  const [correctingId, setCorrectingId] = useState<string | null>(null);
  const [correctIntent, setCorrectIntent] = useState("");
  const [correctSentiment, setCorrectSentiment] = useState("");
  const canApprove = useCan("can_approve");
  const canReject = useCan("can_reject");

  const shadowParams: Record<string, string> = { status: "shadow_pending", page: String(page), limit: "20" };
  if (selectedPageId) shadowParams.page_id = selectedPageId;

  const { data, isLoading } = useQuery({
    queryKey: ["shadow-pending", selectedPageId, page],
    queryFn: () => api.getConversations(shadowParams),
    refetchInterval: 15_000,
  });

  const approveMutation = useMutation({
    mutationFn: (id: string) => api.approveShadow(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["shadow-pending"] }); toast.success("تم الموافقة وإرسال الرد"); },
    onError: () => toast.error("فشلت الموافقة"),
  });

  const rejectMutation = useMutation({
    mutationFn: (id: string) => api.rejectShadow(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["shadow-pending"] }); toast.success("تم الرفض"); },
    onError: () => toast.error("فشل الرفض"),
  });

  const undoMutation = useMutation({
    mutationFn: (id: string) => api.undoShadow(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["shadow-pending"] }); toast.success("تم الرجوع عن القرار"); },
    onError: () => toast.error("فشل الرجوع"),
  });

  const correctMutation = useMutation({
    mutationFn: (id: string) => api.correctShadow(id, correctIntent || undefined, correctSentiment || undefined),
    onSuccess: () => { 
      qc.invalidateQueries({ queryKey: ["shadow-pending"] }); 
      setCorrectingId(null);
      setCorrectIntent("");
      setCorrectSentiment("");
      toast.success("تم تصحيح التقييم"); 
    },
    onError: () => toast.error("فشل التصحيح"),
  });

  const conversations = data?.data ?? [];
  const total = data?.total ?? 0;

  return (
    <div className="p-6 space-y-5">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">وضع الظل</h1>
          <p className="text-sm text-muted-foreground mt-1">مراجعة ردود الذكاء الاصطناعي قبل الإرسال</p>
        </div>
        <div className="flex items-center gap-2 rounded-lg border border-yellow-500/30 bg-yellow-500/10 px-3 py-1.5">
          <div className="h-2 w-2 rounded-full bg-yellow-400 animate-pulse" />
          <span className="text-xs text-yellow-400 font-medium">{total} في الانتظار</span>
        </div>
      </div>

      {isLoading ? (
        <div className="flex h-48 items-center justify-center text-muted-foreground">جاري التحميل...</div>
      ) : conversations.length === 0 ? (
        <div className="flex flex-col items-center justify-center rounded-xl border border-border bg-card h-64 gap-3 text-muted-foreground">
          <Eye className="h-10 w-10 opacity-30" />
          <div className="text-center">
            <p className="font-medium">لا توجد ردود في الانتظار</p>
            <p className="text-sm mt-1">جميع الردود مراجعة ✓</p>
          </div>
        </div>
      ) : (
        <div className="space-y-4">
          {conversations.map((conv) => (
            <div key={conv.id} className="rounded-xl border border-border bg-card p-5 space-y-4">
              <div className="flex items-start justify-between gap-4">
                <div className="flex-1 min-w-0">
                  <div className="flex items-center gap-2 mb-1">
                    <span className="font-medium text-sm text-foreground">{conv.customer_name}</span>
                    <div className="flex items-center gap-1 bg-accent/50 rounded-md px-2 py-0.5">
                      {conv.platform === "facebook" && <Facebook className="h-3 w-3 text-blue-500" />}
                      {conv.platform === "whatsapp" && <Smartphone className="h-3 w-3 text-green-500" />}
                      {conv.platform === "instagram" && <div className="h-3 w-3 rounded-full bg-gradient-to-tr from-yellow-400 via-red-500 to-purple-500" />}
                      <span className="text-[10px] text-muted-foreground uppercase font-bold tracking-wider">{conv.platform}</span>
                    </div>
                    <span className="text-xs text-muted-foreground bg-accent rounded px-1.5 py-0.5">{conv.page_name}</span>
                    {conv.intent && <span className="text-xs text-blue-400 bg-blue-500/10 rounded px-1.5 py-0.5">{conv.intent}</span>}

                    {conv.sentiment && (
                      <span className={cn("text-xs rounded px-1.5 py-0.5",
                        conv.sentiment === "positive" ? "text-green-400 bg-green-500/10" :
                        conv.sentiment === "angry" ? "text-red-400 bg-red-500/10" :
                        "text-yellow-400 bg-yellow-500/10"
                      )}>{conv.sentiment}</span>
                    )}
                  </div>
                  <p className="text-xs text-muted-foreground mb-3">
                    <Clock className="h-3 w-3 inline mr-1" />
                    {new Date(conv.created_at).toLocaleString("ar-SA")}
                    {conv.confidence_score && ` · ثقة: ${(conv.confidence_score * 100).toFixed(1)}%`}
                  </p>

                  {/* Original comment */}
                  <div className="rounded-lg bg-muted/30 border border-border p-3 mb-3">
                    <div className="flex items-center gap-1.5 mb-1.5">
                      <MessageSquare className="h-3.5 w-3.5 text-muted-foreground" />
                      <span className="text-xs text-muted-foreground">التعليق الأصلي</span>
                    </div>
                    <p className="text-sm text-foreground">{conv.original_comment}</p>
                  </div>

                  {/* AI Reply */}
                  {conv.ai_reply && (
                    <div className="rounded-lg bg-primary/5 border border-primary/20 p-3">
                      <div className="flex items-center gap-1.5 mb-1.5">
                        <Bot className="h-3.5 w-3.5 text-primary" />
                        <span className="text-xs text-primary">رد الذكاء الاصطناعي المقترح</span>
                      </div>
                      <p className="text-sm text-foreground">{conv.ai_reply}</p>
                    </div>
                  )}
                </div>
              </div>

              {/* Action buttons */}
              <div className="flex gap-2 pt-3 border-t border-border flex-wrap">
                {canApprove && (
                  <button
                    onClick={() => approveMutation.mutate(conv.id)}
                    disabled={approveMutation.isPending}
                    className="flex-1 rounded-lg bg-green-500/20 text-green-400 hover:bg-green-500/30 px-3 py-2 text-sm font-medium transition-colors disabled:opacity-50"
                  >
                    <CheckCircle className="h-4 w-4 inline mr-1" />
                    موافقة
                  </button>
                )}
                {canReject && (
                  <button
                    onClick={() => rejectMutation.mutate(conv.id)}
                    disabled={rejectMutation.isPending}
                    className="flex-1 rounded-lg bg-red-500/20 text-red-400 hover:bg-red-500/30 px-3 py-2 text-sm font-medium transition-colors disabled:opacity-50"
                  >
                    <XCircle className="h-4 w-4 inline mr-1" />
                    رفض
                  </button>
                )}
                <button
                  onClick={() => setExpandedId(expandedId === conv.id ? null : conv.id)}
                  className="rounded-lg bg-accent hover:bg-accent/70 px-3 py-2 text-sm font-medium transition-colors"
                >
                  {expandedId === conv.id ? "إغلاق" : "خيارات"}
                </button>
              </div>

              {/* Expanded options */}
              {expandedId === conv.id && (
                <div className="space-y-3 pt-3 border-t border-border">
                  {/* Correct option */}
                  <div className="rounded-lg bg-blue-500/10 border border-blue-500/20 p-3 space-y-2">
                    <p className="text-xs font-medium text-blue-400">تصحيح التقييم</p>
                    <div className="space-y-2">
                      <input
                        type="text"
                        placeholder="تصحيح النية (مثال: complaint)"
                        value={correctingId === conv.id ? correctIntent : ""}
                        onChange={(e) => { setCorrectingId(conv.id); setCorrectIntent(e.target.value); }}
                        className="w-full rounded-lg border border-border bg-background px-2.5 py-1.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
                      />
                      <input
                        type="text"
                        placeholder="تصحيح المشاعر (مثال: angry)"
                        value={correctingId === conv.id ? correctSentiment : ""}
                        onChange={(e) => { setCorrectingId(conv.id); setCorrectSentiment(e.target.value); }}
                        className="w-full rounded-lg border border-border bg-background px-2.5 py-1.5 text-xs text-foreground focus:outline-none focus:ring-1 focus:ring-primary"
                      />
                      <button
                        onClick={() => correctMutation.mutate(conv.id)}
                        disabled={correctMutation.isPending || !correctIntent && !correctSentiment}
                        className="w-full rounded-lg bg-primary px-2.5 py-1.5 text-xs font-medium text-primary-foreground hover:bg-primary/90 disabled:opacity-50 transition-colors"
                      >
                        {correctMutation.isPending ? "جاري..." : "تصحيح"}
                      </button>
                    </div>
                  </div>

                  {/* Undo option */}
                  <button
                    onClick={() => undoMutation.mutate(conv.id)}
                    disabled={undoMutation.isPending}
                    className="w-full rounded-lg bg-yellow-500/10 border border-yellow-500/20 hover:bg-yellow-500/20 px-3 py-2 text-xs font-medium text-yellow-400 transition-colors disabled:opacity-50"
                  >
                    الرجوع عن القرار
                  </button>
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {/* Pagination */}
      {total > 20 && (
        <div className="flex items-center justify-center gap-2">
          <button onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page === 1}
            className="px-3 py-1.5 text-sm rounded-lg border border-border disabled:opacity-40 hover:bg-accent transition-colors">السابق</button>
          <span className="text-sm text-muted-foreground">صفحة {page} من {Math.ceil(total / 20)}</span>
          <button onClick={() => setPage(p => p + 1)} disabled={page * 20 >= total}
            className="px-3 py-1.5 text-sm rounded-lg border border-border disabled:opacity-40 hover:bg-accent transition-colors">التالي</button>
        </div>
      )}
    </div>
  );
}
