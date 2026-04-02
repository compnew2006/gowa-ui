import { describe, expect, it } from "vitest";

import {
  buildMarketingRedirectTarget,
  shouldAutoRedirect,
} from "./marketing-redirect";

describe("marketing-redirect", () => {
  it("builds an external redirect target that preserves route, query, and hash", () => {
    expect(
      buildMarketingRedirectTarget({
        marketingBaseUrl: "https://marketing.example.com/sidecar",
        currentPath: "/pricing",
        search: "?utm=demo",
        hash: "#plans",
      }),
    ).toBe("https://marketing.example.com/sidecar/pricing?utm=demo#plans");
  });

  it("supports same-origin path prefixes for sidecar routing", () => {
    expect(
      buildMarketingRedirectTarget({
        marketingBaseUrl: "/marketing",
        currentPath: "/offer",
        origin: "https://app.example.com",
      }),
    ).toBe("https://app.example.com/marketing/offer");
  });

  it("returns null when the marketing destination is not configured", () => {
    expect(
      buildMarketingRedirectTarget({
        marketingBaseUrl: "   ",
        currentPath: "/plans",
      }),
    ).toBeNull();
  });

  it("suppresses redirects when the target matches the current location", () => {
    expect(
      shouldAutoRedirect(
        "https://app.example.com/pricing",
        "https://app.example.com/pricing",
      ),
    ).toBe(false);
  });
});
