import { randomUUID } from "node:crypto";
import { execFile } from "node:child_process";
import { promisify } from "node:util";
import { fileURLToPath } from "node:url";
import { test } from "@e2e-dev/web";
import { expect } from "e2e";
import { prepareFreshBrowserSession } from "./auth-session";

const execFileAsync = promisify(execFile);
const repositoryRoot = fileURLToPath(new URL("../..", import.meta.url));

interface SeededPage {
  path: string;
  heading: string;
  resourceId?: string;
}

type FixtureScenario =
  | "accounts-list"
  | "account-detail"
  | "campaign-form"
  | "campaign-detail"
  | "campaigns-list"
  | "contacts-list"
  | "canned-responses-list"
  | "canned-response-detail"
  | "contact-detail"
  | "user-detail"
  | "audit-logs-list"
  | "audit-log-detail"
  | "gowa-servers"
  | "teams"
  | "templates";

async function runFixtureCommand(
  args: string[],
  relatedResourceIds: string[] = [],
): Promise<string> {
  const configPath = process.env.E2E_CONFIG_PATH ?? "config.toml";
  try {
    const { stdout } = await execFileAsync(
      "go",
      ["run", "./tests/e2e-fixtures", ...args, configPath, ...relatedResourceIds],
      {
        cwd: repositoryRoot,
        timeout: 90_000,
        maxBuffer: 256 * 1024,
      },
    );
    return stdout;
  } catch (error) {
    const code =
      typeof error === "object" && error !== null && "code" in error
        ? String(error.code)
        : "unknown";
    // Do not forward subprocess output: database errors can contain connection
    // details, and fixture commands must never expose config or credentials.
    throw new Error(`E2E fixture command failed (exit ${code})`);
  }
}

function requireLocalApp(baseUrl: string): void {
  const { hostname, protocol } = new URL(baseUrl);
  if (
    protocol !== "http:" ||
    !["localhost", "127.0.0.1", "[::1]", "::1"].includes(hostname)
  ) {
    throw new Error("E2E data fixtures are restricted to a local HTTP app");
  }
}

function requireBaseUrl(baseUrl: string | undefined): string {
  if (!baseUrl) {
    throw new Error("The E2E target must define an app URL.");
  }
  return baseUrl;
}

async function seedPage(
  baseUrl: string,
  scenario: FixtureScenario,
): Promise<{ data: SeededPage; runId: string }> {
  requireLocalApp(baseUrl);
  const runId = randomUUID();
  try {
    const stdout = await runFixtureCommand(["seed", scenario, runId]);
    let data: unknown;
    try {
      data = JSON.parse(stdout);
    } catch {
      throw new Error("E2E fixture command returned invalid data");
    }
    if (
      typeof data !== "object" ||
      data === null ||
      !("path" in data) ||
      typeof data.path !== "string" ||
      !data.path.startsWith("/") ||
      !("heading" in data) ||
      typeof data.heading !== "string" ||
      ("resource_id" in data && typeof data.resource_id !== "string")
    ) {
      throw new Error("E2E fixture command returned an invalid page route");
    }
    const resourceId =
      "resource_id" in data && typeof data.resource_id === "string"
        ? data.resource_id
        : undefined;
    return {
      data: { path: data.path, heading: data.heading, resourceId },
      runId,
    };
  } catch (error) {
    // The seed may have committed even if its output could not be parsed.
    await cleanupPage(runId).catch(() => undefined);
    throw error;
  }
}

async function cleanupPage(runId: string, resourceId?: string): Promise<void> {
  await runFixtureCommand(
    ["cleanup", runId],
    resourceId ? [resourceId] : [],
  );
}

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
