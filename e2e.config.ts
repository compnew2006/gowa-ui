import type { E2EConfig } from "e2e";
import { web } from "@e2e-dev/web";
import { chatgpt } from "e2e/oauth/chatgpt";

export default {
  // Your ChatGPT subscription serves the model; sign in once with `e2e login openai`, `e2e models openai` lists the ids.
  agents: {
    default: {
      model: chatgpt("gpt-6-luna"),
      providerOptions: { openai: { reasoningEffort: "high" } },
      system: [
        "You are a careful QA agent. Verify every outcome on screen.",
        "Prefer reversible, read-only interactions.",
        "Never create, update, or delete records; send messages; start or stop campaigns; connect devices; issue keys; or save settings.",
        "Do not read, repeat, or expose password and secret values.",
      ].join(" "),
    },
  },
  secrets: {
    "invalid-login-password": () =>
      process.env.E2E_SECRET_INVALID_LOGIN_PASSWORD ??
      "not-a-valid-e2e-password",
  },
  credentials: {
    "super-admin": {
      username: process.env.E2E_SUPER_ADMIN_EMAIL ?? "admin@admin.com",
      password: () => {
        const password =
          process.env.E2E_USER_SUPER_ADMIN_PASSWORD ??
          process.env.E2E_SUPER_ADMIN_PASSWORD;
        if (!password) throw new Error("Set E2E_USER_SUPER_ADMIN_PASSWORD");
        return password;
      },
    },
    "regular-user": {
      username: process.env.E2E_USER_EMAIL ?? "e2e-agent@test.com",
      password: () => {
        const password =
          process.env.E2E_USER_REGULAR_USER_PASSWORD ??
          process.env.E2E_USER_PASSWORD;
        if (!password) throw new Error("Set E2E_USER_REGULAR_USER_PASSWORD");
        return password;
      },
    },
  },
  targets: [
    {
      engine: web(),
      app: {
        url: process.env.APP_URL ?? "http://localhost:3000",
        // Or let the runner start the dev server:
        // command: { executable: 'npm', args: ['run', 'dev'] },
      },
    },
  ],
} satisfies E2EConfig;
