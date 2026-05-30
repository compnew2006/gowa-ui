"use client";

import { useState } from "react";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { usePageContext } from "@/lib/page-context";
import { cn, formatRelativeTime, getStatusColor } from "@/lib/utils";
import { MessageSquare, Search, Send, CheckCircle, Bot, ThumbsUp, Facebook, Smartphone } from "lucide-react";
import { toast } from "sonner";
import { useCan } from "@/components/auth/permission-guard";

export default function ConversationsPage() {
  const qc = useQueryClient();
  const { selectedPageId } = usePageContext();
  const [page, setPage] = useState(1);
  const [status, setStatus] = useState("");
  const [search, setSearch] = useState("");
  const [replyId, setReplyId] = useState<string | null>(null);
  const [replyText, setReplyText] = useState("");
  const [selectedIds, setSelectedIds] = useState<string[]>([]);
  const canApprove = useCan("can_approve");

  const params: Record<string, string> = { page: String(page), limit: "20" };
  if (status) params.status = status;
  if (search) params.search = search;
  if (selectedPageId) params.page_id = selectedPageId;

  const { data, isLoading } = useQuery({
    queryKey: ["conversations", params],
    queryFn: () => api.getConversations(params),
    refetchInterval: 4_000, // Poll every 4 seconds for real-time comment updates
  });

  const replyMutation = useMutation({
    mutationFn: ({ id, reply }: { id: string; reply: string }) => api.manualReply(id, reply),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["conversations"] }); setReplyId(null); setReplyText(""); toast.success("تم إرسال الرد"); },
    onError: () => toast.error("فشل الإرسال"),
  });



  const approveMutation = useMutation({
    mutationFn: ({ id, reply }: { id: string; reply?: string }) => api.approveReply(id, reply),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["conversations"] }); toast.success("تم نشر الرد على فيسبوك"); },
    onError: () => toast.error("فشل نشر الرد"),
  });

  const resolveMutation = useMutation({
    mutationFn: (id: string) => api.resolveConversation(id),
    onSuccess: () => { qc.invalidateQueries({ queryKey: ["conversations"] }); toast.success("تم الحل"); },
    onError: () => toast.error("فشل الإجراء"),
  });

  const bulkMutation = useMutation({
    mutationFn: (action: string) => api.bulkConversationAction(selectedIds, action),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["conversations"] });
      setSelectedIds([]);
      toast.success("تم تنفيذ الإجراء الجماعي");
    },
    onError: () => toast.error("فشل الإجراء الجماعي"),
  });

  const toggleSelected = (id: string) => {
    setSelectedIds(ids => ids.includes(id) ? ids.filter(item => item !== id) : [...ids, id]);
  };

  return (
    <div className="p-6 space-y-5">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">المحادثات</h1>
          <p className="text-sm text-muted-foreground mt-1">{data?.total ?? 0} محادثة إجمالاً</p>
        </div>
      </div>

      {/* Filters */}
      <div className="flex gap-3 flex-wrap">
        <div className="relative flex-1 min-w-[200px]">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <input
            className="w-full rounded-lg border border-border bg-card pl-9 pr-4 py-2 text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/50"
            placeholder="البحث في التعليقات..."
            value={search}
            onChange={(e) => { setSearch(e.target.value); setPage(1); }}
          />
        </div>
        <select
          className="rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50"
          value={status}
          onChange={(e) => { setStatus(e.target.value); setPage(1); }}
        >
          <option value="">كل الحالات</option>
          <option value="pending">في الانتظار</option>
          <option value="replied">مُجاب</option>
          <option value="escalated">مُصعَّد</option>
          <option value="resolved">مُحلّ</option>
          <option value="shadow_pending">بانتظار الموافقة</option>
        </select>
      </div>

      {selectedIds.length > 0 && (
        <div className="flex flex-wrap items-center gap-2 rounded-lg border border-border bg-card px-3 py-2">
          <span className="text-sm text-muted-foreground">{selectedIds.length} محددة</span>
          <button
            onClick={() => bulkMutation.mutate("resolve")}
            disabled={bulkMutation.isPending}
            className="rounded-lg border border-border px-3 py-1.5 text-xs text-muted-foreground hover:bg-accent disabled:opacity-50"
          >
            حل المحدد
          </button>
          {canApprove && (
            <button
              onClick={() => bulkMutation.mutate("approve")}
              disabled={bulkMutation.isPending}
              className="rounded-lg border border-border px-3 py-1.5 text-xs text-muted-foreground hover:bg-accent disabled:opacity-50"
            >
              موافقة جماعية
            </button>
          )}
          <button onClick={() => setSelectedIds([])} className="text-xs text-muted-foreground hover:text-foreground">إلغاء التحديد</button>
        </div>
      )}

      {/* Table */}
      <div className="rounded-xl border border-border bg-card overflow-hidden">
        {isLoading ? (
          <div className="flex items-center justify-center h-48 text-muted-foreground">جاري التحميل...</div>
        ) : (
          <div className="divide-y divide-border">
            {(data?.data ?? []).map((conv) => (
              <div key={conv.id} className="p-4 hover:bg-accent/20 transition-colors">
                <div className="flex items-start justify-between gap-4">
                  <input
                    type="checkbox"
                    checked={selectedIds.includes(conv.id)}
                    onChange={() => toggleSelected(conv.id)}
                    className="mt-1 rounded border-border"
                    aria-label={`تحديد محادثة ${conv.customer_name}`}
                  />
                  <div className="flex-1 min-w-0">
                    <div className="flex items-center gap-2 mb-1">
                      <span className="font-medium text-sm text-foreground">{conv.customer_name}</span>
                      <div className="flex items-center gap-1.5 ml-auto">
                        {conv.platform === "facebook" && <Facebook className="h-3.5 w-3.5 text-blue-500" />}
                        {conv.platform === "whatsapp" && <Smartphone className="h-3.5 w-3.5 text-green-500" />}
                        {conv.platform === "instagram" && <div className="h-3.5 w-3.5 rounded bg-gradient-to-tr from-yellow-400 via-red-500 to-purple-500" />}
                        <span className="text-xs text-muted-foreground">{conv.platform}</span>
                      </div>
                      <span className={cn("text-xs px-2 py-0.5 rounded-full border", getStatusColor(conv.status))}>{conv.status === "shadow_pending" ? "بانتظار الموافقة" : conv.status === "escalated" ? "مُصعَّد" : conv.status === "replied" ? "مُجاب" : conv.status === "pending" ? "في الانتظار" : conv.status === "resolved" ? "مُحلّ" : conv.status}</span>
                      {conv.intent && <span className="text-xs text-muted-foreground bg-accent rounded px-1.5 py-0.5">{conv.intent}</span>}

                      {conv.language && <span className="text-xs text-muted-foreground">{conv.language.toUpperCase()}</span>}
                    </div>
                    <p className="text-sm text-foreground/80 mb-2">{conv.original_comment}</p>
                    {conv.ai_reply && (
                      <div className="flex items-start gap-2 bg-primary/5 border border-primary/20 rounded-lg p-2.5 mb-2">
                        <Bot className="h-3.5 w-3.5 text-primary mt-0.5 shrink-0" />
                        <p className="text-xs text-foreground/70">{conv.ai_reply}</p>
                      </div>
                    )}
                    <p className="text-xs text-muted-foreground">{conv.page_name} · {formatRelativeTime(conv.created_at)}</p>
                  </div>
                  <div className="flex gap-2 shrink-0">
                    {conv.status !== "resolved" && (
                      <>
	{canApprove && conv.ai_reply && (conv.status === "shadow_pending" || conv.status === "pending" || conv.status === "escalated") && (
                        <button
                          onClick={() => approveMutation.mutate({ id: conv.id })}
                          disabled={approveMutation.isPending}
                          className="p-1.5 rounded-lg hover:bg-green-100 text-muted-foreground hover:text-green-600 transition-colors disabled:opacity-50"
                          title="موافقة ونشر على فيسبوك"
                        >
                          <ThumbsUp className="h-4 w-4" />
                        </button>
                      )}
                        <button
                          onClick={() => setReplyId(replyId === conv.id ? null : conv.id)}
                          className="p-1.5 rounded-lg hover:bg-accent text-muted-foreground hover:text-foreground transition-colors"
                          title="رد يدوي"
                        >
                          <Send className="h-4 w-4" />
                        </button>
                        <button
                          onClick={() => resolveMutation.mutate(conv.id)}
                          className="p-1.5 rounded-lg hover:bg-accent text-muted-foreground hover:text-foreground transition-colors"
                          title="حل"
                        >
                          <CheckCircle className="h-4 w-4" />
                        </button>
                      </>
                    )}
                  </div>
                </div>
                {replyId === conv.id && (
                  <div className="mt-3 flex gap-2">
                    <input
                      className="flex-1 rounded-lg border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-primary/50"
                      placeholder="اكتب ردك هنا..."
                      value={replyText}
                      onChange={(e) => setReplyText(e.target.value)}
                      onKeyDown={(e) => e.key === "Enter" && replyText && replyMutation.mutate({ id: conv.id, reply: replyText })}
                    />
                    <button
                      onClick={() => replyMutation.mutate({ id: conv.id, reply: replyText })}
                      disabled={!replyText || replyMutation.isPending}
                      className="rounded-lg bg-primary px-4 py-2 text-sm font-medium text-primary-foreground disabled:opacity-50"
                    >
                      إرسال
                    </button>
                  </div>
                )}
              </div>
            ))}
            {data?.data.length === 0 && (
              <div className="flex flex-col items-center justify-center h-48 gap-2 text-muted-foreground">
                <MessageSquare className="h-8 w-8 opacity-40" />
                <p className="text-sm">لا توجد محادثات</p>
              </div>
            )}
          </div>
        )}
      </div>

      {/* Pagination */}
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
