<!--
## Sync Impact Report
- Version change: 1.0.0 → 1.1.0 (MINOR — new principles and expanded guidance)
- Modified principles:
  - II. Zero-Regression Changes → expanded with Strangler Pattern mandate
  - III. Test-First Verification → expanded with reproduction archival workflow
  - VII. Modular File Structure → expanded with anti-hallucination import rules
- Added principles:
  - VIII. Surgical Impact Analysis
  - IX. Persistent Learning (Ralph Method)
  - X. Skeptical Self-Review
- Added sections:
  - Mandatory Live Documents (under Development Workflow)
  - Context Anchoring (under Development Workflow)
- Removed sections: None
- Templates requiring updates:
  - `.specify/templates/plan-template.md` — ✅ Constitution Check aligns; no changes needed
  - `.specify/templates/spec-template.md` — ✅ User stories and requirements align; no changes needed
  - `.specify/templates/tasks-template.md` — ✅ Phase structure and test-first align; no changes needed
- Follow-up TODOs: None
-->

# Whatomate Constitution

## Core Principles

### I. Adapter-First Architecture

All external service integrations (WhatsApp providers, AI engines,
payment gateways) MUST be implemented behind a well-defined Go
interface. Concrete implementations (Meta Cloud API, whatsmeow,
OpenAI, Anthropic) MUST be interchangeable without modifying
handler or service layers.

- Every adapter MUST implement the same interface contract.
- Handler code MUST NOT import provider-specific packages directly.
- New providers MUST be addable by implementing the interface and
  registering in the connection manager.

### II. Zero-Regression Changes (NON-NEGOTIABLE)

Every modification MUST be verified against existing functionality
before merge. Breaking changes MUST be documented with a migration
path.

- Database migrations MUST preserve rollback capability (keep old
  tables during transition periods).
- API endpoint removals MUST be versioned or gated behind feature
  flags.
- Frontend MUST gracefully degrade when backend features are
  unavailable (e.g., hide Meta-only UI when using whatsmeow).
- **Strangler Pattern**: When modifying critical, complex, or
  high-dependency files, you MUST NOT edit the existing function
  in-place. Instead: (1) CREATE a new file/module with the improved
  logic, (2) SWITCH imports in consuming files one by one,
  (3) DELETE the old file only after full verification. This ensures
  instant rollback by reverting a single import.

### III. Test-First Verification

New features and bug fixes MUST include verification evidence.
For backend: integration tests against the handler/service layer.
For frontend: E2E tests (Playwright) covering critical user
journeys. No PR is complete without a passing test suite.

- Reproduction scripts MUST precede bug fixes.
- Contract tests MUST cover all API endpoint changes.
- WebSocket event flows MUST have integration test coverage.
- **Reproduction Archival**: After a fix is verified, the
  reproduction script MUST be moved to
  `.repro_archive/YYYY-MM-DD_issue_[id].ext` with a header comment
  explaining the bug and the fix. This creates a permanent
  regression safety net.

### IV. Multi-Tenant Isolation

All data access MUST be scoped by `organization_id`. No endpoint
or service method may return or modify data belonging to a
different organization.

- Database queries MUST include organization filtering.
- API handlers MUST extract and enforce tenant context from JWT
  claims.
- Session stores (whatsmeow device data) MUST be isolated per
  instance and per organization.

### V. Real-Time First

User-facing state changes (connection status, QR codes, incoming
messages, delivery receipts) MUST be delivered via WebSocket in
real-time. HTTP polling is acceptable only as a fallback health
check.

- New event types MUST be documented in the API specification.
- Frontend components MUST subscribe to relevant WebSocket events
  on mount and unsubscribe on unmount.
- Event payloads MUST include the `instance_id` for routing to the
  correct UI context.

### VI. Single-Binary Simplicity

Whatomate ships as a single binary with an embedded frontend. All
features MUST work with the default `config.toml` and a PostgreSQL
database. External service dependencies (Redis, message queues) are
optional optimizations, never requirements for core functionality.

- New features MUST NOT introduce mandatory external service
  dependencies.
- The `make build-prod` pipeline MUST produce a single deployable
  artifact.
- CLI commands (`server`, `worker`, `version`) MUST remain the
  primary operational interface.

### VII. Modular File Structure

Source files MUST NOT exceed 500 lines. Features MUST be organized
into focused packages.

- Backend: `internal/handlers/`, `internal/service/`,
  `internal/models/`, `internal/repository/`, `pkg/`.
- Frontend: single-responsibility components under
  `frontend/src/` feature directories.
- New backend features MUST follow handler → service → repository
  layering.
- New frontend features MUST use Vue 3 Composition API with
  shadcn-vue components.
- Shared utilities: `pkg/` (backend), `frontend/src/utils/`
  (frontend).
- **Anti-Hallucination**: Before importing any package or module,
  verify it exists via `ls`, `package.json`, or `go.mod`. NEVER
  assume an import path. When adding new functions/classes, create
  a new file and import it — do not pile logic into existing files
  (Puzzle Pattern).

