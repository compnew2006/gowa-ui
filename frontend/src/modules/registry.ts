export interface CompiledModuleDefinition {
  key: string;
  pathPrefixes: readonly string[];
}

export const compiledModules: readonly CompiledModuleDefinition[] = Object.freeze([
  {
    key: "facebook-accounts",
    pathPrefixes: ["/facebook/accounts"],
  },
  {
    key: "facebook-comments",
    pathPrefixes: ["/facebook/comments"],
  },
  {
    key: "facebook-page-search",
    pathPrefixes: ["/facebook/page-search"],
  },
  {
    key: "facebook-people-search",
    pathPrefixes: ["/facebook/people-search"],
  },
  {
    key: "facebook-auto-share",
    pathPrefixes: ["/facebook/auto-share"],
  },
  {
    key: "facebook-extract-data",
    pathPrefixes: ["/facebook/extract-data"],
  },
  {
    key: "facebook-extract-likes",
    pathPrefixes: ["/facebook/extract-likes"],
  },
  {
    key: "facebook-group-search",
    pathPrefixes: ["/facebook/group-search"],
  },
  {
    key: "facebook-page-messengers",
    pathPrefixes: ["/facebook/page-messengers"],
  },
  {
    key: "facebook-retargeting",
    pathPrefixes: ["/facebook/retargeting"],
  },
  // facebook-core must stay LAST: its "/facebook" prefix is the catch-all for
  // the Facebook hub itself and any future sub-paths without a dedicated
  // module. Specific module prefixes above take precedence because
  // moduleKeyForPath returns on the first match.
  {
    key: "facebook-core",
    pathPrefixes: ["/facebook"],
  },
]);

function normalizedPath(path: string): string {
  const cleanPath = path.split(/[?#]/, 1)[0] || "/";
  if (cleanPath.length > 1 && cleanPath.endsWith("/")) {
    return cleanPath.slice(0, -1);
  }
  return cleanPath;
}

export function moduleKeyForPath(path: string): string | undefined {
  const normalized = normalizedPath(path);
  for (const module of compiledModules) {
    if (
      module.pathPrefixes.some(
        (prefix) => normalized === prefix || normalized.startsWith(prefix + "/"),
      )
    ) {
      return module.key;
    }
  }
  return undefined;
}

export function isCompiledModulePathEnabled(
  path: string,
  isModuleEnabled: (key: string) => boolean,
): boolean {
  const moduleKey = moduleKeyForPath(path);
  return moduleKey === undefined || isModuleEnabled(moduleKey);
}
