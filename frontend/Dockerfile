FROM node:22-alpine AS build

WORKDIR /app

RUN npm install -g bun

COPY package.json bun.lock* ./
RUN bun install --frozen-lockfile --ignore-scripts

COPY . .

RUN NODE_OPTIONS="--max-old-space-size=4096" bun run build

FROM oven/bun:1

WORKDIR /app

COPY --from=build /app/.output /app

EXPOSE 3000/tcp

ENTRYPOINT ["bun", "--bun", "run", "/app/server/index.mjs"]