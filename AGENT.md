# AGENT.md

## Operational Rules (Ralph Protocol)

1. **Start-Up**: Run Serena MCP activation and onboarding if not done.
2. **Context Ingestion**: Read AGENT.md, MEMORY.md, CHANGELOG.md, session_summary.md before generating code.
3. **Silent Execution Protocol**: No chat, immediate action, output only results.
4. **Code Integrity**: Blast-radius analysis before modifying functions.
5. **Modular Architecture**: New features in class/struct-based modules.

## THE LOOP: Implement -> Review -> Test -> Fix (Until Success)
- **Phase 1: IMPLEMENT**: Write code following modular architecture.
- **Phase 2: REVIEW**: Correctness, completeness, blast radius, imports, types, modularity, syntax.
- **Phase 3: TEST**: Build, Lint, Unit Tests, E2E Tests, Manual Tests.
- **Phase 4: FIX**: Diagnose and apply surgical fix, route back to Review.

## Post-Loop Protocol
- Update `MEMORY.md` with timestamp `date +"%Y-%m-%d %H:%M"`.
- Add to `CHANGELOG.md`.
- Create/update `session_summary.md`.
- Regenerate `STRUCTURE.md`.
- Create/Update E2E tests.
- Git commit.
