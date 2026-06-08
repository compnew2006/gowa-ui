---
name: agent-orchestrator
description: >
  Universal 5-phase agent orchestration workflow for any codebase.
  Enforces: session startup → memory + skill selection → deep analysis →
  decision gate → implementation → verification → architectural audit.
  Includes SWARM protocol for large tasks, MCP-first tooling rules,
  strict Serena-first file operations, and mandatory codegraph/codebase-memory
  verification before any code change.
  Activate this skill on EVERY coding session regardless of project.
---

# Agent Orchestrator — Universal Workflow v2.0

**This skill defines the mandatory workflow for every coding session and every user request.**
It is project-agnostic — it works with any codebase, any language, any framework.

All project-specific details (build commands, architecture, critical paths) are
discovered dynamically during onboarding and analysis phases.

---

## ⛔ HARD RULES — Always Active, No Exceptions

These rules are non-negotiable. They apply at ALL times during the session.

### Rule 1: MCP-First, Shell-Second

**Forbidden for code navigation:**
```
cat  head  tail  less  grep  rg  ag  find  ls  ← FORBIDDEN on source files
```

Use MCP tools (`serena`, `codegraph`, `codebase-memory-mcp`) for all code navigation.

**Shell allowed ONLY for:**
- Running tests
- Git operations
- Package installs (npm, pip, go get, etc.)
- Linters and formatters
- Running/building the application
- File creation (`touch`, `mkdir`)

### Rule 2: Serena-First for All File Operations

**Reading files:**
```
serena → get_symbols_overview(path = file)     ← structure overview
serena → find_symbol(name = symbol)            ← locate specific symbol
serena → find_declaration(name = symbol)       ← find where declared
serena → search_for_pattern(pattern = text)    ← search across files
```

**Editing files:**
```
serena → replace_symbol_body(...)              ← modify functions/methods
serena → replace_content(...)                  ← targeted text replacement
serena → insert_after_symbol(...)              ← add code after symbol
serena → insert_before_symbol(...)             ← add code before symbol
serena → rename_symbol(...)                    ← rename across project
serena → safe_delete_symbol(...)               ← delete with safety check
```

> **NEVER edit a file without reading it first.**
> Always use `serena → get_symbols_overview()` or read the file before any modification.

### Rule 3: Verify Before Change — codegraph

**Before ANY code modification, verify with codegraph:**
```
codegraph → fn_impact(name = target, flags = -T)     ← who will be affected?
codegraph → co_changes(file = target)                 ← what files usually change together?
codegraph → where(name = target)                      ← where is this used?
```

**After ANY code modification, verify with codegraph:**
```
git add <edited-file>
codegraph → diff_impact(flags = --staged -T)          ← did we break anything?
codegraph → check(flags = --staged --no-new-cycles)   ← any new cycles?
```

If unexpected dependents appear → **STOP. Report to user. Do NOT commit.**

### Rule 4: Check Patterns First — codebase-memory-mcp

**Before implementing anything, check if a pattern already exists:**
```
codebase-memory-mcp → search_graph(query = task description)    ← existing architecture patterns
codebase-memory-mcp → search_code(pattern = keyword)            ← existing code patterns
codebase-memory-mcp → get_architecture()                        ← high-level project structure
```

If an existing pattern is found → **follow it**. Do not invent new patterns when
the codebase already has an established way of doing things.

### Rule 5: Files Under 500 Lines

- Keep all files under 500 lines. Extract into new files when exceeding.
- Preserve all existing comments and docstrings unrelated to your changes.
- Follow existing code style and conventions discovered during onboarding.

---

## RUNTIME DETECTION

Execute at the very start of every session:

```
If running in OpenCode  → prepend ALL tool calls with server prefix:
                          codegraph_*, serena_*, codebase_memory_*, chrome_devtools_*, zai_*, openspace_*
If running in Claude Code → use bare tool names: semantic_search, fn_impact, replace_symbol_body ...
If running in Antigravity → use call_mcp_tool with ServerName and ToolName
```

---

## MCP SERVER AVAILABILITY CHECK

**Execute BEFORE Session Startup. Determines which tools are available.**

