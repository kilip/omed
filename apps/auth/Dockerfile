FROM oven/bun:1.4.2-alpine AS build
WORKDIR /app
COPY . .
RUN bun install --frozen-lockfile --ignore-scripts
RUN cd apps/auth && bun build src/main.ts --target=bun --outdir=dist

FROM oven/bun:1.4.2-alpine
WORKDIR /app
ENV NODE_ENV=production
COPY --from=build /app/apps/auth/dist ./
USER bun
EXPOSE 8001
CMD ["bun", "run", "main.js"]
