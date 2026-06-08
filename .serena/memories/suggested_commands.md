# Suggested Commands — Whatomate

## Backend
```sh
make build                         # build Go binary
make test                          # all Go tests
go test -v -race -cover ./...      # verbose with race + coverage
go test -v -run TestFoo ./pkg/...  # single package
golangci-lint run                  # Go lint
```

## Frontend
```sh
cd frontend && npm run dev         # Vite dev (port 3000, proxy to 8080)
cd frontend && npm run build       # production build
cd frontend && npm run typecheck   # vue-tsc --noEmit
cd frontend && npm run lint        # eslint
cd frontend && npm run format      # prettier
cd frontend && npm run test:unit   # vitest
cd frontend && npm run test:e2e    # playwright
```

## Combined dev
```sh
make frontend-dev                  # frontend dev only
make backend-watch                 # Go hot-reload with air
make dev-watch                     # both frontend + backend
make build-prod                    # frontend build + Go build (single binary)
```

## Docker
```sh
make docker-up                     # docker compose up -d db redis
make docker-down                   # docker compose down
```

## Migration
```sh
make migrate                       # run pending migrations
```

## Darwin-specific
```sh
# sed -i needs empty backup extension on macOS
sed -i '' 's/foo/bar/g' file.go

# xargs -I is standard on macOS
```

Use `-p 1` for multi-package Go test runs sharing the test DB:
```sh
go test -p 1 -v ./internal/handlers/... ./internal/models/...
```