```
Step 0: Check available MCP servers

  List all available MCP servers and tools.
  Mark availability:
    serena              ☐ available  ☐ unavailable
    codegraph           ☐ available  ☐ unavailable
    codebase-memory-mcp ☐ available  ☐ unavailable
    chrome-devtools     ☐ available  ☐ unavailable
    zai-mcp-server      ☐ available  ☐ unavailable
    openspace           ☐ available  ☐ unavailable

  FALLBACK MODES:
  ┌─────────────────────────┬────────────────────────────────────────────────┐
  │ If unavailable          │ Fallback behavior                             │
  ├─────────────────────────┼────────────────────────────────────────────────┤
  │ serena                  │ Use built-in file read/edit tools.             │
  │                         │ Skip Steps 1-3 of Session Startup.            │
  │                         │ Use grep_search for pattern searching.        │
  │                         │ Use view_file / replace_file_content for edits│
  ├─────────────────────────┼────────────────────────────────────────────────┤
  │ codegraph               │ Use serena → find_referencing_symbols for     │
  │                         │ impact analysis. Skip automated blast radius. │
  │                         │ Manually verify callers before editing.       │
  │                         │ Skip Architecture Enforcement steps.          │
  ├─────────────────────────┼────────────────────────────────────────────────┤
  │ codebase-memory-mcp     │ Skip Phase 1 memory search.                   │
  │                         │ Proceed without memory — log "no memory       │
  │                         │ available" and continue.                       │
  │                         │ Skip post-success index_repository.           │
  ├─────────────────────────┼────────────────────────────────────────────────┤
  │ chrome-devtools         │ Skip visual verification in Phase 5.          │
  │                         │ Rely on test suite only for UI changes.       │
  ├─────────────────────────┼────────────────────────────────────────────────┤
  │ zai-mcp-server          │ Skip ui_diff_check and diagnose_error.        │
  │                         │ Use chrome-devtools screenshots if available. │
  ├─────────────────────────┼────────────────────────────────────────────────┤
  │ openspace               │ Skip skill activation via openspace.          │
  │                         │ Load skills manually from skills-map.md.      │
  └─────────────────────────┴────────────────────────────────────────────────┘

  Store availability status for the rest of this session.
```

---

## SESSION STARTUP — Execute Every Session (in order)

These 6 steps run **once** at the start of every coding session, in strict order:

### Step 1: Load Serena Instructions
```
serena → initial_instructions()
```
> Skip if serena unavailable (see Fallback Modes above).

### Step 2: Project Onboarding + Context Detection
```
serena → onboarding()          ← skip if done in last 24h
```

During onboarding (especially first session), identify and memorize:
- Project language(s) and framework(s)
- Build system and commands (Makefile, package.json scripts, Cargo.toml, etc.)
- Test framework and commands
- Lint/typecheck commands
- Directory structure and architectural layers
- Core/critical directories that should NOT be modified without approval
- Extension mechanism (plugins, modules, packages, middleware, decorators, etc.)
- Scoping mechanism (multi-tenancy, user scoping, org scoping, or none)
- Error handling convention (custom error types, HTTP envelope, exceptions, etc.)

Store findings:
```
serena → write_memory(name = "project-context", content = findings)
```

> This combines the previous Steps 2 and 7 — no redundancy.

### Step 3: Load Memory
```
serena → list_memories()
serena → read_memory("summary")     ← if exists — NEVER replace, only append
serena → read_memory("project-context")  ← if exists — load project conventions
```

### Step 4: Check and Load Skills Map
```
Does skills-map.md exist?
  YES → read it
  NO  → auto-generate it:
    1. serena → search_for_pattern(substring_pattern = "\\.md$", relative_path = ".claude/skills")
    2. serena → search_for_pattern(substring_pattern = "\\.md$", relative_path = ".agents/skills")
    3. For each skill file found: read first 20 lines → extract description + keywords
    4. bash: touch skills-map.md
    5. serena → replace_content(file = "skills-map.md", content = generated routing table)
```

**Skills Map format:**
```markdown
# Skills Map — Auto-generated <DATE>
<!-- Regenerate: delete this file and restart session -->

| Skill ID | File | Description | Best For |
|----------|------|-------------|----------|
| `skill-name` | `path/to/skill.md` | ... | ... |

## Routing Rules
| Task signals | Load skill |
|---|---|
| style, design, CSS, component, page, view | ui-ux skill |
| test, spec, playwright, e2e, unit | testing skill |
| refactor, extract, rename, cleanup | code-intelligence |
| build, add, implement, feature | code-intelligence + domain skill |
| fix, bug, error, broken | code-intelligence |
| model, migration, schema, query | database skill |
| handler, endpoint, route, API | API skill |
```

