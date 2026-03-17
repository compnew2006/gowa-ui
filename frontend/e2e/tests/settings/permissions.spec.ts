import {
  test,
  expect,
  Page,
  Browser,
  BrowserContext,
  APIRequestContext,
  request,
} from "@playwright/test";
import {
  loginAsAdmin,
  ApiHelper,
  generateUniqueName,
  generateUniqueEmail,
} from "../../helpers";
import { TablePage, DialogPage } from "../../pages";

// Helper to get visible sidebar menu items
async function getSidebarMenuItems(page: Page): Promise<string[]> {
  const items: string[] = [];
  const navLinks = page.locator(
    'aside nav a, aside nav button[class*="justify-start"]',
  );
  const count = await navLinks.count();
  for (let i = 0; i < count; i++) {
    const text = await navLinks.nth(i).textContent();
    if (text && text.trim()) items.push(text.trim().toLowerCase());
  }
  return items;
}

// Helper to login with specific credentials
async function loginWithCredentials(
  page: Page,
  email: string,
  password: string,
) {
  await page.goto("/login");
  await page.locator('input[name="email"], input[type="email"]').fill(email);
  await page
    .locator('input[name="password"], input[type="password"]')
    .fill(password);
  await page.locator('button[type="submit"]').click();
  await page.waitForURL((url) => !url.pathname.includes("/login"), {
    timeout: 10000,
  });
}

async function createLoggedInSession(
  browser: Browser,
  email: string,
  password: string,
): Promise<{ context: BrowserContext; page: Page }> {
  const context = await browser.newContext();
  const page = await context.newPage();
  await loginWithCredentials(page, email, password);
  return { context, page };
}

let adminApi: ApiHelper;
let adminApiContext: APIRequestContext;

test.beforeAll(async () => {
  adminApiContext = await request.newContext({
    baseURL: process.env.BASE_URL || "http://localhost:8080",
  });
  adminApi = new ApiHelper(adminApiContext);
  await adminApi.loginAsAdmin();
});

test.afterAll(async () => {
  await adminApiContext?.dispose();
});

test.describe("Custom Role with Limited Permissions", () => {
  test.describe.configure({ mode: "serial" });

  const roleName = generateUniqueName("E2E Limited Role");
  const userEmail = generateUniqueEmail("e2e-limited");
  const userPassword = "Password123!";

  let api: ApiHelper;
  let roleId: string;
  let userId: string;
  let context: BrowserContext;
  let page: Page;
  let initialLandingUrl = "";

  test.beforeAll(async ({ browser }) => {
    api = adminApi;

    // Create role with only chat read permission (very limited)
    const permissions = await api.findPermissionKeys([
      { resource: "chat", action: "read" },
    ]);

    const role = await api.createRole({
      name: roleName,
      description: "E2E test role with limited permissions",
      permissions,
    });
    roleId = role.id;

    // Create user with the custom role
    const user = await api.createUser({
      email: userEmail,
      password: userPassword,
      full_name: "E2E Limited User",
      role_id: roleId,
    });
    userId = user.id;

    ({ context, page } = await createLoggedInSession(
      browser,
      userEmail,
      userPassword,
    ));
    initialLandingUrl = page.url();
  });

  test.afterAll(async () => {
    await context?.close().catch(() => {});
    // Cleanup
    if (userId) await api.deleteUser(userId).catch(() => {});
    if (roleId) await api.deleteRole(roleId).catch(() => {});
  });

  test("user with limited role sees only permitted menu items", async () => {
    await page.goto("/chat");
    await page.waitForSelector("aside nav");
    await page.waitForTimeout(500);

    const menuItems = await getSidebarMenuItems(page);

    // Should see Chat (has chat:read permission)
    expect(menuItems.some((item) => item.includes("chat"))).toBeTruthy();

    // Should NOT see user/role management items without matching permissions.
    expect(menuItems.some((item) => item.includes("users"))).toBeFalsy();
    expect(menuItems.some((item) => item.includes("roles"))).toBeFalsy();

    // Should NOT see Analytics/Dashboard (no analytics permissions)
    expect(
      menuItems.some(
        (item) => item.includes("analytics") || item.includes("dashboard"),
      ),
    ).toBeFalsy();
  });

  test("user with limited role is redirected from unauthorized pages", async () => {
    // Try to access settings directly
    await page.goto("/settings");
    await page.waitForLoadState("networkidle");

    // Should be redirected away from settings
    expect(page.url()).not.toContain("/settings");
  });

  test("user with limited role can access permitted pages", async () => {
    // Should be able to access chat
    await page.goto("/chat");
    await page.waitForLoadState("networkidle");

    expect(page.url()).toContain("/chat");
    await expect(page.locator("body")).not.toContainText("forbidden", {
      ignoreCase: true,
    });
  });

  test("user lands on first accessible page after login", async () => {
    // Should land on chat (first accessible route based on permissions)
    expect(initialLandingUrl).toContain("/chat");
  });
});

