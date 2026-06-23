import { describe, expect, it } from "vitest";
import {
  isCompiledModulePathEnabled,
  moduleKeyForPath,
} from "./registry";

describe("compiled module registry", () => {
  it("keeps non-module paths enabled and gates compiled-module paths", () => {
    const isEnabled = (key: string) => key !== "facebook-comments";

    expect(isCompiledModulePathEnabled("/chat", isEnabled)).toBe(true);
    expect(
      isCompiledModulePathEnabled("/facebook/comments", isEnabled),
    ).toBe(false);
    expect(
      isCompiledModulePathEnabled("/facebook/accounts", isEnabled),
    ).toBe(true);
  });

  it.each([
    ["/facebook", "facebook-core"],
    ["/facebook/accounts", "facebook-accounts"],
    ["/facebook/accounts/account-1", "facebook-accounts"],
    ["/facebook/comments", "facebook-comments"],
    ["/facebook/comments/comment-1", "facebook-comments"],
    ["/facebook/page-search", "facebook-page-search"],
    ["/facebook/people-search", "facebook-people-search"],
    ["/facebook/group-search", "facebook-group-search"],
    ["/facebook/extract-likes", "facebook-extract-likes"],
    ["/facebook/extract-data", "facebook-extract-data"],
    ["/facebook/page-messengers", "facebook-page-messengers"],
    ["/facebook/auto-share", "facebook-auto-share"],
    ["/facebook/retargeting", "facebook-retargeting"],
    ["/facebook/retargeting/123", "facebook-retargeting"],
    ["/chat", undefined],
  ])("maps %s to %s", (path, expected) => {
    expect(moduleKeyForPath(path)).toBe(expected);
  });
});