### Step 5: Ensure summary.md Exists
```
Does summary.md exist and contain "<!-- END -->"?
  NO  → create it with: "# Summary\n\n<!-- END -->"
  YES → do nothing
```

### Step 6: Scan Recent Changes
```
codegraph → semantic_search("recent changes; last task; last modified")
```
> Skip if codegraph unavailable.

---

## REQUEST CLASSIFICATION — Before Phase 1

**Not every request needs all 5 phases. Classify first:**

```
┌──────────────────────────────────────────────────────────────────────┐
│                      USER REQUEST RECEIVED                          │
└──────────────────────────┬───────────────────────────────────────────┘
                           │
                           ▼
              ┌─────────────────────────┐
              │  What type of request?  │
              └─────────────────────────┘
                     │
     ┌───────────────┼───────────────────┬────────────────────┐
     │               │                   │                    │
     ▼               ▼                   ▼                    ▼
  TYPE A          TYPE B              TYPE C               TYPE D
  Investigation   Trivial Change      Standard Change      Large Change
  ─────────────   ──────────────      ───────────────      ────────────
  "explain X"     "fix typo"          "add feature"        "> 5 files"
  "where is Y"    "add comment"       "fix bug"            "full-stack"
  "how does Z"    "format this"       "refactor X"         "architectural"
  "show me"       "rename A→B"        "implement Y"        "cross-module"
  "list all"      "update string"     "change behavior"    "new subsystem"
     │               │                   │                    │
     ▼               ▼                   ▼                    ▼
  Phase 1 + 2     Phase 2 + 4 + 5     ALL 5 Phases         ALL 5 + SWARM
  (read-only)     (skip Gate)         (full workflow)       (multi-agent)
```

---

## LOCAL ORCHESTRATOR PROTOCOL

**Master workflow for code change requests. Phase execution depends on request type (see above).**

---

### PHASE 1 — MEMORY + SKILL SELECTION

```
Step 1: Check existing patterns in codebase memory
    codebase-memory-mcp → search_graph(query = task description)
    codebase-memory-mcp → search_code(pattern = keyword)
    codebase-memory-mcp → get_architecture()

    ⚠️ If a matching pattern is found:
       → READ it carefully
       → FOLLOW the established pattern
       → Do NOT invent new approaches when one already exists

Step 2: Read summary from Serena memory
    serena → read_memory("summary")      ← read only, never replace

Step 3: Select and activate matching skill
    Read skills-map.md → match task type to skill
    Activate the selected skill

Note: Empty memory = normal. Log "no prior patterns" and continue.
```

---

### PHASE 2 — ANALYSIS

> Use `codegraph` tools on current `HEAD` (no `--staged` flag) to analyze the baseline.
> Use `serena` to read and understand file contents. NEVER use cat/head/tail.

```
Step 1: Semantic search for the task
    codegraph → semantic_search(query = task description)

Step 2: Find target symbol
    serena → find_symbol(name = target)

Step 3: Locate definition and all usages
    codegraph → where(name = target)

Step 4: Analyze function-level blast radius
    codegraph → fn_impact(name = target, flags = -T)

Step 5: Analyze file-level impact
    codegraph → impact_analysis(target = module/file)

Step 6: Get full context (source + deps + callers + tests)
    codegraph → context(name = target, flags = -T)

Step 7: Detect historically coupled files
    codegraph → co_changes(file = target file)

Step 8: Read and understand target file structure
    serena → get_symbols_overview(path = target file)
```

**Adaptive analysis — not every step needed for every task:**
- **Investigation** → Steps 1, 2, 3, 8 may suffice
- **Simple rename** → Steps 2, 3, 4
- **Bug fix** → Steps 1, 2, 3, 6, 8
- **New feature** → All steps
- **Refactor** → Steps 2, 3, 4, 5, 7, 8

**If a tool returns empty results, use fallback chain:**

| Primary Tool | Fallback 1 | Fallback 2 |
|---|---|---|
| `codegraph → semantic_search` | `codebase-memory → search_code` | `serena → search_for_pattern` |
| `codegraph → fn_impact` | `codegraph → query` (manual callers) | `serena → find_referencing_symbols` |
| `codegraph → where` | `serena → find_symbol` + `find_declaration` | `serena → search_for_pattern` |
| `codegraph → co_changes` | `git log --follow` (shell allowed) | Skip — proceed with extra caution |
| `codegraph → diff_impact` | `codegraph → fn_impact` per changed fn | `serena → find_referencing_symbols` |
| `codegraph → impact_analysis` | `codegraph → file_deps` | `serena → find_referencing_symbols` |
| `serena → find_symbol` | `codegraph → where` | `codegraph → semantic_search` |
| `serena → get_symbols_overview` | `codegraph → brief` | `codegraph → list_functions` |

