import * as Sentry from "@sentry/nextjs";

const SENTRY_DSN = process.env.NEXT_PUBLIC_SENTRY_DSN;
const ENABLE_REPLAY = process.env.NEXT_PUBLIC_SENTRY_REPLAY === "true";

if (SENTRY_DSN) {
  Sentry.init({
    dsn: SENTRY_DSN,
    environment: process.env.NODE_ENV,
    tracesSampleRate: 0.2,
    replaysSessionSampleRate: ENABLE_REPLAY ? 0.01 : 0,
    replaysOnErrorSampleRate: ENABLE_REPLAY ? 0.1 : 0,
    integrations: ENABLE_REPLAY
      ? [
          Sentry.replayIntegration({
            maskAllText: true,
            blockAllMedia: true,
          }),
        ]
      : [],
  });
}