test.describe("Role with Settings Access", () => {
  test.describe.configure({ mode: "serial" });

  const roleName = generateUniqueName("E2E Settings Role");
  const userEmail = generateUniqueEmail("e2e-settings");
  const userPassword = "Password123!";

  let api: ApiHelper;
  let roleId: string;
  let userId: string;
  let context: BrowserContext;
  let page: Page;

  test.beforeAll(async ({ browser }) => {
    api = adminApi;

    // Create role with settings permissions
    const permissions = await api.findPermissionKeys([
      { resource: "chat", action: "read" },
      { resource: "users", action: "read" },
      { resource: "users", action: "write" },
      { resource: "settings.general", action: "read" },
    ]);

    const role = await api.createRole({
      name: roleName,
      description: "E2E test role with settings access",
      permissions,
    });
    roleId = role.id;

    const user = await api.createUser({
      email: userEmail,
      password: userPassword,
      full_name: "E2E Settings User",
      role_id: roleId,
    });
    userId = user.id;

    ({ context, page } = await createLoggedInSession(
      browser,
      userEmail,
      userPassword,
    ));
  });

  test.afterAll(async () => {
    await context?.close().catch(() => {});
    if (userId) await api.deleteUser(userId).catch(() => {});
    if (roleId) await api.deleteRole(roleId).catch(() => {});
  });

  test("user with settings permission sees Settings menu", async () => {
    await page.goto("/settings/users");
    await page.waitForSelector("aside nav");
    await page.waitForTimeout(500);

    const menuItems = await getSidebarMenuItems(page);
    expect(menuItems.some((item) => item.includes("settings"))).toBeTruthy();
  });

  test("user with users:read can access users page", async () => {
    await page.goto("/settings/users");
    await page.waitForLoadState("networkidle");

    expect(page.url()).toContain("/settings/users");
    await expect(page.locator('table, [role="table"]').first()).toBeVisible();
  });

  test("user with users:write sees Add button", async () => {
    await page.goto("/settings/users");
    await page.waitForLoadState("networkidle");

    // Should see Add/Create button
    const addButton = page.locator("button").filter({ hasText: /add|create/i });
    await expect(addButton.first()).toBeVisible();
  });
});

test.describe("Role with Read-Only Roles Access", () => {
  test.describe.configure({ mode: "serial" });

  const roleName = generateUniqueName("E2E Roles Read Role");
  const userEmail = generateUniqueEmail("e2e-roles-read");
  const userPassword = "Password123!";

  let api: ApiHelper;
  let roleId: string;
  let userId: string;
  let context: BrowserContext;
  let page: Page;

  test.beforeAll(async ({ browser }) => {
    api = adminApi;

    const permissions = await api.findPermissionKeys([
      { resource: "roles", action: "read" },
    ]);

    const role = await api.createRole({
      name: roleName,
      description: "E2E test role with read-only roles access",
      permissions,
    });
    roleId = role.id;

    const user = await api.createUser({
      email: userEmail,
      password: userPassword,
      full_name: "E2E Read Only Roles User",
      role_id: roleId,
    });
    userId = user.id;

    ({ context, page } = await createLoggedInSession(
      browser,
      userEmail,
      userPassword,
    ));
  });

  test.afterAll(async () => {
    await context?.close().catch(() => {});
    if (userId) await api.deleteUser(userId).catch(() => {});
    if (roleId) await api.deleteRole(roleId).catch(() => {});
  });

  test("user with roles:read can inspect roles without create or delete controls", async () => {
    await page.goto("/settings/roles");
    await page.waitForLoadState("networkidle");

    const tablePage = new TablePage(page);
    const dialogPage = new DialogPage(page);

    await expect(page).toHaveURL(/\/settings\/roles/);
    await expect(tablePage.addButton).toHaveCount(0);
    await expect(page.locator('button[aria-label*="Delete role"]')).toHaveCount(
      0,
    );

    await tablePage.search("admin");
    await tablePage.editRow("admin");
    await dialogPage.waitForOpen();

    await expect(
      dialogPage.dialog.getByRole("button", { name: "Close" }).first(),
    ).toBeVisible();
    await expect(
      dialogPage.dialog.getByRole("button", { name: "Update Role" }),
    ).toHaveCount(0);
    await expect(dialogPage.dialog.getByLabel("Description")).toBeDisabled();
  });
});

