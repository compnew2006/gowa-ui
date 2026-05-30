import { Router, type IRouter } from "express";
import { eq, count, avg, and, gte, SQL } from "drizzle-orm";
import { db, conversationsTable, escalationsTable, customersTable, pagesTable } from "@workspace/db";
import {
  GetConversationAnalyticsQueryParams,
  GetDashboardStatsResponse,
  GetConversationAnalyticsResponse,
  GetIntentBreakdownResponse,
  GetSentimentBreakdownResponse,
  GetResponseTimeStatsResponse,
} from "@workspace/api-zod";

const router: IRouter = Router();

router.get("/analytics/dashboard", async (_req, res): Promise<void> => {
  const [totalConvRow] = await db.select({ count: count() }).from(conversationsTable);
  const [pendingRow] = await db
    .select({ count: count() })
    .from(conversationsTable)
    .where(eq(conversationsTable.status, "pending"));
  const [openEscRow] = await db
    .select({ count: count() })
    .from(escalationsTable)
    .where(eq(escalationsTable.status, "open"));
  const [totalCustRow] = await db.select({ count: count() }).from(customersTable);
  const [highIntentRow] = await db
    .select({ count: count() })
    .from(customersTable)
    .where(eq(customersTable.purchaseIntent, "High"));
  const [shadowRow] = await db
    .select({ count: count() })
    .from(conversationsTable)
    .where(eq(conversationsTable.isShadowMode, true));

  const tokenStats = await db.select({ status: pagesTable.tokenStatus }).from(pagesTable);
  const tokenHealthy = tokenStats.filter((t) => t.status === "valid").length;
  const tokenExpiringSoon = tokenStats.filter((t) => t.status === "expiring_soon").length;
  const tokenExpired = tokenStats.filter((t) => t.status === "expired" || t.status === "error").length;

  const repliedConvs = await db
    .select({ processingTime: conversationsTable.processingTime })
    .from(conversationsTable)
    .where(eq(conversationsTable.status, "replied"));

  const avgResponseTime =
    repliedConvs.length > 0
      ? repliedConvs.reduce((s, c) => s + (c.processingTime ?? 0), 0) / repliedConvs.length
      : 3.2;

  const autoReplyRate =
    totalConvRow.count > 0
      ? ((totalConvRow.count - (pendingRow.count ?? 0)) / totalConvRow.count) * 100
      : 0;

  const convScores = await db
    .select({ score: conversationsTable.confidenceScore })
    .from(conversationsTable)
    .where(eq(conversationsTable.status, "replied"));
  const avgConfidence =
    convScores.filter((c) => c.score !== null).length > 0
      ? convScores.reduce((s, c) => s + (c.score ?? 0), 0) / convScores.filter((c) => c.score !== null).length
      : 0.82;

  res.json(
    GetDashboardStatsResponse.parse({
      totalConversations: totalConvRow.count,
      pendingConversations: pendingRow.count,
      openEscalations: openEscRow.count,
      avgConfidenceScore: Math.round(avgConfidence * 100) / 100,
      avgResponseTimeSeconds: Math.round(avgResponseTime * 10) / 10,
      autoReplyRate: Math.round(autoReplyRate),
      totalCustomers: totalCustRow.count,
      highIntentLeads: highIntentRow.count,
      shadowModeReviews: shadowRow.count,
      tokenHealthy,
      tokenExpiringSoon,
      tokenExpired,
    })
  );
});

router.get("/analytics/conversations", async (req, res): Promise<void> => {
  const parsed = GetConversationAnalyticsQueryParams.safeParse(req.query);
  if (!parsed.success) {
    res.status(400).json({ error: parsed.error.message });
    return;
  }

  const period = parsed.data.period ?? "7d";
  const days = period === "7d" ? 7 : period === "30d" ? 30 : 90;
  const startDate = new Date(Date.now() - days * 24 * 60 * 60 * 1000);

  const conversations = await db
    .select({
      createdAt: conversationsTable.createdAt,
      status: conversationsTable.status,
    })
    .from(conversationsTable)
    .where(gte(conversationsTable.createdAt, startDate));

  const byDate: Record<string, { total: number; replied: number; escalated: number }> = {};
  for (let i = 0; i < days; i++) {
    const d = new Date(startDate.getTime() + i * 24 * 60 * 60 * 1000);
    const key = d.toISOString().split("T")[0]!;
    byDate[key] = { total: 0, replied: 0, escalated: 0 };
  }

  for (const conv of conversations) {
    const key = conv.createdAt.toISOString().split("T")[0]!;
    if (byDate[key]) {
      byDate[key].total++;
      if (conv.status === "replied" || conv.status === "resolved") byDate[key].replied++;
      if (conv.status === "escalated") byDate[key].escalated++;
    }
  }

  const result = Object.entries(byDate).map(([date, vals]) => ({ date, ...vals }));
  res.json(GetConversationAnalyticsResponse.parse(result));
});

router.get("/analytics/intents", async (_req, res): Promise<void> => {
  const all = await db
    .select({ intent: conversationsTable.intent })
    .from(conversationsTable);
  const total = all.length || 1;
  const counts: Record<string, number> = {};
  for (const c of all) {
    const k = c.intent ?? "general";
    counts[k] = (counts[k] ?? 0) + 1;
  }
  const result = Object.entries(counts).map(([intent, cnt]) => ({
    intent,
    count: cnt,
    percentage: Math.round((cnt / total) * 100),
  }));
  res.json(GetIntentBreakdownResponse.parse(result));
});

router.get("/analytics/sentiment", async (_req, res): Promise<void> => {
  const all = await db
    .select({ sentiment: conversationsTable.sentiment })
    .from(conversationsTable);
  const total = all.length || 1;
  const counts: Record<string, number> = {};
  for (const c of all) {
    const k = c.sentiment ?? "neutral";
    counts[k] = (counts[k] ?? 0) + 1;
  }
  const result = Object.entries(counts).map(([sentiment, cnt]) => ({
    sentiment,
    count: cnt,
    percentage: Math.round((cnt / total) * 100),
  }));
  res.json(GetSentimentBreakdownResponse.parse(result));
});

router.get("/analytics/response-times", async (_req, res): Promise<void> => {
  const times = await db
    .select({ t: conversationsTable.processingTime })
    .from(conversationsTable)
    .where(eq(conversationsTable.status, "replied"));
  const vals = times.map((r) => r.t ?? 0).sort((a, b) => a - b);
  const avg = vals.length ? vals.reduce((s, v) => s + v, 0) / vals.length : 3.2;
  const p50 = vals[Math.floor(vals.length * 0.5)] ?? 2.8;
  const p95 = vals[Math.floor(vals.length * 0.95)] ?? 8.1;
  const underFive = vals.length ? (vals.filter((v) => v < 5).length / vals.length) * 100 : 92;
  res.json(
    GetResponseTimeStatsResponse.parse({
      avgSeconds: Math.round(avg * 10) / 10,
      p50Seconds: Math.round(p50 * 10) / 10,
      p95Seconds: Math.round(p95 * 10) / 10,
      underFiveSeconds: Math.round(underFive),
    })
  );
});

export default router;
