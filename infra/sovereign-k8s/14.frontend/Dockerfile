FROM node:22-alpine AS base
ENV PNPM_HOME="/pnpm"
ENV PATH="$PNPM_HOME:$PATH"
RUN corepack enable
WORKDIR /app

FROM base AS builder
# Copy package.json, pnpm-workspace.yaml, and lockfile
COPY package.json pnpm-workspace.yaml pnpm-lock.yaml ./
COPY packages/api-spec ./packages/api-spec
COPY frontend ./frontend

# Install dependencies
RUN --mount=type=cache,id=pnpm,target=/pnpm/store pnpm install --frozen-lockfile --ignore-scripts

# Build the project
RUN pnpm --filter frontend build

FROM base AS runner
WORKDIR /app
COPY --from=builder /app/package.json /app/pnpm-workspace.yaml /app/pnpm-lock.yaml ./
COPY --from=builder /app/packages/api-spec ./packages/api-spec
COPY --from=builder /app/frontend ./frontend
RUN --mount=type=cache,id=pnpm,target=/pnpm/store pnpm install --frozen-lockfile --prod --ignore-scripts

WORKDIR /app/frontend
EXPOSE 3000
ENV PORT=3000
CMD ["pnpm", "start"]
