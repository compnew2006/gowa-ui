import { Router, type IRouter } from "express";
import { eq, and, SQL, desc } from "drizzle-orm";
import { db, conversationsTable } from "@workspace/db";
import {
  ListConversationsQueryParams,
  GetConversationParams,
  ManualReplyParams,
  ManualReplyBody,
  ResolveConversationParams,
  ListConversationsResponse,
  GetConversationResponse,
  ManualReplyResponse,
  ResolveConversationResponse,
} from "@workspace/api-zod";

const router: IRouter = Router();

router.get("/conversations", async (req, res): Promise<void> => {
  const parsed = ListConversationsQueryParams.safeParse(req.query);
  if (!parsed.success) {
    res.status(400).json({ error: parsed.error.message });
    return;
  }
  const { status, pageId, intent, sentiment, limit, offset } = parsed.data;

  const conditions: SQL[] = [];
  if (status && status !== "all") {
    conditions.push(eq(conversationsTable.status, status as "pending" | "replied" | "escalated" | "resolved"));
  }
  if (pageId) {
    conditions.push(eq(conversationsTable.pageId, pageId));
  }
  if (intent) {
    conditions.push(eq(conversationsTable.intent, intent as "price_inquiry" | "purchase" | "complaint" | "refund" | "general"));
  }
  if (sentiment) {
    conditions.push(eq(conversationsTable.sentiment, sentiment as "positive" | "neutral" | "negative" | "angry" | "urgent"));
  }

  const whereClause = conditions.length > 0 ? and(...conditions) : undefined;
  const conversations = await db
    .select()
    .from(conversationsTable)
    .where(whereClause)
    .orderBy(desc(conversationsTable.createdAt))
    .limit(limit ?? 50)
    .offset(offset ?? 0);

  const total = await db.$count(conversationsTable, whereClause);

  res.json(
    ListConversationsResponse.parse({
      data: conversations,
      total,
      limit: limit ?? 50,
      offset: offset ?? 0,
    })
  );
});

router.get("/conversations/:conversationId", async (req, res): Promise<void> => {
  const params = GetConversationParams.safeParse(req.params);
  if (!params.success) {
    res.status(400).json({ error: params.error.message });
    return;
  }
  const [conv] = await db
    .select()
    .from(conversationsTable)
    .where(eq(conversationsTable.id, params.data.conversationId));
  if (!conv) {
    res.status(404).json({ error: "Conversation not found" });
    return;
  }
  res.json(GetConversationResponse.parse(conv));
});

router.post("/conversations/:conversationId/reply", async (req, res): Promise<void> => {
  const params = ManualReplyParams.safeParse(req.params);
  if (!params.success) {
    res.status(400).json({ error: params.error.message });
    return;
  }
  const body = ManualReplyBody.safeParse(req.body);
  if (!body.success) {
    res.status(400).json({ error: body.error.message });
    return;
  }
  const [conv] = await db
    .update(conversationsTable)
    .set({
      adminReply: body.data.reply,
      status: "replied",
      repliedAt: new Date(),
    })
    .where(eq(conversationsTable.id, params.data.conversationId))
    .returning();
  if (!conv) {
    res.status(404).json({ error: "Conversation not found" });
    return;
  }
  res.json(ManualReplyResponse.parse(conv));
});

router.post("/conversations/:conversationId/resolve", async (req, res): Promise<void> => {
  const params = ResolveConversationParams.safeParse(req.params);
  if (!params.success) {
    res.status(400).json({ error: params.error.message });
    return;
  }
  const [conv] = await db
    .update(conversationsTable)
    .set({ status: "resolved" })
    .where(eq(conversationsTable.id, params.data.conversationId))
    .returning();
  if (!conv) {
    res.status(404).json({ error: "Conversation not found" });
    return;
  }
  res.json(ResolveConversationResponse.parse(conv));
});

export default router;
