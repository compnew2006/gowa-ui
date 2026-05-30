"use client";

import { useQuery } from "@tanstack/react-query";
import { api } from "@/lib/api";
import { usePageContext } from "@/lib/page-context";
import { cn } from "@/lib/utils";
import { useState } from "react";
import {
  TrendingUp, Users, DollarSign, Zap, Globe,
  AlertTriangle, Clock, Target, Activity,
} from "lucide-react";
import {
  AreaChart, Area, BarChart, Bar,
  LineChart, Line, PieChart, Pie, Cell,
  XAxis, YAxis, CartesianGrid, Tooltip, ResponsiveContainer, Legend,
} from "recharts";

const COLORS = ["#3b82f6", "#8b5cf6", "#06b6d4", "#10b981", "#f59e0b", "#ef4444"];

function StatCard({ title, value, sub, icon: Icon, color, prefix = "" }: any) {
  return (
    <div className="rounded-xl border border-border bg-card p-5">
      <div className="flex items-start justify-between">
        <div>
          <p className="text-sm text-muted-foreground">{title}</p>
          <p className="mt-1 text-2xl font-bold text-foreground">{prefix}{value ?? "—"}</p>
          {sub && <p className="mt-1 text-xs text-muted-foreground">{sub}</p>}
        </div>
        <div className={cn("rounded-lg p-2", color)}>
          <Icon className="h-5 w-5" />
        </div>
      </div>
    </div>
  );
}

