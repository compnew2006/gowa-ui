import { Router, type IRouter } from "express";
import { db, settingsTable } from "@workspace/db";
import { VerifyWebhookQueryParams, VerifyWebhookResponse, ReceiveWebhookResponse } from "@workspace/api-zod";
import { logger } from "../lib/logger";

const router: IRouter = Router();

router.get("/webhook/meta", async (req, res): Promise<void> => {
  const parsed = VerifyWebhookQueryParams.safeParse(req.query);
  if (!parsed.success) {
    res.status(400).json({ error: "Invalid query params" });
    return;
  }

  const settings = await db.select().from(settingsTable).limit(1);
  const verifyToken = settings[0]?.webhookVerifyToken ?? "verify_token_change_me";

  if (
    parsed.data["hub.mode"] === "subscribe" &&
    parsed.data["hub.verify_token"] === verifyToken
  ) {
    res.json(VerifyWebhookResponse.parse({ challenge: parsed.data["hub.challenge"] ?? "" }));
    return;
  }

  res.status(403).json({ error: "Forbidden" });
});

router.post("/webhook/meta", async (req, res): Promise<void> => {
  logger.info({ body: req.body }, "Received Meta webhook event");
  res.json(ReceiveWebhookResponse.parse({ received: true }));
});

export default router;
