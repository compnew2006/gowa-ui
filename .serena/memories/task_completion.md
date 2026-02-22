# Task Completion Checklist

After implementing any change:

1. **Build check**: `cd frontend && npm run build` (must exit 0)
2. **Type check**: `cd frontend && npx vue-tsc --noEmit` (note: pre-existing errors in ChatbotFlowBuilderView.vue and MigrationView.vue are known)
3. **Update CHANGELOG.md**: Add entry under `## [Unreleased]` with Added/Changed/Fixed
4. **Update RALPH_MEMORY.md**: Add entry with The Trap, The Reality, The Fix, The Law
5. **Regenerate STRUCTURE.md**: `python3 gen_md_structure.py`
6. **Git commit**: `git add -A && git commit -m "<message>"`
