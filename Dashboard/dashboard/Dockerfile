FROM node:20-alpine AS base
RUN corepack enable && corepack prepare pnpm@10.15.1 --activate

FROM base AS deps
WORKDIR /app
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml .npmrc ./
COPY artifacts/dashboard/package.json artifacts/dashboard/package.json
RUN pnpm install --frozen-lockfile --filter @workspace/dashboard...

FROM base AS builder
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY --from=deps /app/artifacts/dashboard/node_modules ./artifacts/dashboard/node_modules
COPY . .
RUN pnpm --filter @workspace/dashboard build

FROM base AS runner
WORKDIR /app
ENV NODE_ENV=production

RUN addgroup --system --gid 1001 nodejs && \
    adduser --system --uid 1001 nextjs

COPY --from=builder --chown=nextjs:nodejs /app/artifacts/dashboard/public ./artifacts/dashboard/public
COPY --from=builder --chown=nextjs:nodejs /app/artifacts/dashboard/.next/standalone ./
COPY --from=builder --chown=nextjs:nodejs /app/artifacts/dashboard/.next/static ./artifacts/dashboard/.next/static

USER nextjs
EXPOSE 3000

CMD ["node", "artifacts/dashboard/server.js"]
