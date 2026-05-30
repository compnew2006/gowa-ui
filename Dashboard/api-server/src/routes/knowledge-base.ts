import { Router, type IRouter } from "express";
import { eq, ilike, and, SQL, desc } from "drizzle-orm";
import { db, knowledgeBaseTable } from "@workspace/db";
import {
  ListKnowledgeBaseEntriesQueryParams,
  CreateKnowledgeBaseEntryBody,
  GetKnowledgeBaseEntryParams,
  UpdateKnowledgeBaseEntryParams,
  UpdateKnowledgeBaseEntryBody,
  DeleteKnowledgeBaseEntryParams,
  ListKnowledgeBaseEntriesResponse,
  GetKnowledgeBaseEntryResponse,
  UpdateKnowledgeBaseEntryResponse,
  CreateKnowledgeBaseEntryBody as CreateKBBody,
} from "@workspace/api-zod";

const router: IRouter = Router();

router.get("/knowledge-base", async (req, res): Promise<void> => {
  const parsed = ListKnowledgeBaseEntriesQueryParams.safeParse(req.query);
  if (!parsed.success) {
    res.status(400).json({ error: parsed.error.message });
    return;
  }
  const { search, category, limit, offset } = parsed.data;

  const conditions: SQL[] = [];
  if (search) {
    conditions.push(ilike(knowledgeBaseTable.question, `%${search}%`));
  }
  if (category) {
    conditions.push(eq(knowledgeBaseTable.category, category));
  }

  const whereClause = conditions.length > 0 ? and(...conditions) : undefined;
  const entries = await db
    .select()
    .from(knowledgeBaseTable)
    .where(whereClause)
    .orderBy(desc(knowledgeBaseTable.updatedAt))
    .limit(limit ?? 50)
    .offset(offset ?? 0);

  const total = await db.$count(knowledgeBaseTable, whereClause);

  res.json(
    ListKnowledgeBaseEntriesResponse.parse({
      data: entries,
      total,
      limit: limit ?? 50,
      offset: offset ?? 0,
    })
  );
});

router.post("/knowledge-base", async (req, res): Promise<void> => {
  const parsed = CreateKnowledgeBaseEntryBody.safeParse(req.body);
  if (!parsed.success) {
    res.status(400).json({ error: parsed.error.message });
    return;
  }
  const [entry] = await db.insert(knowledgeBaseTable).values(parsed.data).returning();
  res.status(201).json(GetKnowledgeBaseEntryResponse.parse(entry));
});

router.get("/knowledge-base/:entryId", async (req, res): Promise<void> => {
  const params = GetKnowledgeBaseEntryParams.safeParse(req.params);
  if (!params.success) {
    res.status(400).json({ error: params.error.message });
    return;
  }
  const [entry] = await db
    .select()
    .from(knowledgeBaseTable)
    .where(eq(knowledgeBaseTable.id, params.data.entryId));
  if (!entry) {
    res.status(404).json({ error: "Entry not found" });
    return;
  }
  res.json(GetKnowledgeBaseEntryResponse.parse(entry));
});

router.patch("/knowledge-base/:entryId", async (req, res): Promise<void> => {
  const params = UpdateKnowledgeBaseEntryParams.safeParse(req.params);
  if (!params.success) {
    res.status(400).json({ error: params.error.message });
    return;
  }
  const body = UpdateKnowledgeBaseEntryBody.safeParse(req.body);
  if (!body.success) {
    res.status(400).json({ error: body.error.message });
    return;
  }
  const [entry] = await db
    .update(knowledgeBaseTable)
    .set(body.data)
    .where(eq(knowledgeBaseTable.id, params.data.entryId))
    .returning();
  if (!entry) {
    res.status(404).json({ error: "Entry not found" });
    return;
  }
  res.json(UpdateKnowledgeBaseEntryResponse.parse(entry));
});

router.delete("/knowledge-base/:entryId", async (req, res): Promise<void> => {
  const params = DeleteKnowledgeBaseEntryParams.safeParse(req.params);
  if (!params.success) {
    res.status(400).json({ error: params.error.message });
    return;
  }
  const [entry] = await db
    .delete(knowledgeBaseTable)
    .where(eq(knowledgeBaseTable.id, params.data.entryId))
    .returning();
  if (!entry) {
    res.status(404).json({ error: "Entry not found" });
    return;
  }
  res.sendStatus(204);
});

export default router;
