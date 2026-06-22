import { beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  api: {
    get: vi.fn(),
    put: vi.fn(),
  },
}));

vi.mock("@/services/api", () => ({
  api: mocks.api,
}));

import { modulesService } from "./modules";

describe("modulesService", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("uses the module catalog endpoints", () => {
    modulesService.listEffective();
    modulesService.listGlobal();
    modulesService.listOrganization("org-1");

    expect(mocks.api.get).toHaveBeenNthCalledWith(1, "/modules/effective");
    expect(mocks.api.get).toHaveBeenNthCalledWith(2, "/admin/modules");
    expect(mocks.api.get).toHaveBeenNthCalledWith(
      3,
      "/organizations/org-1/modules",
    );
  });

  it("updates global and organization state with encoded keys", () => {
    modulesService.updateGlobal("facebook/comments", false);
    modulesService.updateOrganization("org-1", "facebook/comments", true);

    expect(mocks.api.put).toHaveBeenNthCalledWith(
      1,
      "/admin/modules/facebook%2Fcomments",
      { enabled: false },
    );
    expect(mocks.api.put).toHaveBeenNthCalledWith(
      2,
      "/organizations/org-1/modules/facebook%2Fcomments",
      { enabled: true },
    );
  });
});
