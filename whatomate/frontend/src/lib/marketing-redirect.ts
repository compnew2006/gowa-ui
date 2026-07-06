const PLACEHOLDER_ORIGIN = "https://whatomate.invalid";

type MarketingRedirectOptions = {
  marketingBaseUrl: string;
  currentPath: string;
  search?: string;
  hash?: string;
  origin?: string;
};

function joinUrlPath(basePath: string, currentPath: string): string {
  const normalizedBase = basePath.replace(/\/+$/, "");
  const normalizedCurrent = currentPath.replace(/^\/+/, "");

  if (normalizedBase === "") {
    return normalizedCurrent === "" ? "/" : `/${normalizedCurrent}`;
  }

  if (normalizedCurrent === "") {
    return normalizedBase;
  }

  return `${normalizedBase}/${normalizedCurrent}`;
}

export function buildMarketingRedirectTarget({
  marketingBaseUrl,
  currentPath,
  search = "",
  hash = "",
  origin = PLACEHOLDER_ORIGIN,
}: MarketingRedirectOptions): string | null {
  const trimmedBaseUrl = marketingBaseUrl.trim();
  if (trimmedBaseUrl === "") {
    return null;
  }

  try {
    const targetUrl = new URL(trimmedBaseUrl, origin);
    targetUrl.pathname = joinUrlPath(targetUrl.pathname, currentPath);
    targetUrl.search = search;
    targetUrl.hash = hash;
    return targetUrl.toString();
  } catch {
    return null;
  }
}

export function shouldAutoRedirect(
  targetUrl: string | null,
  currentUrl: string,
): boolean {
  if (!targetUrl) {
    return false;
  }

  try {
    return new URL(targetUrl).toString() !== new URL(currentUrl).toString();
  } catch {
    return targetUrl !== currentUrl;
  }
}
