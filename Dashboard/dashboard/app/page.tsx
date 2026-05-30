"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { api } from "@/lib/api";
import { usePageContext } from "@/lib/page-context";
import { cn } from "@/lib/utils";
import { Skeleton } from "@/components/ui/skeleton";
import {
  CheckCircle2, Activity, Key,
} from "lucide-react";
import {
  AreaChart, Area, BarChart, Bar, PieChart, Pie, Cell,
  XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer,
} from "recharts";

const INTENT_COLORS = ["#3b82f6", "#8b5cf6", "#06b6d4", "#10b981", "#f59e0b", "#ef4444"];
const SENTIMENT_COLORS: Record<string, string> = {
  positive: "#10b981", neutral: "#6b7280", negative: "#f59e0b", angry: "#ef4444",
};
const CONFIDENCE_THRESHOLD = 0.85;

function TriageSkeleton() {
  return (
    <div className="rounded-lg border border-border p-4">
      <div className="flex items-center gap-4">
        <Skeleton className="h-2 w-2 rounded-full" />
        <Skeleton className="h-5 w-12" />
        <Skeleton className="h-4 w-24" />
        <Skeleton className="h-2 w-2 rounded-full" />
        <Skeleton className="h-5 w-8" />
        <Skeleton className="h-4 w-20" />
      </div>
    </div>
  );
}

function VolumeSkeleton() {
  return (
    <div className="flex flex-wrap gap-x-6 gap-y-1">
      <Skeleton className="h-4 w-28" />
      <Skeleton className="h-4 w-20" />
      <Skeleton className="h-4 w-32" />
      <Skeleton className="h-4 w-24" />
    </div>
  );
}

function ChartSkeleton({ height = 220 }: { height?: number }) {
  return (
    <div className="rounded-xl border border-border bg-card p-5">
      <div className="flex items-center gap-2 mb-4">
        <Skeleton className="h-4 w-4 rounded-full" />
        <Skeleton className="h-5 w-32" />
      </div>
      <Skeleton className="mx-auto" style={{ height, width: "100%" }} />
    </div>
  );
}

function ConfidenceBar({ value, className }: { value: number; className?: string }) {
  const color = value >= CONFIDENCE_THRESHOLD ? "bg-green-400" : value >= 0.6 ? "bg-yellow-400" : "bg-red-400";
  return (
    <div className={cn("flex items-center gap-2", className)}>
      <span className="text-xs text-muted-foreground">الثقة</span>
      <div className="h-1.5 w-16 rounded-full bg-muted overflow-hidden">
        <div className={cn("h-full rounded-full transition-all", color)} style={{ width: `${Math.min(value * 100, 100)}%` }} />
      </div>
      <span className={cn("text-xs tabular-nums", value >= CONFIDENCE_THRESHOLD ? "text-muted-foreground" : value >= 0.6 ? "text-yellow-400" : "text-red-400")}>
        {(value * 100).toFixed(0)}%
      </span>
    </div>
  );
}

