import type { Browser, Cookie } from "@e2e-dev/web";
import { credentials } from "e2e";
import type { JsonValue } from "e2e";

interface LoginPayload {
  data?: {
    user?: { [key: string]: JsonValue };
  };
}

export interface ApiSession {
  cookies: Cookie[];
  cookieHeader: string;
  csrfToken: string;
  user: { [key: string]: JsonValue };
}

export type E2ESessionName = "super-admin" | "user";

interface AppSessionController {
  baseUrl?: string;
  clearState(): Promise<void>;
  open(path: string): Promise<void>;
}

const credentialForSession = {
  "super-admin": {
    name: "super-admin",
    passwordEnvs: ["E2E_USER_SUPER_ADMIN_PASSWORD", "E2E_SUPER_ADMIN_PASSWORD"],
    label: "super-admin",
  },
  user: {
    name: "regular-user",
    passwordEnvs: ["E2E_USER_REGULAR_USER_PASSWORD", "E2E_USER_PASSWORD"],
    label: "regular user",
  },
} as const;

export function getCredentialPassword(sessionName: E2ESessionName): string {
  const credential = credentialForSession[sessionName];
  for (const envName of credential.passwordEnvs) {
    const password = process.env[envName];
    if (password) return password;
  }

  throw new Error(
    `Set ${credential.passwordEnvs[0]} to authenticate the ${credential.label} E2E account.`,
  );
}

export async function loginForE2E(
  baseUrl: string,
  sessionName: E2ESessionName,
): Promise<ApiSession> {
  const config = credentialForSession[sessionName];
  const email = credentials.user(config.name).username;
  const response = await fetch(new URL("/api/auth/login", baseUrl), {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({
      email,
      password: getCredentialPassword(sessionName),
    }),
  });

  if (!response.ok) {
    throw new Error(
      `Could not authenticate the ${config.label} E2E account (HTTP ${response.status}). Configure its E2E credentials and ensure the account exists.`,
    );
  }

  const payload = (await response.json()) as LoginPayload;
  const user = payload.data?.user;
  if (!user) {
    throw new Error(
      `The ${config.label} login response did not include a user session.`,
    );
  }

  const cookies = response.headers.getSetCookie().map((header): Cookie => {
    const [cookiePair, ...attributes] = header.split(";");
    const separator = cookiePair.indexOf("=");
    const name = cookiePair.slice(0, separator).trim();
    const value = cookiePair.slice(separator + 1).trim();
    const path = attributes
      .map((attribute) => attribute.trim())
      .find((attribute) => attribute.toLowerCase().startsWith("path="))
      ?.slice("path=".length);
    const sameSiteAttribute = attributes
      .map((attribute) => attribute.trim())
      .find((attribute) => attribute.toLowerCase().startsWith("samesite="))
      ?.slice("samesite=".length)
      .toLowerCase();

    return {
      name,
      value,
      domain: new URL(baseUrl).hostname,
      ...(path ? { path } : {}),
      httpOnly: attributes.some(
        (attribute) => attribute.trim().toLowerCase() === "httponly",
      ),
      secure: attributes.some(
        (attribute) => attribute.trim().toLowerCase() === "secure",
      ),
      ...(sameSiteAttribute
        ? {
            sameSite: (sameSiteAttribute[0].toUpperCase() +
              sameSiteAttribute.slice(1)) as Cookie["sameSite"],
          }
        : {}),
    };
  });
  const csrfToken = cookies.find((cookie) => cookie.name === "whm_csrf")?.value;

  if (cookies.length === 0 || !csrfToken) {
    throw new Error(
      `The ${config.label} login response did not include the expected auth cookies.`,
    );
  }

  return {
    cookies,
    cookieHeader: cookies
      .map(({ name, value }) => `${name}=${value}`)
      .join("; "),
    csrfToken,
    user,
  };
}

export async function installFreshBrowserSession(
  browser: Browser,
  baseUrl: string,
  sessionName: E2ESessionName,
): Promise<void> {
  const session = await loginForE2E(baseUrl, sessionName);
  await browser.setCookies(session.cookies);
  await browser.evaluate((user) => {
    window.localStorage.setItem("user", JSON.stringify(user));
    return null;
  }, session.user);
}

export async function prepareFreshBrowserSession(
  app: AppSessionController,
  browser: Browser,
  sessionName: E2ESessionName,
): Promise<void> {
  if (!app.baseUrl) {
    throw new Error("The E2E target must define an app URL.");
  }

  await app.clearState();
  await app.open("/login");
  await installFreshBrowserSession(browser, app.baseUrl, sessionName);
}