export default function AnalyticsPage() {
  const { selectedPageId } = usePageContext();
  const [period, setPeriod] = useState("7d");

  const { data: analytics } = useQuery({
    queryKey: ["analytics-summary", selectedPageId, period],
    queryFn: () => api.getAdvancedAnalyticsSummary(period, selectedPageId),
    refetchInterval: 60_000,
  });

  const roi = analytics?.roi;
  const funnel = analytics?.funnel;
  const perf = analytics?.performance;
  const langBreakdown = analytics?.language_breakdown;
  const churnData = analytics?.churn_risk;
  const respTrend = analytics?.response_time_trend;

  const churnPieData = churnData
    ? [
        { name: "منخفض", value: churnData.distribution.low, color: "#10b981" },
        { name: "متوسط", value: churnData.distribution.medium, color: "#f59e0b" },
        { name: "مرتفع", value: churnData.distribution.high, color: "#ef4444" },
      ]
    : [];

  return (
    <div className="p-6 space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold text-foreground">التحليلات المتقدمة</h1>
          <p className="text-sm text-muted-foreground mt-1">عائد الاستثمار، قمع التحويل، أداء الذكاء الاصطناعي</p>
        </div>
        <div className="flex rounded-lg border border-border overflow-hidden">
          {["7d", "30d"].map((p) => (
            <button
              key={p}
              onClick={() => setPeriod(p)}
              className={cn(
                "px-4 py-1.5 text-sm font-medium transition-colors",
                period === p ? "bg-primary text-primary-foreground" : "text-muted-foreground hover:bg-accent"
              )}
            >
              {p === "7d" ? "7 أيام" : "30 يوم"}
            </button>
          ))}
        </div>
      </div>

      {/* ROI Stats */}
      <div className="grid grid-cols-2 gap-4 lg:grid-cols-4">
        <StatCard title="الوقت الموفر" value={roi?.estimated_time_saved_hours} sub="ساعة عمل" icon={Clock} color="bg-blue-500/20 text-blue-400" />
        <StatCard title="التوفير المقدر" value={roi?.estimated_cost_saved_usd?.toFixed(0)} sub="دولار أمريكي" icon={DollarSign} color="bg-green-500/20 text-green-400" prefix="$" />
        <StatCard title="معدل الرد التلقائي" value={roi?.auto_reply_rate_pct ? `${roi.auto_reply_rate_pct}%` : undefined} sub="من إجمالي التعليقات" icon={Zap} color="bg-indigo-500/20 text-indigo-400" />
        <StatCard title="نسبة التحويل" value={roi?.conversion_rate_pct ? `${roi.conversion_rate_pct}%` : undefined} sub={`${roi?.converted_customers ?? 0} تحويل`} icon={Target} color="bg-purple-500/20 text-purple-400" />
      </div>

      {/* Charts row 1 */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        {/* AI Performance Trend */}
        <div className="rounded-xl border border-border bg-card p-5">
          <div className="flex items-center gap-2 mb-4">
            <Activity className="h-4 w-4 text-muted-foreground" />
            <h2 className="font-semibold text-foreground">أداء الذكاء الاصطناعي</h2>
          </div>
          <ResponsiveContainer width="100%" height={220}>
            <LineChart data={perf ?? []}>
              <CartesianGrid strokeDasharray="3 3" stroke="#1e2b47" />
              <XAxis dataKey="date" tick={{ fill: "#64748b", fontSize: 11 }} tickFormatter={(v) => v.slice(5)} />
              <YAxis tick={{ fill: "#64748b", fontSize: 11 }} unit="%" />
              <Tooltip contentStyle={{ background: "#0f172a", border: "1px solid #1e293b", borderRadius: 8 }} />
              <Legend />
              <Line type="monotone" dataKey="avg_confidence" stroke="#3b82f6" dot={false} name="الثقة %" />
              <Line type="monotone" dataKey="auto_reply_rate" stroke="#10b981" dot={false} name="الرد التلقائي %" strokeDasharray="4 2" />
              <Line type="monotone" dataKey="escalation_rate" stroke="#ef4444" dot={false} name="التصعيد %" strokeDasharray="4 2" />
            </LineChart>
          </ResponsiveContainer>
        </div>

        {/* Response Time Trend */}
        <div className="rounded-xl border border-border bg-card p-5">
          <div className="flex items-center gap-2 mb-4">
            <Clock className="h-4 w-4 text-muted-foreground" />
            <h2 className="font-semibold text-foreground">وقت الاستجابة</h2>
          </div>
          <ResponsiveContainer width="100%" height={220}>
            <AreaChart data={respTrend ?? []}>
              <defs>
                <linearGradient id="rtGrad" x1="0" y1="0" x2="0" y2="1">
                  <stop offset="5%" stopColor="#8b5cf6" stopOpacity={0.3} />
                  <stop offset="95%" stopColor="#8b5cf6" stopOpacity={0} />
                </linearGradient>
              </defs>
              <CartesianGrid strokeDasharray="3 3" stroke="#1e2b47" />
              <XAxis dataKey="date" tick={{ fill: "#64748b", fontSize: 11 }} tickFormatter={(v) => v.slice(5)} />
              <YAxis tick={{ fill: "#64748b", fontSize: 11 }} unit="s" />
              <Tooltip contentStyle={{ background: "#0f172a", border: "1px solid #1e293b", borderRadius: 8 }} />
              <Area type="monotone" dataKey="avg_response_time_sec" stroke="#8b5cf6" fill="url(#rtGrad)" name="ثانية" />
            </AreaChart>
          </ResponsiveContainer>
        </div>
      </div>

      {/* Charts row 2 */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        {/* Conversion Funnel */}
        <div className="rounded-xl border border-border bg-card p-5 lg:col-span-2">
          <div className="flex items-center gap-2 mb-4">
            <TrendingUp className="h-4 w-4 text-muted-foreground" />
            <h2 className="font-semibold text-foreground">قمع التحويل</h2>
          </div>
          {funnel?.stages && funnel.stages.length > 0 ? (
            <div className="space-y-2">
              {funnel.stages.map((stage: any, i: number) => (
                <div key={stage.stage} className="flex items-center gap-3">
                  <span className="text-xs text-muted-foreground w-28 text-right shrink-0">{stage.stage}</span>
                  <div className="flex-1 bg-muted/30 rounded-full h-6 overflow-hidden">
                    <div
                      className="h-full rounded-full flex items-center justify-end pr-2 text-xs font-medium text-white transition-all"
                      style={{
                        width: `${Math.max(stage.pct, 2)}%`,
                        background: COLORS[i % COLORS.length],
                      }}
                    >
                      {stage.count}
                    </div>
                  </div>
                  <span className="text-xs text-muted-foreground w-12 shrink-0">{stage.pct}%</span>
                </div>
              ))}
            </div>
          ) : (
            <div className="flex h-48 items-center justify-center text-muted-foreground text-sm">لا توجد بيانات</div>
          )}
        </div>

        {/* Churn Risk Pie */}
        <div className="rounded-xl border border-border bg-card p-5">
          <div className="flex items-center gap-2 mb-4">
            <AlertTriangle className="h-4 w-4 text-muted-foreground" />
            <h2 className="font-semibold text-foreground">مخاطر المغادرة</h2>
          </div>
          {churnPieData.some(d => d.value > 0) ? (
            <ResponsiveContainer width="100%" height={160}>
              <PieChart>
                <Pie data={churnPieData} dataKey="value" nameKey="name" cx="50%" cy="50%" outerRadius={60} label={({ name, value }) => `${name}: ${value}`} labelLine={false}>
                  {churnPieData.map((entry) => (
                    <Cell key={entry.name} fill={entry.color} />
                  ))}
                </Pie>
                <Tooltip contentStyle={{ background: "#0f172a", border: "1px solid #1e293b", borderRadius: 8 }} />
              </PieChart>
            </ResponsiveContainer>
          ) : (
            <div className="flex h-40 items-center justify-center text-muted-foreground text-sm">لا توجد بيانات</div>
          )}
        </div>
      </div>

      {/* Language + High Churn Customers */}
      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        {/* Language Breakdown */}
        <div className="rounded-xl border border-border bg-card p-5">
          <div className="flex items-center gap-2 mb-4">
            <Globe className="h-4 w-4 text-muted-foreground" />
            <h2 className="font-semibold text-foreground">توزيع اللغات</h2>
          </div>
          {langBreakdown && langBreakdown.length > 0 ? (
            <ResponsiveContainer width="100%" height={200}>
              <BarChart data={langBreakdown} layout="vertical">
                <CartesianGrid strokeDasharray="3 3" stroke="#1e2b47" horizontal={false} />
                <XAxis type="number" tick={{ fill: "#64748b", fontSize: 11 }} />
                <YAxis type="category" dataKey="language" tick={{ fill: "#64748b", fontSize: 11 }} width={60} />
                <Tooltip contentStyle={{ background: "#0f172a", border: "1px solid #1e293b", borderRadius: 8 }} />
                <Bar dataKey="count" radius={[0, 4, 4, 0]}>
                  {langBreakdown.map((item: any, i: number) => (
                    <Cell key={item.language} fill={COLORS[i % COLORS.length]} />
                  ))}
                </Bar>
              </BarChart>
            </ResponsiveContainer>
          ) : (
            <div className="flex h-48 items-center justify-center text-muted-foreground text-sm">لا توجد بيانات</div>
          )}
        </div>

        {/* High Churn Customers */}
        <div className="rounded-xl border border-border bg-card p-5">
          <div className="flex items-center gap-2 mb-4">
            <Users className="h-4 w-4 text-muted-foreground" />
            <h2 className="font-semibold text-foreground">عملاء في خطر مغادرة</h2>
          </div>
          <div className="space-y-3">
            {(churnData?.high_risk_customers ?? []).length === 0 ? (
              <div className="flex h-40 items-center justify-center text-muted-foreground text-sm">لا يوجد عملاء في خطر عالٍ</div>
            ) : (
              churnData!.high_risk_customers.map((c: any) => (
                <div key={c.id} className="flex items-start gap-3 p-3 rounded-lg bg-red-500/5 border border-red-500/20">
                  <div className="h-8 w-8 rounded-full bg-red-500/20 flex items-center justify-center text-xs font-bold text-red-400">
                    {(c.name ?? "?")[0].toUpperCase()}
                  </div>
                  <div className="flex-1 min-w-0">
                    <p className="text-sm font-medium text-foreground">{c.name}</p>
                    <p className="text-xs text-muted-foreground truncate">{c.next_best_action}</p>
                  </div>
                  <span className="text-xs bg-red-500/20 text-red-400 px-2 py-0.5 rounded-full shrink-0">
                    {Math.round((c.churn_risk_score ?? 0) * 100)}%
                  </span>
                </div>
              ))
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
