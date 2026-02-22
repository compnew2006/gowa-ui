# Suggested Commands

## Frontend
```bash
# Dev server
cd frontend && npm run dev

# Build
cd frontend && npm run build

# Type check
cd frontend && npx vue-tsc --noEmit

# Lint
cd frontend && npm run lint

# E2E tests
cd frontend && npm run test:e2e

# Format
cd frontend && npm run format
```

## Backend
```bash
# Build production
make build-prod

# Run
./whatomate -config config.toml
```

## Utilities
```bash
# Regenerate structure
python3 gen_md_structure.py

# Git
git status / git diff / git log -n 5
```
