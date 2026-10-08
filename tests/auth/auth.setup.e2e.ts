import { test } from "@e2e-dev/web";
import { credentials, expect } from "e2e";
import type { JsonValue } from "e2e";
import {
  getCredentialPassword,
  loginForE2E,
  type ApiSession,
} from "../helpers/auth-session";

interface RolePayload {
  data?: {
    roles?: Array<{ id: string; name: string }>;
  };
}

interface UsersPayload {
  data?: {
    users?: Array<{ email: string; role_id?: string }>;
  };
}

async function ensureRegularUser(
  baseUrl: string,
  session: ApiSession,
): Promise<void> {
  const userEmail = credentials.user("regular-user").username;
  const headers = { cookie: session.cookieHeader };
  const [rolesResponse, usersResponse] = await Promise.all([
    fetch(new URL("/api/roles", baseUrl), { headers }),
    fetch(
      new URL(`/api/users?search=${encodeURIComponent(userEmail)}`, baseUrl),
      { headers },
    ),
  ]);

  if (!rolesResponse.ok || !usersResponse.ok) {
    throw new Error(
      `Could not verify E2E user fixtures (roles HTTP ${rolesResponse.status}, users HTTP ${usersResponse.status}).`,
    );
  }

  const roles = (await rolesResponse.json()) as RolePayload;
  const users = (await usersResponse.json()) as UsersPayload;
  const agentRole = roles.data?.roles?.find((role) => role.name === "agent");
  const existingUser = users.data?.users?.find(
    (user) => user.email === userEmail,
  );

  if (!agentRole) {
    throw new Error(
      "The E2E regular-user fixture requires the system agent role.",
    );
  }

  if (existingUser) {
    if (existingUser.role_id !== agentRole.id) {
      throw new Error(
        "The configured regular E2E account must have the agent role; choose another E2E_USER_EMAIL or correct its test role.",
      );
    }
    return;
  }

  const createResponse = await fetch(new URL("/api/users", baseUrl), {
    method: "POST",
    headers: {
      ...headers,
      "content-type": "application/json",
      "x-csrf-token": session.csrfToken,
    },
    body: JSON.stringify({
      email: userEmail,
      password: getCredentialPassword("user"),
      full_name: "E2E Regular User",
      role_id: agentRole.id,
      is_active: true,
    }),
  });

  if (!createResponse.ok) {
    throw new Error(
      `Could not provision the isolated regular E2E account (HTTP ${createResponse.status}).`,
    );
  }
}

test.setup(
  "authenticate as super-admin and regular user",
  { sessions: ["super-admin", "user"] },
  async ({ app, browser, screen, session }) => {
    const baseUrl = app.baseUrl;
    if (!baseUrl) {
      throw new Error(
        "The E2E target must define an app URL to create authenticated sessions.",
      );
    }

    await app.open("/login");

    const superAdmin = await loginForE2E(baseUrl, "super-admin");
    await browser.setCookies(superAdmin.cookies);
    await browser.evaluate((user) => {
      window.localStorage.setItem("user", JSON.stringify(user));
      return null;
    }, superAdmin.user);
    await app.open("/");

    await expect(browser).not.toHaveURL(/\/login(?:$|[?#])/);
    await expect(screen.getByRole("heading", "Dashboard")).toBeVisible({
      timeout: 20_000,
    });
    await session.save("super-admin");

    await ensureRegularUser(baseUrl, superAdmin);

    const user = await loginForE2E(baseUrl, "user");
    await browser.setCookies(user.cookies);
    await browser.evaluate((userData) => {
      window.localStorage.setItem("user", JSON.stringify(userData));
      return null;
    }, user.user);
    await app.open("/");
    await expect(browser).not.toHaveURL(/\/login(?:$|[?#])/);
    await expect(browser).toHaveURL("/chat");
    await expect(
      screen.getByRole("heading", "Select a conversation"),
    ).toBeVisible({
      timeout: 20_000,
    });
    await session.save("user");
  },
);
