import { test } from "@e2e-dev/web";
import { expect } from "e2e";
import { prepareFreshBrowserSession } from "./auth-session";
import {
  cleanupPage,
  requireBaseUrl,
  seedPage,
  type FixtureScenario,
  type SeededPage,
} from "./fixtures";

interface PageExploration {
  name: string;
  path: string;
  heading: string;
  instructions: string;
  regularUserInstructions?: string;
  permission?: string;
  fixture?: FixtureScenario;
}

const regularUserPermissions = new Set([
  "accounts",
  "analytics.agents",
  "canned_responses",
  "chat",
  "tags",
]);

const sessions = [
  { name: "super-admin", label: "super-admin", canInspectRestrictedPage: true },
  { name: "user", label: "regular user", canInspectRestrictedPage: false },
] as const;

export function definePageExploration({
  name,
  path,
  heading,
  instructions,
  regularUserInstructions,
  permission,
  fixture,
}: PageExploration): void {
  const pageTest = test.extend<{ seededPage: SeededPage | null }>({
    seededPage: async ({ app }, use) => {
      if (!fixture) {
        await use(null);
        return;
      }

      const seeded = await seedPage(requireBaseUrl(app.baseUrl), fixture);
      try {
        await use(seeded.data);
      } finally {
        await cleanupPage(seeded.runId, seeded.data.resourceId);
      }
    },
  });

  for (const session of sessions) {
    pageTest(
      `${name} (${session.label})`,
      {
        session: session.name,
        tags: ["page", "authenticated", session.name],
      },
      async ({ app, browser, screen, agent, seededPage }) => {
        const targetPath = seededPage?.path ?? path;
        const targetHeading = seededPage?.heading ?? heading;
        const userIsDenied =
          !session.canInspectRestrictedPage &&
          permission !== undefined &&
          !regularUserPermissions.has(permission);

        // A saved E2E session's rotating refresh token is single-use. Log in
        // for each attempt so sequential tests never restore a consumed token.
        await prepareFreshBrowserSession(
          app,
          browser,
          session.name === "user" ? "user" : "super-admin",
        );

        if (userIsDenied) {
          await app.open(targetPath);
          const chatHeading = screen.getByRole(
            "heading",
            "Select a conversation",
          );
          await expect(browser).toHaveURL("/chat");
          await expect(chatHeading).toBeVisible({ timeout: 20_000 });

          await agent.act(
            `As a regular user, verify that opening ${targetPath} is blocked and routes to the first page available to this role. Confirm the chat page is shown and do not try to bypass access controls.`,
          );

          await expect(browser).toHaveURL("/chat");
          await expect(chatHeading).toBeVisible();
          return;
        }

        await app.open(targetPath);
        const pageHeading = screen
          .getByRole("main")
          .getByRole("heading", targetHeading)
          .last();
        await expect(pageHeading).toBeVisible({ timeout: 20_000 });
        await expect(browser).toHaveURL(targetPath);

        await agent.act(
          `${session.label === "regular user" && regularUserInstructions ? regularUserInstructions : instructions} Explore this page as ${session.label}. Stay on ${targetPath}. Use safe, reversible interactions only. Do not save or submit valid data, change persistent settings, or trigger an external action.`,
        );

        await expect(pageHeading).toBeVisible();
        await expect(browser).toHaveURL(targetPath);
      },
    );
  }
}
