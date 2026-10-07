# Agent Tooling Reference

Read this file only when tool installation, registration, repair, or detailed discovery is needed. The standing instructions define authorization boundaries; this reference does not expand them.

## Portable tool discovery and configuration

Keep this reusable prompt independent of the machine. Never embed installation paths, home directories, usernames, drive letters, clone locations, or path placeholders in it. Refer to tools by their verified project/package names and official source links. Discover their actual locations afresh in each environment.

1. Inspect the tools and skills exposed by the active client, then inspect its supported MCP registrations and skill-discovery metadata. Read only the configuration fields needed for discovery; keep credentials out of output. An available remote or client-managed tool may not need a local executable. A server's registration name does not determine its installation directory.
2. For a required local command, resolve it through the current environment using `command -v` on macOS/Linux or `Get-Command` in PowerShell. If unresolved, check the configured launcher, relevant package-manager metadata, and the tool's documented installation mechanism. Resolve relative locations against their documented base and follow symlinks when needed. Search only relevant installation/configuration locations; do not scan the entire machine or assume a username, home directory, drive, package-manager prefix, or repository clone location.
3. Discover skill files, references, optional routing files, browser extension assets, and built server entry points through the active client's catalog, installation metadata, and current upstream documentation. Do not assume they live inside the current repository or a particular global directory. If a known installation has moved, verify its new location and repair only the affected local registration rather than reinstalling or creating duplicates.
4. When a required tool is missing, install it from the official references below using its supported installer and an appropriate user-scoped location for the current OS/client. Discover the resulting executable or assets after installation; do not derive their locations from this prompt or from another machine's setup.
5. Configure launchers using the active client's documented command and argument format. Account for the client's actual launch environment: a command that works in an interactive shell may be unavailable to a desktop app. Use a resolved absolute executable or asset path in local client configuration only when needed. Use environment-variable expansion or a launcher wrapper only when the client supports it; do not assume shell expansion. Preserve proper argument boundaries, including paths containing spaces.
6. Keep resolved paths in machine-local configuration or launchers only. Do not copy them into this prompt, shared agent instructions, or portable setup examples. On another machine, repeat discovery and setup. Before reusing an existing registration, verify that its executable and required assets still exist and are accessible to that client.
7. Verify startup, MCP initialization, expected capabilities, and the intended repository or browser context from the actual client. Keep `browser-controller` as the Browser Controller MCP registration name; discover its exposed tool names from the current session rather than inferring them from a directory name. Apply the existing reload/checkpoint procedure when configuration changes require reconnection.

## Installation reference: official projects and downloads

Use these sources during preflight to install missing tools. Read the current upstream instructions, confirm platform/version requirements, and verify release integrity when the publisher provides checksums or signatures. These links identify the projects; they do not authorize upgrading working installations or changing the project's dependency choices. Download installer scripts for inspection before executing them.

### Core workflow tools

