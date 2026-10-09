import { randomUUID } from "node:crypto";
import { execFile } from "node:child_process";
import { promisify } from "node:util";
import { fileURLToPath } from "node:url";

const execFileAsync = promisify(execFile);
const repositoryRoot = fileURLToPath(new URL("../..", import.meta.url));

export interface SeededPage {
  path: string;
  heading: string;
  resourceId?: string;
}

export type FixtureScenario =
  | "accounts-list"
  | "account-detail"
  | "campaign-form"
  | "campaign-detail"
  | "campaigns-list"
  | "chat-conversation"
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

export function requireBaseUrl(baseUrl: string | undefined): string {
  if (!baseUrl) {
    throw new Error("The E2E target must define an app URL.");
  }
  return baseUrl;
}

export async function seedPage(
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

export async function cleanupPage(runId: string, resourceId?: string): Promise<void> {
  await runFixtureCommand(
    ["cleanup", runId],
    resourceId ? [resourceId] : [],
  );
}