### VIII. Surgical Impact Analysis

BEFORE writing a single line of production code modification, a
"Blast Radius" check MUST be performed:

1. **Read** the target file.
2. **Trace** all files importing or using the target function/class
   (1 level deep: direct imports + callers of callers).
3. **Report** an explicit list: "Modifying `X` will affect: [A, B, C]".
4. **Decide** whether the change is safe or requires the Strangler
   Pattern (Principle II).

This principle exists to prevent cascading regressions from
seemingly small changes.

### IX. Persistent Learning (Ralph Method)

Immediately after verifying a fix and before final cleanup, the
team MUST record a learning entry in `RALPH_MEMORY.md`:

```
## [YYYY-MM-DD] Issue: [Brief Title]
- **The Trap:** [What assumption or approach failed]
- **The Reality:** [What was actually true about the codebase]
- **The Fix:** [The surgical change that worked]
- **The Law:** [One-sentence rule to prevent this regression]
```

`RALPH_MEMORY.md` MUST be read at the start of every development
session to avoid repeating past mistakes.

### X. Skeptical Self-Review

After agreeing on an implementation plan and BEFORE writing code,
a visible self-debate MUST be performed to identify weak points:

```
## 🛑 SKEPTICAL REVIEW
- **The Plan:** [Brief summary]
- **The Critic:** "This is fragile because..." or
  "This adds unnecessary complexity because..."
- **The Defense/Fix:** "Good point. I will instead..."
```

If no weak point can be found, look harder. There is always a
tradeoff.

## Technology Constraints

| Layer | Technology | Version Policy |
|:------|:-----------|:---------------|
| Backend | Go (Fastglue) | Pin minor version in `go.mod` |
| Frontend | Vue.js 3 + shadcn-vue | Pin in `package.json` |
| Database | PostgreSQL | GORM for ORM, raw SQL for complex queries |
| WhatsApp | whatsmeow (primary), Meta Cloud API (legacy) | Pin version, wrap in adapter |
| Auth | JWT (access + refresh tokens) | HMAC-SHA256, configurable expiry |
| Real-time | WebSocket (Gorilla) | Hub pattern, per-org channels |
| AI | OpenAI, Anthropic, Google | Adapter interface, provider-agnostic |
| Testing | Go `testing` + testify / Playwright | Run on every PR |
| Deployment | Docker / single binary | `make build-prod` = embedded frontend |

## Development Workflow

### Code Change Lifecycle

1. **Scout**: Map the blast radius (Principle VIII).
2. **Reproduce**: For bugs, create a reproduction script that fails
   against current code (Principle III).
3. **Design**: For features, write spec and plan before
   implementation. Run Skeptical Self-Review (Principle X).
4. **Implement**: Follow adapter/strangler patterns for risky
   changes (Principles I, II). Files stay under 500 lines (VII).
5. **Verify**: Run the full test suite. Confirm no regressions.
   Update CHANGELOG.md.
6. **Learn**: Record in RALPH_MEMORY.md if a surprise was
   encountered (Principle IX).
7. **Document**: Regenerate STRUCTURE.md.

### Mandatory Live Documents

These files MUST be maintained as living project memory:

| Document | Purpose | Update Trigger |
|:---------|:--------|:---------------|
| `PLAN.md` | Immediate next steps for the current task | Start of each task |
| `CHANGELOG.md` | What changed and WHY | After every implementation |
| `RALPH_MEMORY.md` | Persistent learning from past mistakes | After verifying a fix |
| `STRUCTURE.md` | File/export map of the repository | After any structural change |

### Context Anchoring

At the start of every session or immediately after implementation,
`STRUCTURE.md` MUST be regenerated (via `python3 gen_md_structure.py`
or equivalent) and reviewed to ensure the file hierarchy and
exports are correct.

### PR Requirements

- All tests MUST pass.
- CHANGELOG.md MUST be updated.
- Breaking API changes MUST include migration notes.
- Frontend changes MUST include screenshots or browser recordings.

### Branch Strategy

- Feature branches: `feature/[short-description]`
- Bug fixes: `fix/[short-description]`
- All branches merge to `main` via PR with review.

## Governance

This constitution is the authoritative source for development
standards in the Whatomate project. It supersedes ad-hoc decisions
and informal conventions.

- **Amendments** require: (1) a documented rationale, (2) impact
  analysis on existing code, and (3) an update to this file with
  version increment.
- **Versioning** follows semantic versioning: MAJOR for principle
  removals/redefinitions, MINOR for new principles or expanded
  guidance, PATCH for clarifications and typo fixes.
- **Compliance** is verified during code review. PRs introducing
  architectural changes MUST reference the relevant constitution
  principles.
- **Exceptions** MUST be documented in the plan's Complexity
  Tracking table with justification and rejected alternatives.

**Version**: 1.1.0 | **Ratified**: 2026-02-17 | **Last Amended**: 2026-02-17
