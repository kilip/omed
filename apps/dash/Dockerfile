FROM oven/bun:1.4.2-alpine AS build
WORKDIR /app
COPY . .
RUN bun install --frozen-lockfile --ignore-scripts
RUN cd apps/dash && bun run build

FROM oven/bun:1.4.2-alpine AS prod-deps
WORKDIR /app
COPY . .
RUN bun install --frozen-lockfile --production --ignore-scripts

FROM node:24-alpine
WORKDIR /app
ENV NODE_ENV=production PORT=3000

COPY --from=prod-deps /app/node_modules ./node_modules
COPY --from=prod-deps /app/packages ./packages
COPY --from=build /app/apps/dash/package.json ./apps/dash/package.json
COPY --from=build /app/apps/dash/build ./apps/dash/build
COPY --from=prod-deps /app/apps/dash/node_modules ./apps/dash/node_modules

WORKDIR /app/apps/dash
USER node
EXPOSE 3000
CMD ["node", "./node_modules/@react-router/serve/bin.cjs", "./build/server/index.js"]