| Tool | Official source and installation/download links | Installation guidance |
| --- | --- | --- |
| codebase-memory-mcp | [Repository and setup guide](https://github.com/DeusData/codebase-memory-mcp), [release downloads](https://github.com/DeusData/codebase-memory-mcp/releases), [macOS/Linux installer source](https://github.com/DeusData/codebase-memory-mcp/blob/main/install.sh), [Windows installer source](https://github.com/DeusData/codebase-memory-mcp/blob/main/install.ps1) | Select the release for the actual OS/architecture. Follow the current installer options and preserve existing agent configuration and indexes. |
| Serena | [Official repository](https://github.com/oraios/serena), [installation guide](https://github.com/oraios/serena/blob/main/docs/02-usage/010_installation.md), [serena-agent package](https://pypi.org/project/serena-agent/) | Use the documented uv installation for `serena-agent` and a compatible Python interpreter. Register its MCP server for the actual coding client. |
| Graphify | [Official repository and setup](https://github.com/Graphify-Labs/graphify), [graphifyy package](https://pypi.org/project/graphifyy/) | The Python package is `graphifyy`; the command is `graphify`. Install the appropriate extras only when needed, then install its skill for the active client. |
| Context7 MCP | [Official repository](https://github.com/upstash/context7), [client setup guide](https://context7.com/docs/resources/all-clients) | Use the documented local `@upstash/context7-mcp` package or supported remote connection. Preserve existing authentication and report any required user login. |
| Playwright MCP | [Microsoft repository and MCP setup](https://github.com/microsoft/playwright-mcp) | The MCP package is `@playwright/mcp`. Register it for the active client and obtain any required compatible browser through its documented setup. Use it for localhost testing. |
| Browser Controller MCP | [compnew2006/browser-controller: source, build, extension and pairing guide](https://github.com/compnew2006/browser-controller) | This is the intended existing-profile controller. Follow its repository installation rather than guessing an npm package with a similar name. |
| Impeccable | [Official repository](https://github.com/pbakaus/impeccable), [installation options](https://impeccable.style/), [installation/update FAQ](https://impeccable.style/faq/) | Use its supported installer for the target provider and scope. The documented CLI entry is `npx impeccable install`; inspect its current options before using it. Install the skill and its required engine, then verify client discovery. |

Register Browser Controller MCP as `browser-controller`. Verify and reuse an existing installation; if its registration uses a different name, update that registration while preserving its settings instead of creating a duplicate. Follow its current README to build the server and discover the actual Node.js executable, built MCP entry point, and browser extension assets on the current machine. Register the discovered launcher in the local client configuration, load the discovered extension into the intended existing Chrome/Chromium/Edge profile, and pair it with the local daemon. For multiple profiles, follow its documented per-profile connection setup. Preserve existing sessions and pairing credentials. If an extension-loading or pairing step needs user interaction, finish the independent setup and report that exact step; do not print pairing secrets in chat or logs.

### Missing prerequisites

Install only prerequisites needed by the selected tools or project, while preserving working runtimes:

- [uv: official installation instructions](https://docs.astral.sh/uv/getting-started/installation/) for isolated Python tooling and Serena/Graphify setup.
- [Node.js: official downloads](https://nodejs.org/en/download) for a compatible Node.js/npm installation when missing.
- [Git: official downloads](https://git-scm.com/downloads/) when repository operations require Git and it is missing.
- [Go: official installation instructions](https://go.dev/doc/install) only when the project or selected Go tooling requires a missing Go installation.

### Project-specific validation tools

Use the project's existing tool choices and version constraints. These references support required checks; they are not an instruction to install every tool or replace an existing test framework.

| Tool | Official installation/source reference |
| --- | --- |
| Gitleaks | [Source and installation](https://github.com/gitleaks/gitleaks), [release downloads](https://github.com/gitleaks/gitleaks/releases) |
| govulncheck | [Official Go package and installation](https://pkg.go.dev/golang.org/x/vuln/cmd/govulncheck) |
| Semgrep | [Official source and installation](https://github.com/semgrep/semgrep) |
| golangci-lint | [Official local installation guide](https://golangci-lint.run/docs/welcome/install/local/) |
| Staticcheck | [Official installation guide](https://staticcheck.dev/docs/getting-started/) |
| pre-commit | [Official installation guide](https://pre-commit.com/#installation) |
| Playwright Test | [Official test-runner installation](https://playwright.dev/docs/intro); this is separate from Playwright MCP |
| Vitest | [Official getting-started guide](https://vitest.dev/guide/) |
| Vue Test Utils | [Official installation guide](https://test-utils.vuejs.org/installation/) when the project uses Vue |

SkillPilot is an optional local routing integration. If its existing files/scanner are absent, discover the installed tools directly; do not install an unrelated same-named package. For any other required tool, verify its official source from the project's manifest or publisher before installing it.