---

### PHASE 3 — DECISION GATE

**The most important phase — determines whether to proceed or wait for approval.**

#### Step 1: Build and present risk table

| Symbol | File | Direct Callers | Cross-Module? | Risk |
|--------|------|----------------|---------------|------|
| ...    | ...  | ...            | ...           | ...  |

#### Step 2: Check for existing patterns before proposing new ones

```
codebase-memory-mcp → search_graph(query = "pattern for <what you're about to do>")

If pattern found → follow it, note in risk table: "follows existing pattern → lower risk"
If no pattern   → note in risk table: "new pattern → higher risk, document after success"
```

#### Step 3: Apply decision rules

**🔴 HARD STOP — wait for explicit user approval if:**
- Any symbol has > 5 callers AND at least 1 is cross-module
- `codegraph → check()` returns a cycle warning
- `codegraph → co_changes()` returns > 3 coupled files not in current scope
- Edit targets core/critical paths identified during onboarding
- Change modifies a public API or interface contract
- Change affects shared infrastructure (auth, middleware, database layer, etc.)
- No existing pattern found AND change touches > 3 files
- `codegraph → fn_impact()` shows transitive callers across > 2 modules

**🟢 AUTO-PROCEED — no approval needed if:**
- All callers are in the same module/package
- New files with zero existing callers
- Pure styling/UI changes with no backend impact
- Changes within the project's designated extension mechanism
- Documentation-only changes
- Test-only changes
- Follows an established pattern found in codebase-memory

---

### PHASE 4 — IMPLEMENTATION

```
Step 1: Create a new git branch
    git checkout -b agent/<kebab-case-task-name>

Step 2: READ before EDIT — mandatory for every file
    ⚠️ Before touching ANY file, you MUST read it first:

    serena → get_symbols_overview(path = target file)   ← understand structure
    serena → find_symbol(name = specific function)      ← locate what to change

    Only after reading and understanding the file, proceed to edit.

Step 3: Check impact BEFORE editing — mandatory
    codegraph → fn_impact(name = function-to-change, flags = -T)
    codegraph → where(name = function-to-change)
    serena → find_referencing_symbols(name = function-to-change)

    Understand WHO depends on this symbol before you touch it.

Step 4: Execute edits using Serena tools

    | Operation                    | Serena Tool              |
    |------------------------------|--------------------------|
    | Modify existing function     | replace_symbol_body      |
    | Targeted text/regex replace  | replace_content          |
    | Add code after a symbol      | insert_after_symbol      |
    | Add code before a symbol     | insert_before_symbol     |
    | Rename across project        | rename_symbol            |
    | Delete with safety check     | safe_delete_symbol       |
    | Find usages before editing   | find_referencing_symbols |
    | New file                     | bash: touch <path> then replace_content |

    ⚠️ New files require TWO steps:
       1. bash: touch <path>
       2. serena → replace_content(file = path, content = content)

    ⚠️ Code Quality:
       - Keep files under 500 lines. Extract when exceeding.
       - Preserve all existing comments/docstrings unrelated to your changes.
       - Follow existing code style discovered during onboarding.
       - Follow existing patterns found via codebase-memory-mcp.

Step 5: After EACH edit — verify with codegraph
    git add <edited-file>
    codegraph → diff_impact(flags = --staged -T)
    codegraph → check(flags = --staged --no-new-cycles)

    ┌─────────────────────────────────────────────────────────┐
    │  If unexpected new dependents or cycles appear:         │
    │  ← STOP immediately                                    │
    │  ← Report to user with full impact table                │
    │  ← Do NOT commit                                        │
    │  ← Consider reverting: git checkout -- <file>           │
    └─────────────────────────────────────────────────────────┘

Step 6: Commit if everything is clean
    git add -p && git commit -m "<type>(<scope>): <description>"

    Commit types: feat, fix, refactor, test, docs, chore, style, perf
    Commit scope = most specific affected area:
      - Module/package name (e.g., auth, payments, chat)
      - Feature name (e.g., campaign-scheduler)
      - Layer name (e.g., api, db, ui, middleware)
      - Plugin/extension name (e.g., plugin/analytics)

    Examples:
      feat(auth): add refresh token rotation
      fix(chat): prevent duplicate message delivery
      refactor(db): extract query builder into helper
      test(api): add integration tests for contacts endpoint
```

