# ---- Install & Build (monorepo context) ----
FROM oven/bun:1 AS builder

WORKDIR /app

# Copy root workspace files for dependency resolution
COPY package.json bun.lock turbo.json ./
COPY packages/ ./packages/
COPY apps/dash/ ./apps/dash/

# Install all dependencies (workspace-aware)
RUN bun install --frozen-lockfile

# Build workspace packages first, then dash
ARG VITE_AUTH_URL
ARG VITE_PUBLIC_URL
RUN bun run --cwd packages/api build && \
    bun run --cwd packages/better-auth build && \
    bun run --cwd apps/dash build

# ---- Runtime: serve static SPA with nginx ----
FROM nginx:1-alpine

# Remove default config
RUN rm /etc/nginx/conf.d/default.conf

COPY apps/dash/nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=builder /app/apps/dash/build/client /usr/share/nginx/html

EXPOSE 3001

CMD ["nginx", "-g", "daemon off;"]
