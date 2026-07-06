import { describe, expect, it } from "vitest";

import { segmentMessageLinks } from "./message-linkify";

describe("segmentMessageLinks", () => {
  it("returns a plain text segment when no links are present", () => {
    expect(segmentMessageLinks("hello world")).toEqual([
      { type: "text", text: "hello world" },
    ]);
  });

  it("linkifies http and https urls inside message text", () => {
    expect(
      segmentMessageLinks("Visit https://example.com/docs for more details."),
    ).toEqual([
      { type: "text", text: "Visit " },
      {
        type: "link",
        text: "https://example.com/docs",
        href: "https://example.com/docs",
      },
      { type: "text", text: " for more details." },
    ]);
  });

  it("normalizes bare domains and www links to https hrefs", () => {
    expect(
      segmentMessageLinks("Check www.example.com or example.org/path?q=1"),
    ).toEqual([
      { type: "text", text: "Check " },
      {
        type: "link",
        text: "www.example.com",
        href: "https://www.example.com/",
      },
      { type: "text", text: " or " },
      {
        type: "link",
        text: "example.org/path?q=1",
        href: "https://example.org/path?q=1",
      },
    ]);
  });

  it("keeps trailing punctuation outside the clickable anchor", () => {
    expect(
      segmentMessageLinks("(https://example.com/path). Next: foo.bar/test,"),
    ).toEqual([
      { type: "text", text: "(" },
      {
        type: "link",
        text: "https://example.com/path",
        href: "https://example.com/path",
      },
      { type: "text", text: "). Next: " },
      {
        type: "link",
        text: "foo.bar/test",
        href: "https://foo.bar/test",
      },
      { type: "text", text: "," },
    ]);
  });

  it("does not turn email addresses into anchors", () => {
    expect(
      segmentMessageLinks("Send mail to support@example.com before browsing."),
    ).toEqual([
      {
        type: "text",
        text: "Send mail to support@example.com before browsing.",
      },
    ]);
  });
});