---

### PHASE 5 — VERIFICATION

> Two tracks based on the type of change:

#### Track A: Frontend / UI Changes

```
Step 1: Run E2E tests
    npx playwright test --project=chromium       ← or project-specific test command
    npx playwright test                           ← full suite if shared components changed
    npx playwright test --debug                   ← if failures

Step 2: Visual verification (if chrome-devtools available)
    chrome-devtools → take_screenshot()
    zai-mcp-server → ui_diff_check()             ← if zai available
    zai-mcp-server → diagnose_error_screenshot()  ← if console errors visible
```

#### Track B: Backend Changes

```
Step 1: Run tests and lint
    <project test command>                        ← make test, npm test, cargo test, go test, etc.
    <project lint command>                        ← make lint, npm run lint, etc.
    <project typecheck command>                   ← if applicable (npm run typecheck, mypy, etc.)

Step 2: Structural verification with codegraph
    codegraph → check(flags = --staged --no-new-cycles)
    codegraph → diff_impact(flags = --staged -T)
```

#### After Success (both tracks):

```
Step 3: Save pattern to codebase memory
    codebase-memory-mcp → index_repository()

Step 4: Save approach notes to Serena memory
    serena → write_memory(
        name = "feature/<name>-<date>",
        content = "approach taken + gotchas + files changed"
    )

Step 5: Append to summary.md (NEVER replace)
    serena → replace_content(
        file    = "summary.md",
        pattern = "<!-- END -->",
        replace = "\n## <task-name> — <date>\n<summary>\n<!-- END -->"
    )

    summary.md entry must include:
    ├── Task name + date
    ├── Files changed (with links)
    ├── Approach taken
    ├── Blast radius table (from Phase 3)
    ├── Patterns followed or created
    ├── Tests run and results
    └── Gotchas and notes for future sessions
```

#### After Failure (3 attempts exhausted):

```
Step 6: Save work safely
    git add -p
    git stash push -m "agent/<task>/failed-attempt-<N>"

    If stash fails (SWARM mode / conflicts):
    git branch agent/<task>/backup-<timestamp>
    git reset --hard HEAD

Step 7: Document failure in summary.md
    Append: failure reason + stash ref + what was tried

Step 8: Report to user
    Provide: exact error + files affected + approaches attempted + stash reference
```

---

## ARCHITECTURE ENFORCEMENT — After Every Session With Code Changes

```
Step 1: Comprehensive audit
    codegraph → audit(target = modified files, -T)

Step 2: Prioritize issues
    codegraph → triage(-T)

Step 3: Detect dead code
    codegraph → node_roles(role = dead, -T)

Step 4: Measure complexity
    codegraph → complexity(-T)

Step 5: Verify memory is up to date
    codebase-memory-mcp → detect_changes()

⚠️ If found:
   - Dependency cycles        → Report as 🔴 CRITICAL warning
   - Dead exports             → Report as 🟡 MEDIUM warning
   - Complexity > 20          → Report as 🟡 MEDIUM warning
   - Stale codebase memory    → Re-index: codebase-memory-mcp → index_repository()

Report all warnings BEFORE proceeding with any further tasks.
```

> Skip if codegraph or codebase-memory-mcp unavailable (see Fallback Modes).

---

## EXTENSION-FIRST PRINCIPLE

> The universal rule: prefer extensions over core modifications.

Before ANY implementation, apply this decision rule:

```
Does this change touch core/critical paths identified during onboarding?
  YES → Is this a bug fix or a necessary core interface change?
    NO  → Use the project's extension mechanism instead:
          - Go: plugin/ directory or separate package
          - Node.js: new module/package
          - Python: new module or decorator
          - Java/Kotlin: new package or Spring bean
          - Rails: new concern, service object, or engine
          - Django: new app
          - Vue/React: new composable/hook + component
          - PHP/Laravel: new service provider or package
          - Rust: new module or crate
    YES → Get explicit user approval first.
  NO → Proceed normally.
```

When unsure whether something is "core":
```
codegraph → node_roles(target = file, -T)

If role = "core" or "entry" → treat as critical, require approval
If role = "utility" or "adapter" or "leaf" → lower risk, can auto-proceed
```

