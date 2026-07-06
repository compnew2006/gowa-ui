---
name: mcp-code-operations
description: Mandatory MCP-only code operations policy. Use for every coding task that searches, reads, edits, creates, deletes, refactors, debugs, or verifies source code. Routes Socraticode to code understanding and function relationships, codebase-memory-mcp to architectural memory and existing patterns, and Serena to precise read/edit/remove operations.
---

# MCP Code Operations

This skill is mandatory for source-code work in Whatomate.

## Non-Negotiable Rules

1. **Do not use internal tools for source-code search, edit, create, or remove.**
   - Forbidden on source code: internal `read`, `edit`, `write`, shell `cat`, `head`, `tail`, `less`, `grep`, `rg`, `ag`, `find`, and `ls`.
   - Allowed shell use: tests, builds, linters, formatters, package managers, git, and empty `mkdir`/`touch` scaffolding only when paired with Serena for content.
2. **If an MCP needed for safe code work is unavailable, stop and ask before using an internal fallback.**
3. **Read before edit.** Never change source before an MCP has located and inspected the target.
4. **Understand relationships before change.** Use Socraticode and/or codebase-memory-mcp to identify callers, callees, dependencies, routes, and blast radius.
5. **Modify with Serena only.** Source edits, symbol deletion, renames, and targeted replacements go through Serena tools.

## Best Tool for Each Job

| MCP | Best job | Use when | Main tools |
|---|---|---|---|
| **Socraticode** | Live code understanding and relationship mapping | Semantic discovery, file dependency graph, symbol 360°, call flow, impact/blast radius, circular dependency checks | `codebase_status`, `codebase_update`, `codebase_search`, `codebase_symbols`, `codebase_symbol`, `codebase_flow`, `codebase_impact`, `codebase_graph_query`, `codebase_graph_stats`, `codebase_graph_circular` |
| **codebase-memory-mcp** | Persistent project memory and existing architecture patterns | Prior patterns, architecture graph, route/channel graph, cross-service links, ADRs, stale-memory detection | `search_graph`, `search_code`, `get_architecture`, `trace_path`, `query_graph`, `detect_changes`, `index_repository`, `manage_adr` |
| **Serena** | Precise source reading, editing, diagnostics, and memory notes | Symbol overview/body, declarations/references, safe edits, rename, delete, diagnostics, project memories | `get_symbols_overview`, `find_symbol`, `find_declaration`, `find_referencing_symbols`, `get_diagnostics_for_file`, `replace_symbol_body`, `replace_content`, `insert_after_symbol`, `insert_before_symbol`, `rename_symbol`, `safe_delete_symbol`, `write_memory` |

## Operation Routing

### Search / Discover Code

1. Start with Socraticode:
   - `socraticode_codebase_search(query)` for natural-language discovery.
   - `socraticode_codebase_graph_query(filePath)` for known file dependencies.
   - `socraticode_codebase_symbol(name)` / `socraticode_codebase_symbols(query/file)` for known symbols.
2. Check existing patterns with codebase-memory-mcp:
   - `codebase_memory_mcp_search_graph(query)`.
   - `codebase_memory_mcp_search_code(pattern)`.
   - `codebase_memory_mcp_get_architecture()`.
3. Use Serena for exact source locations:
   - `serena_get_symbols_overview(relative_path)`.
   - `serena_find_symbol(name_path_pattern, include_body only when needed)`.
   - `serena_find_declaration(regex)`.

### Read / Analyze Code

1. Use Serena for authoritative source structure and symbol bodies.
2. Use Socraticode for callers/callees, dependency direction, impact, and execution flow.
3. Use codebase-memory-mcp for remembered architecture and code patterns.
4. Internal `read` is allowed only for non-source documentation/skill files, or after explicit MCP fallback approval.

### Edit Existing Code

1. Pre-check with Socraticode:
   - `socraticode_codebase_impact(target)`.
   - `socraticode_codebase_symbol(name)` or `socraticode_codebase_flow(entrypoint)`.
   - `socraticode_codebase_graph_query(filePath)` for file-level dependencies.
2. Pattern-check with codebase-memory-mcp:
   - `codebase_memory_mcp_search_graph(query = "pattern for <change>")`.
3. Inspect with Serena:
   - `serena_get_symbols_overview` then `serena_find_symbol` / `serena_find_referencing_symbols`.
4. Edit with Serena only:
   - whole symbol: `serena_replace_symbol_body`
   - targeted lines: `serena_replace_content`
   - new declarations near symbols: `serena_insert_before_symbol` / `serena_insert_after_symbol`
   - rename: `serena_rename_symbol`
5. Verify after edit:
   - `socraticode_codebase_update()` if needed.
   - `socraticode_codebase_impact(changed target)`.
   - `socraticode_codebase_graph_circular()`.
   - `serena_get_diagnostics_for_file()`.

### Create Code

1. Prefer adding symbols into existing plugin/extension files with Serena insertion tools.
2. For a new source file, use shell only for `mkdir -p` / `touch` if no MCP file-create tool exists, then populate content with Serena.
3. Never use internal `write` for source code unless the user explicitly approves an MCP-unavailable fallback.

### Remove Code

1. Symbol/function/class removal:
   - Check `socraticode_codebase_impact(target)`.
   - Check `serena_find_referencing_symbols`.
   - Remove with `serena_safe_delete_symbol` when possible.
2. File removal:
   - Check `socraticode_codebase_impact(file)` and `socraticode_codebase_graph_query(filePath)`.
   - Search codebase-memory for ownership/patterns.
   - Ask the user before deleting the file.
   - If no MCP file-delete tool exists, use `git rm` only after approval.
3. Verify no broken references/cycles after removal.

## Whatomate-Specific Guardrails

- New product features should go under `plugin/<name>/` when possible.
- Core paths (`internal/handlers`, `internal/models`, `internal/database`, `internal/middleware`, `internal/tenant`, `pkg`) require approval for non-bug-fix behavior changes.
- Multi-tenant scoping must be preserved with `app.requestDB(rc)` in core handlers or `tenant.ScopedDB()` in plugins.
- Do not add plugin migrations to `internal/database/postgres.go`.
- Use `fasthttp` + `fastglue`, not Gin or new `net/http` handlers.

## Fallback Policy

- **Serena unavailable**: no source edits; ask the user whether to continue with internal tools.
- **Socraticode unavailable**: do not change relationship-sensitive code; ask before Serena-only reference analysis.
- **codebase-memory-mcp unavailable**: proceed only after noting that pattern memory is unavailable.
- **Any internal-tool fallback for source code**: requires explicit user approval in the current conversation.