export default function DashboardPage() {
  const { selectedPageId, selectedPage } = usePageContext();

  const { data: stats, isLoading: statsLoading } = useQuery({
    queryKey: ["dashboard-stats", selectedPageId],
    queryFn: () => api.getDashboardStats(selectedPageId),
    refetchInterval: 30_000,
  });

  const { data: convData } = useQuery({
    queryKey: ["conv-analytics", "7d", selectedPageId],
    queryFn: () => api.getConversationAnalytics("7d", selectedPageId),
  });

  const { data: intents } = useQuery({
    queryKey: ["intents", selectedPageId],
    queryFn: () => api.getIntentBreakdown(selectedPageId),
  });

  const { data: sentiments } = useQuery({
    queryKey: ["sentiments", selectedPageId],
    queryFn: () => api.getSentimentBreakdown(selectedPageId),
  });

  const hasAttentionItems = stats && (stats.pending_conversations > 0 || stats.open_escalations > 0);

  return (
    <div className="p-6 space-y-6">
      <div>
        <h1 className="text-2xl font-bold text-foreground">لوحة التحكم</h1>
        <p className="text-muted-foreground text-sm mt-1">
          {selectedPage ? `${selectedPage.name} — ` : ""}نظرة عامة على أداء أتمتة التعليقات
        </p>
      </div>

      {/* Triage strip */}
      {statsLoading ? <TriageSkeleton /> : stats && (
        <div className="rounded-lg border border-border p-4">
          {hasAttentionItems ? (
            <div className="flex flex-wrap items-center gap-x-8 gap-y-3">
              {stats.pending_conversations > 0 && (
                <Link href="/shadow-mode" className="flex items-center gap-2 group">
                  <span className="h-2 w-2 rounded-full bg-yellow-400 shrink-0" />
                  <span className="text-lg font-semibold text-foreground tabular-nums">{stats.pending_conversations}</span>
                  <span className="text-sm text-muted-foreground">مراجعات معلقة</span>
                  <span className="text-xs text-primary opacity-0 group-hover:opacity-100 transition-opacity">مراجعة</span>
                </Link>
              )}
              {stats.open_escalations > 0 && (
                <Link href="/escalations" className="flex items-center gap-2 group">
                  <span className="h-2 w-2 rounded-full bg-red-400 shrink-0" />
                  <span className="text-lg font-semibold text-foreground tabular-nums">{stats.open_escalations}</span>
                  <span className="text-sm text-muted-foreground">تصعيدات مفتوحة</span>
                  <span className="text-xs text-primary opacity-0 group-hover:opacity-100 transition-opacity">مراجعة</span>
                </Link>
              )}
              <ConfidenceBar value={stats.avg_confidence_score} className="mr-auto" />
            </div>
          ) : (
            <div className="flex flex-wrap items-center gap-x-8 gap-y-3">
              <div className="flex items-center gap-2">
                <CheckCircle2 className="h-4 w-4 text-green-400" />
                <span className="text-sm text-muted-foreground">لا توجد عناصر تحتاج مراجعة</span>
              </div>
              <ConfidenceBar value={stats.avg_confidence_score} className="mr-auto" />
            </div>
          )}
        </div>
      )}

      {/* Volume row */}
      {statsLoading ? <VolumeSkeleton /> : stats && (
        <div className="flex flex-wrap gap-x-6 gap-y-1 text-sm text-muted-foreground">
          <span>إجمالي المحادثات: <strong className="text-foreground tabular-nums">{stats.total_conversations}</strong></span>
          <span>العملاء: <strong className="text-foreground tabular-nums">{stats.total_customers}</strong></span>
          <span>عملاء بنية شراء عالية: <strong className="text-foreground tabular-nums">{stats.high_intent_leads}</strong></span>
          <span>معدل الرد التلقائي: <strong className="text-foreground tabular-nums">{stats.auto_reply_rate.toFixed(1)}%</strong></span>
          <span>مراجعات وضع الظل: <strong className="text-foreground tabular-nums">{stats.shadow_mode_reviews}</strong></span>
        </div>
      )}

      {/* Token Health */}
      {statsLoading ? <ChartSkeleton height={80} /> : stats && (
        <div className="rounded-xl border border-border bg-card p-5">
          <div className="flex items-center gap-2 mb-4">
            <Key className="h-4 w-4 text-muted-foreground" />
            <h2 className="font-semibold text-foreground">صحة الرموز</h2>
          </div>
          <div className="flex gap-6">
            <div className="flex items-center gap-2">
              <div className="h-2.5 w-2.5 rounded-full bg-green-400" />
              <span className="text-sm text-muted-foreground">صالح: <strong className="text-foreground">{stats.token_healthy}</strong></span>
            </div>
            <div className="flex items-center gap-2">
              <div className="h-2.5 w-2.5 rounded-full bg-yellow-400" />
              <span className="text-sm text-muted-foreground">تنتهي قريباً: <strong className="text-foreground">{stats.token_expiring_soon}</strong></span>
            </div>
            <div className="flex items-center gap-2">
              <div className="h-2.5 w-2.5 rounded-full bg-red-400" />
              <span className="text-sm text-muted-foreground">منتهي: <strong className="text-foreground">{stats.token_expired}</strong></span>
            </div>
          </div>
        </div>
      )}

      {/* Charts row */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        {!convData ? <ChartSkeleton height={220} /> : (
          <div className="rounded-xl border border-border bg-card p-5">
            <div className="flex items-center gap-2 mb-4">
              <Activity className="h-4 w-4 text-muted-foreground" />
              <h2 className="font-semibold text-foreground">المحادثات — 7 أيام</h2>
            </div>
            <ResponsiveContainer width="100%" height={220}>
              <AreaChart data={convData}>
                <defs>
                  <linearGradient id="totalGrad" x1="0" y1="0" x2="0" y2="1">
                    <stop offset="5%" stopColor="#3b82f6" stopOpacity={0.3} />
                    <stop offset="95%" stopColor="#3b82f6" stopOpacity={0} />
                  </linearGradient>
                </defs>
                <CartesianGrid strokeDasharray="3 3" stroke="#1e2b47" />
                <XAxis dataKey="date" tick={{ fill: "#64748b", fontSize: 11 }} tickFormatter={(v) => v.slice(5)} />
                <YAxis tick={{ fill: "#64748b", fontSize: 11 }} />
                <Tooltip contentStyle={{ background: "#0f172a", border: "1px solid #1e293b", borderRadius: 8 }} />
                <Area type="monotone" dataKey="total" stroke="#3b82f6" fill="url(#totalGrad)" name="إجمالي" />
                <Area type="monotone" dataKey="replied" stroke="#10b981" fill="none" strokeDasharray="4 2" name="مُجاب" />
                <Area type="monotone" dataKey="escalated" stroke="#ef4444" fill="none" strokeDasharray="4 2" name="مُصعَّد" />
              </AreaChart>
            </ResponsiveContainer>
          </div>
        )}

        {!sentiments ? <ChartSkeleton height={220} /> : (
          <div className="rounded-xl border border-border bg-card p-5">
            <h2 className="font-semibold text-foreground mb-4">توزيع المشاعر</h2>
            {sentiments.length > 0 ? (
              <ResponsiveContainer width="100%" height={220}>
                <PieChart>
                  <Pie data={sentiments} dataKey="count" nameKey="sentiment" cx="50%" cy="50%" outerRadius={80} label={({ sentiment, percentage }) => `${sentiment} ${percentage}%`} labelLine={false}>
                    {sentiments.map((s) => (<Cell key={s.sentiment} fill={SENTIMENT_COLORS[s.sentiment] ?? "#6b7280"} />))}
                  </Pie>
                  <Tooltip contentStyle={{ background: "#0f172a", border: "1px solid #1e293b", borderRadius: 8 }} />
                </PieChart>
              </ResponsiveContainer>
            ) : (
              <div className="flex h-[220px] items-center justify-center text-muted-foreground text-sm">لا توجد بيانات</div>
            )}
          </div>
        )}

        {!intents ? <ChartSkeleton height={180} /> : (
          <div className="rounded-xl border border-border bg-card p-5 lg:col-span-2">
            <h2 className="font-semibold text-foreground mb-4">توزيع النوايا</h2>
            {intents.length > 0 ? (
              <ResponsiveContainer width="100%" height={180}>
                <BarChart data={intents} layout="vertical">
                  <CartesianGrid strokeDasharray="3 3" stroke="#1e2b47" horizontal={false} />
                  <XAxis type="number" tick={{ fill: "#64748b", fontSize: 11 }} />
                  <YAxis type="category" dataKey="intent" tick={{ fill: "#64748b", fontSize: 11 }} width={110} />
                  <Tooltip contentStyle={{ background: "#0f172a", border: "1px solid #1e293b", borderRadius: 8 }} />
                  <Bar dataKey="count" radius={[0, 4, 4, 0]}>
                    {intents.map((item, i) => (<Cell key={item.intent} fill={INTENT_COLORS[i % INTENT_COLORS.length]} />))}
                  </Bar>
                </BarChart>
              </ResponsiveContainer>
            ) : (
              <div className="flex h-[180px] items-center justify-center text-muted-foreground text-sm">لا توجد بيانات</div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
