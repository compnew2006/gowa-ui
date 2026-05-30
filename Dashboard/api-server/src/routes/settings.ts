import { Router, type IRouter } from "express";
import { eq } from "drizzle-orm";
import { db, settingsTable } from "@workspace/db";
import { UpdateSettingsBody, GetSettingsResponse, UpdateSettingsResponse } from "@workspace/api-zod";

const router: IRouter = Router();

async function ensureSettings() {
  const existing = await db.select().from(settingsTable).limit(1);
  if (existing.length === 0) {
    const [created] = await db.insert(settingsTable).values({}).returning();
    return created!;
  }
  return existing[0]!;
}

router.get("/settings", async (_req, res): Promise<void> => {
  const settings = await ensureSettings();
  res.json(GetSettingsResponse.parse(settings));
});

router.patch("/settings", async (req, res): Promise<void> => {
  const parsed = UpdateSettingsBody.safeParse(req.body);
  if (!parsed.success) {
    res.status(400).json({ error: parsed.error.message });
    return;
  }
  const existing = await ensureSettings();
  const [updated] = await db
    .update(settingsTable)
    .set(parsed.data)
    .where(eq(settingsTable.id, existing.id))
    .returning();
  res.json(UpdateSettingsResponse.parse(updated));
});

export default router;