---

## SWARM PROTOCOL — For Large Tasks

> Activate automatically if: task affects > 5 files OR spans frontend + backend simultaneously

### Role Assignments

| Role | Count | Phases | Write Access |
|---|---|---|---|
| LEAD | 1 | All | All files + git + summary.md |
| SCOUT | 2 | 1-3 only | Read-only |
| IMPLEMENT | N | 4 only | Declared files only |
| QA | 1 | 5 only | Test files only |
| DOC | 1 | Post-5 | summary.md + memory |

### Coordination Protocol

```
Step 1: LEAD creates task plan
    Create task.md artifact with:
    ├── File assignments per IMPLEMENT agent
    ├── Dependency order (which files must be done first)
    ├── Interface contracts between agents' files
    └── Merge strategy

Step 2: SCOUT agents research (Phases 1-3)
    Each SCOUT runs Phase 2 analysis on assigned modules
    Results reported to LEAD via:
    serena → write_memory(name = "scout-<N>-findings", content = analysis)

Step 3: LEAD builds Decision Gate (Phase 3)
    Aggregates SCOUT findings
    Builds combined risk table
    Determines go/no-go

Step 4: IMPLEMENT agents declare ownership
    Each agent declares its files BEFORE editing:
    serena → write_memory(name = "agent-<N>-owns", content = ["file1", "file2"])

    LEAD checks all declarations for overlap:
    ├── Overlap found → reassign files → re-declare
    └── No overlap → authorize agents to proceed

    ⚠️ NEVER allow concurrent edits to the same file.

Step 5: IMPLEMENT agents execute (Phase 4)
    Each agent works ONLY on its declared files
    Each agent follows Phase 4 rules (read→check→edit→verify)
    Status updates via:
    serena → write_memory(name = "agent-<N>-status", content = "done|blocked|failed")

Step 6: LEAD monitors progress
    Reads agent status memories
    Resolves blocked agents
    Re-assigns failed agent's files

Step 7: QA agent runs verification (Phase 5)
    Runs full test suite
    Reports results to LEAD

Step 8: DOC agent writes documentation
    Updates summary.md with all changes
    Saves patterns to codebase-memory-mcp
```

### Token Budget Management
- LEAD: receives full skill instructions
- SCOUT/IMPLEMENT/QA: receive only their relevant phase sections
- If context > 80k tokens: load tool reference on-demand from references/ directory

---

## EXECUTION ORDER — Quick Reference

| # | Step | When | Required Tools |
|---|---|---|---|
| 0 | MCP Availability Check | Session start | None (checks what's available) |
| 1 | serena initial_instructions | Session start | serena |
| 2 | Project onboarding + context | Session start (every 24h) | serena |
| 3 | Load memory | Session start | serena |
| 4 | Skills map check/generate | Session start | serena |
| 5 | summary.md check | Session start | serena |
| 6 | Scan recent changes | Session start | codegraph |
| 7 | **Classify request** | Every request | None (agent judgment) |
| 8 | Memory search + pattern check + skill selection | Every request — Phase 1 | codebase-memory, serena |
| 9 | Deep analysis with fallbacks | Every request — Phase 2 | codegraph, serena |
| 10 | Risk table + pattern check + decide | Every request — Phase 3 | codegraph, codebase-memory |
| 11 | Read → Check → Edit → Verify → Commit | Every request — Phase 4 | serena, codegraph |
| 12 | Tests + screenshots + structural check | Every request — Phase 5 | project tools, codegraph |
| 13 | Save pattern + index + update summary.md | After success | codebase-memory, serena |
| 14 | Architecture audit | End of session with changes | codegraph, codebase-memory |
| 15 | SWARM protocol | If > 5 files or frontend+backend | All tools |

---

## TOOL REFERENCE

Tool reference tables are in a separate file to save context window space:
```
references/tool-reference.md
```

Load on-demand when you need to look up a specific tool's purpose or flags.

**Quick summary of available tool counts:**
- Codegraph: 34 tools (code analysis, impact, search)
- Codebase-Memory-MCP: 14 tools (knowledge graph, patterns)
- Serena: 21 tools (LSP operations, memory, file edits)
- Chrome-DevTools: 29 tools (browser automation)
- Zai-MCP-Server: 8 tools (visual analysis)
- Openspace: 4 tools (skill management)
- ECC Plugins: playwright (23), github (25), context7 (2), memory (7), sequential-thinking (1)
