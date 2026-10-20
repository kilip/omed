FROM oven/bun:1-alpine AS deps
WORKDIR /app

COPY package.json bun.lock ./
COPY apps/web/package.json ./apps/web/
COPY packages/openapi/package.json ./packages/openapi/
COPY packages/tsconfig/package.json ./packages/tsconfig/

RUN --mount=type=cache,id=bun,target=/root/.bun/install/cache \
    bun install --frozen-lockfile --ignore-scripts

FROM oven/bun:1-alpine AS prod-deps
WORKDIR /app

COPY package.json bun.lock ./
COPY apps/web/package.json ./apps/web/
COPY packages/openapi/package.json ./packages/openapi/
COPY packages/tsconfig/package.json ./packages/tsconfig/

RUN --mount=type=cache,id=bun,target=/root/.bun/install/cache \
    bun install --frozen-lockfile --production --ignore-scripts

FROM oven/bun:1-alpine AS builder
WORKDIR /app

COPY package.json bun.lock ./
COPY packages/tsconfig ./packages/tsconfig
COPY packages/openapi ./packages/openapi
COPY apps/web ./apps/web

COPY --from=deps /app/node_modules ./node_modules
COPY --from=deps /app/apps/web/node_modules ./apps/web/node_modules

WORKDIR /app/apps/web
ENV NODE_ENV=production
RUN rm -rf build .react-router && bun run build

FROM oven/bun:1-alpine AS runner
WORKDIR /app

ENV NODE_ENV=production
ENV PORT=3000

COPY package.json bun.lock ./
COPY packages/openapi ./packages/openapi
COPY apps/web/package.json ./apps/web/

COPY --from=prod-deps /app/node_modules ./node_modules
COPY --from=prod-deps /app/apps/web/node_modules ./apps/web/node_modules
COPY --from=builder /app/apps/web/build ./apps/web/build

WORKDIR /app/apps/web

USER bun:bun
EXPOSE 3000

CMD ["bun", "run", "start"]
