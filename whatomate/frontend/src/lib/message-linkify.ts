export interface LinkifiedTextSegmentText {
  type: "text";
  text: string;
}

export interface LinkifiedTextSegmentLink {
  type: "link";
  text: string;
  href: string;
}

export type LinkifiedTextSegment =
  | LinkifiedTextSegmentText
  | LinkifiedTextSegmentLink;

const URL_PATTERN =
  /(?:https?:\/\/[^\s<>"']+|www\.[^\s<>"']+|(?:[a-z0-9](?:[a-z0-9-]*[a-z0-9])?\.)+[a-z]{2,}(?:[/?#][^\s<>"']*)?)/gi;
const TRAILING_PUNCTUATION = new Set([".", ",", "!", "?", ";", ":"]);
const BALANCED_DELIMITER_PAIRS = new Map<string, string>([
  [")", "("],
  ["]", "["],
  ["}", "{"],
]);

function appendTextSegment(
  segments: LinkifiedTextSegment[],
  text: string,
): void {
  if (!text) {
    return;
  }

  const previousSegment = segments[segments.length - 1];
  if (previousSegment?.type === "text") {
    previousSegment.text += text;
    return;
  }

  segments.push({ type: "text", text });
}

function countOccurrences(value: string, char: string): number {
  let count = 0;

  for (const currentChar of value) {
    if (currentChar === char) {
      count += 1;
    }
  }

  return count;
}

function trimTrailingUrlCharacters(candidate: string): {
  trimmedText: string;
  trailingText: string;
} {
  let trimmedText = candidate;
  let trailingText = "";

  while (trimmedText) {
    const trailingChar = trimmedText.at(-1);
    if (!trailingChar) {
      break;
    }

    if (TRAILING_PUNCTUATION.has(trailingChar)) {
      trailingText = `${trailingChar}${trailingText}`;
      trimmedText = trimmedText.slice(0, -1);
      continue;
    }

    const openingChar = BALANCED_DELIMITER_PAIRS.get(trailingChar);
    if (!openingChar) {
      break;
    }

    const openingCount = countOccurrences(trimmedText, openingChar);
    const closingCount = countOccurrences(trimmedText, trailingChar);

    if (closingCount > openingCount) {
      trailingText = `${trailingChar}${trailingText}`;
      trimmedText = trimmedText.slice(0, -1);
      continue;
    }

    break;
  }

  return { trimmedText, trailingText };
}

function normalizeUrlHref(candidate: string): string | null {
  const normalizedCandidate =
    candidate.startsWith("http://") || candidate.startsWith("https://")
      ? candidate
      : `https://${candidate}`;

  try {
    const parsedUrl = new URL(normalizedCandidate);
    if (parsedUrl.protocol !== "http:" && parsedUrl.protocol !== "https:") {
      return null;
    }
    if (!parsedUrl.hostname || !parsedUrl.hostname.includes(".")) {
      return null;
    }
    return parsedUrl.toString();
  } catch {
    return null;
  }
}

function hasSafeLinkBoundary(text: string, matchIndex: number): boolean {
  if (matchIndex === 0) {
    return true;
  }

  const previousChar = text[matchIndex - 1];
  return /\s/.test(previousChar) || `([{<"'`.includes(previousChar);
}

export function segmentMessageLinks(text: string): LinkifiedTextSegment[] {
  if (!text) {
    return [{ type: "text", text: "" }];
  }

  const segments: LinkifiedTextSegment[] = [];
  let cursor = 0;

  for (const match of text.matchAll(URL_PATTERN)) {
    const matchIndex = match.index ?? -1;
    const rawCandidate = match[0] ?? "";

    if (matchIndex < 0 || !rawCandidate || !hasSafeLinkBoundary(text, matchIndex)) {
      continue;
    }

    const candidateStart = matchIndex;

    if (candidateStart > cursor) {
      appendTextSegment(segments, text.slice(cursor, candidateStart));
    }

    const { trimmedText, trailingText } =
      trimTrailingUrlCharacters(rawCandidate);
    const normalizedHref = normalizeUrlHref(trimmedText);

    if (!trimmedText || !normalizedHref) {
      appendTextSegment(segments, rawCandidate);
      cursor = candidateStart + rawCandidate.length;
      continue;
    }

    segments.push({
      type: "link",
      text: trimmedText,
      href: normalizedHref,
    });
    appendTextSegment(segments, trailingText);

    cursor = candidateStart + rawCandidate.length;
  }

  if (cursor < text.length) {
    appendTextSegment(segments, text.slice(cursor));
  }

  return segments.length > 0 ? segments : [{ type: "text", text }];
}
