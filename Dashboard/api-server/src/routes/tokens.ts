import { Router, type IRouter } from "express";
import { db, pagesTable } from "@workspace/db";
import { ListTokenStatusesResponse } from "@workspace/api-zod";

const router: IRouter = Router();

router.get("/tokens", async (_req, res): Promise<void> => {
  const pages = await db.select().from(pagesTable);
  const statuses = pages.map((page) => {
    const daysUntilExpiry = page.tokenExpiresAt
      ? Math.ceil((page.tokenExpiresAt.getTime() - Date.now()) / (1000 * 60 * 60 * 24))
      : null;
    return {
      pageId: page.id,
      pageName: page.name,
      platform: page.platform,
      status: page.tokenStatus ?? "valid",
      expiresAt: page.tokenExpiresAt,
      daysUntilExpiry,
      lastRefreshedAt: page.tokenLastRefreshedAt,
      lastError: page.tokenLastError,
    };
  });
  res.json(ListTokenStatusesResponse.parse(statuses));
});

export default router;
