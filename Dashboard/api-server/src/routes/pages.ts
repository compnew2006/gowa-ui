import { Router, type IRouter } from "express";
import { eq } from "drizzle-orm";
import { db, pagesTable } from "@workspace/db";
import {
  CreatePageBody,
  UpdatePageBody,
  GetPageParams,
  UpdatePageParams,
  DeletePageParams,
  GetPageTokenStatusParams,
  RefreshPageTokenParams,
  ListPagesResponse,
  GetPageResponse,
  UpdatePageResponse,
  GetPageTokenStatusResponse,
  RefreshPageTokenResponse,
} from "@workspace/api-zod";

const router: IRouter = Router();

router.get("/pages", async (req, res): Promise<void> => {
  const pages = await db.select().from(pagesTable).orderBy(pagesTable.createdAt);
  res.json(ListPagesResponse.parse(pages));
});

router.post("/pages", async (req, res): Promise<void> => {
  const parsed = CreatePageBody.safeParse(req.body);
  if (!parsed.success) {
    res.status(400).json({ error: parsed.error.message });
    return;
  }
  const { accessToken, ...rest } = parsed.data;
  const [page] = await db
    .insert(pagesTable)
    .values({
      ...rest,
      accessTokenEncrypted: accessToken,
      trackingStartDate: new Date(),
      tokenStatus: "valid",
      tokenLastRefreshedAt: new Date(),
      tokenExpiresAt: new Date(Date.now() + 60 * 24 * 60 * 60 * 1000),
    })
    .returning();
  res.status(201).json(GetPageResponse.parse(page));
});

router.get("/pages/:pageId", async (req, res): Promise<void> => {
  const params = GetPageParams.safeParse(req.params);
  if (!params.success) {
    res.status(400).json({ error: params.error.message });
    return;
  }
  const [page] = await db.select().from(pagesTable).where(eq(pagesTable.id, params.data.pageId));
  if (!page) {
    res.status(404).json({ error: "Page not found" });
    return;
  }
  res.json(GetPageResponse.parse(page));
});

router.patch("/pages/:pageId", async (req, res): Promise<void> => {
  const params = UpdatePageParams.safeParse(req.params);
  if (!params.success) {
    res.status(400).json({ error: params.error.message });
    return;
  }
  const parsed = UpdatePageBody.safeParse(req.body);
  if (!parsed.success) {
    res.status(400).json({ error: parsed.error.message });
    return;
  }
  const [page] = await db
    .update(pagesTable)
    .set(parsed.data)
    .where(eq(pagesTable.id, params.data.pageId))
    .returning();
  if (!page) {
    res.status(404).json({ error: "Page not found" });
    return;
  }
  res.json(UpdatePageResponse.parse(page));
});

router.delete("/pages/:pageId", async (req, res): Promise<void> => {
  const params = DeletePageParams.safeParse(req.params);
  if (!params.success) {
    res.status(400).json({ error: params.error.message });
    return;
  }
  const [page] = await db.delete(pagesTable).where(eq(pagesTable.id, params.data.pageId)).returning();
  if (!page) {
    res.status(404).json({ error: "Page not found" });
    return;
  }
  res.sendStatus(204);
});

router.get("/pages/:pageId/token", async (req, res): Promise<void> => {
  const params = GetPageTokenStatusParams.safeParse(req.params);
  if (!params.success) {
    res.status(400).json({ error: params.error.message });
    return;
  }
  const [page] = await db.select().from(pagesTable).where(eq(pagesTable.id, params.data.pageId));
  if (!page) {
    res.status(404).json({ error: "Page not found" });
    return;
  }
  const daysUntilExpiry = page.tokenExpiresAt
    ? Math.ceil((page.tokenExpiresAt.getTime() - Date.now()) / (1000 * 60 * 60 * 24))
    : null;
  res.json(
    GetPageTokenStatusResponse.parse({
      pageId: page.id,
      pageName: page.name,
      platform: page.platform,
      status: page.tokenStatus ?? "valid",
      expiresAt: page.tokenExpiresAt,
      daysUntilExpiry,
      lastRefreshedAt: page.tokenLastRefreshedAt,
      lastError: page.tokenLastError,
    })
  );
});

router.post("/pages/:pageId/token", async (req, res): Promise<void> => {
  const params = RefreshPageTokenParams.safeParse(req.params);
  if (!params.success) {
    res.status(400).json({ error: params.error.message });
    return;
  }
  const [page] = await db
    .update(pagesTable)
    .set({
      tokenStatus: "valid",
      tokenLastRefreshedAt: new Date(),
      tokenExpiresAt: new Date(Date.now() + 60 * 24 * 60 * 60 * 1000),
      tokenLastError: null,
    })
    .where(eq(pagesTable.id, params.data.pageId))
    .returning();
  if (!page) {
    res.status(404).json({ error: "Page not found" });
    return;
  }
  const daysUntilExpiry = page.tokenExpiresAt
    ? Math.ceil((page.tokenExpiresAt.getTime() - Date.now()) / (1000 * 60 * 60 * 24))
    : null;
  res.json(
    RefreshPageTokenResponse.parse({
      pageId: page.id,
      pageName: page.name,
      platform: page.platform,
      status: page.tokenStatus ?? "valid",
      expiresAt: page.tokenExpiresAt,
      daysUntilExpiry,
      lastRefreshedAt: page.tokenLastRefreshedAt,
      lastError: page.tokenLastError,
    })
  );
});

export default router;
