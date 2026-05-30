import { Router, type IRouter } from "express";
import healthRouter from "./health";
import pagesRouter from "./pages";
import conversationsRouter from "./conversations";
import customersRouter from "./customers";
import escalationsRouter from "./escalations";
import knowledgeBaseRouter from "./knowledge-base";
import analyticsRouter from "./analytics";
import tokensRouter from "./tokens";
import settingsRouter from "./settings";
import webhookRouter from "./webhook";

const router: IRouter = Router();

router.use(healthRouter);
router.use(pagesRouter);
router.use(conversationsRouter);
router.use(customersRouter);
router.use(escalationsRouter);
router.use(knowledgeBaseRouter);
router.use(analyticsRouter);
router.use(tokensRouter);
router.use(settingsRouter);
router.use(webhookRouter);

export default router;