test.describe("Role with Accounts Access", () => {
  test.describe.configure({ mode: "serial" });

  const roleName = generateUniqueName("E2E Accounts Role");
  const userEmail = generateUniqueEmail("e2e-accounts");
  const userPassword = "Password123!";

  let api: ApiHelper;
  let roleId: string;
  let userId: string;
  let context: BrowserContext;
  let page: Page;

  test.beforeAll(async ({ browser }) => {
    api = adminApi;

    const permissions = await api.findPermissionKeys([
      { resource: "accounts", action: "read" },
    ]);

    const role = await api.createRole({
      name: roleName,
      description: "E2E test role with WhatsApp settings access",
      permissions,
    });
    roleId = role.id;

    const user = await api.createUser({
      email: userEmail,
      password: userPassword,
      full_name: "E2E Accounts User",
      role_id: roleId,
    });
    userId = user.id;

    ({ context, page } = await createLoggedInSession(
      browser,
      userEmail,
      userPassword,
    ));
  });

  test.afterAll(async () => {
    await context?.close().catch(() => {});
    if (userId) await api.deleteUser(userId).catch(() => {});
    if (roleId) await api.deleteRole(roleId).catch(() => {});
  });

  test("user with accounts:read can access /settings/instances", async () => {
    await page.goto("/settings/instances");
    await page.waitForLoadState("networkidle");

    expect(page.url()).toContain("/settings/instances");
  });
});

test.describe("Admin vs Limited Role Comparison", () => {
  test.describe.configure({ mode: "serial" });

  let context: BrowserContext;
  let page: Page;

  test.beforeAll(async ({ browser }) => {
    context = await browser.newContext();
    page = await context.newPage();
    await loginAsAdmin(page);
  });

  test.afterAll(async () => {
    await context?.close().catch(() => {});
  });

  test("admin sees all menu items", async () => {
    await page.goto("/chat");
    await page.waitForSelector("aside nav");
    await page.waitForTimeout(500);

    const menuItems = await getSidebarMenuItems(page);

    expect(menuItems.some((item) => item.includes("chat"))).toBeTruthy();
    expect(menuItems.some((item) => item.includes("settings"))).toBeTruthy();
  });

  test("admin can access all settings pages", async () => {
    // Can access users
    await page.goto("/settings/users");
    await page.waitForLoadState("networkidle");
    expect(page.url()).toContain("/settings/users");

    // Can access roles
    await page.goto("/settings/roles");
    await page.waitForLoadState("networkidle");
    expect(page.url()).toContain("/settings/roles");

    // Can access general settings
    await page.goto("/settings");
    await page.waitForLoadState("networkidle");
    expect(page.url()).toContain("/settings");
  });
});

test.describe("Dynamic Role Updates", () => {
  test.describe.configure({ mode: "serial" });

  const roleName = generateUniqueName("E2E Dynamic Role");
  const userEmail = generateUniqueEmail("e2e-dynamic");
  const userPassword = "Password123!";

  let api: ApiHelper;
  let roleId: string;
  let userId: string;
  let context: BrowserContext;
  let page: Page;

  test.beforeAll(async ({ browser }) => {
    api = adminApi;

    // Create role with minimal permissions
    const permissions = await api.findPermissionKeys([
      { resource: "chat", action: "read" },
    ]);

    const role = await api.createRole({
      name: roleName,
      description: "E2E test role for dynamic updates",
      permissions,
    });
    roleId = role.id;

    const user = await api.createUser({
      email: userEmail,
      password: userPassword,
      full_name: "E2E Dynamic User",
      role_id: roleId,
    });
    userId = user.id;

    ({ context, page } = await createLoggedInSession(
      browser,
      userEmail,
      userPassword,
    ));
  });

  test.afterAll(async () => {
    await context?.close().catch(() => {});
    if (userId) await api.deleteUser(userId).catch(() => {});
    if (roleId) await api.deleteRole(roleId).catch(() => {});
  });

  test("user initially has limited access", async () => {
    await page.goto("/chat");
    await page.waitForSelector("aside nav");

    const menuItems = await getSidebarMenuItems(page);

    // Should see Chat
    expect(menuItems.some((item) => item.includes("chat"))).toBeTruthy();

    // Should NOT see Settings
    expect(menuItems.some((item) => item.includes("settings"))).toBeFalsy();
  });
});
