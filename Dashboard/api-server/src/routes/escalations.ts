import { Router, type IRouter } from "express";
import { eq, and, SQL, desc } from "drizzle-orm";
import { db, escalationsTable } from "@workspace/db";
import {
  ListEscalationsQueryParams,
  GetEscalationParams,
  ResolveEscalationParams,
  ResolveEscalationBody,
  ListEscalationsResponse,
  GetEscalationResponse,
  ResolveEscalationResponse,
} from "@workspace/api-zod";

const router: IRouter = Router();

router.get("/escalations", async (req, res): Promise<void> => {
  const parsed = ListEscalationsQueryParams.safeParse(req.query);
  if (!parsed.success) {
    res.status(400).json({ error: parsed.error.message });
    return;
  }
  const { status, priority, limit, offset } = parsed.data;

  const conditions: SQL[] = [];
  if (status && status !== "all") {
    conditions.push(eq(escalationsTable.status, status as "open" | "resolved"));
  }
  if (priority) {
    conditions.push(eq(escalationsTable.priority, priority as "critical" | "high" | "medium" | "low"));
  }

  const whereClause = conditions.length > 0 ? and(...conditions) : undefined;
  const escalations = await db
    .select()
    .from(escalationsTable)
    .where(whereClause)
    .orderBy(desc(escalationsTable.createdAt))
    .limit(limit ?? 50)
    .offset(offset ?? 0);

  const total = await db.$count(escalationsTable, whereClause);

  res.json(
    ListEscalationsResponse.parse({
      data: escalations,
      total,
      limit: limit ?? 50,
      offset: offset ?? 0,
    })
  );
});

router.get("/escalations/:escalationId", async (req, res): Promise<void> => {
  const params = GetEscalationParams.safeParse(req.params);
  if (!params.success) {
    res.status(400).json({ error: params.error.message });
    return;
  }
  const [esc] = await db
    .select()
    .from(escalationsTable)
    .where(eq(escalationsTable.id, params.data.escalationId));
  if (!esc) {
    res.status(404).json({ error: "Escalation not found" });
    return;
  }
  res.json(GetEscalationResponse.parse(esc));
});

router.post("/escalations/:escalationId/resolve", async (req, res): Promise<void> => {
  const params = ResolveEscalationParams.safeParse(req.params);
  if (!params.success) {
    res.status(400).json({ error: params.error.message });
    return;
  }
  const body = ResolveEscalationBody.safeParse(req.body);
  if (!body.success) {
    res.status(400).json({ error: body.error.message });
    return;
  }
  const [esc] = await db
    .update(escalationsTable)
    .set({
      status: "resolved",
      adminNotes: body.data.adminNotes,
      resolvedBy: body.data.resolvedBy,
      resolvedAt: new Date(),
    })
    .where(eq(escalationsTable.id, params.data.escalationId))
    .returning();
  if (!esc) {
    res.status(404).json({ error: "Escalation not found" });
    return;
  }
  res.json(ResolveEscalationResponse.parse(esc));
});

export default router;
