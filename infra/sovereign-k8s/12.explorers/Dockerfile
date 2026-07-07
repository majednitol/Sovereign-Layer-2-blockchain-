FROM node:22-alpine AS base
ENV PNPM_HOME="/pnpm"
ENV PATH="$PNPM_HOME:$PATH"
RUN corepack enable
WORKDIR /app

FROM base AS builder
# Copy package.json, pnpm-workspace.yaml, and lockfile
COPY package.json pnpm-workspace.yaml pnpm-lock.yaml ./
COPY packages/api-spec ./packages/api-spec
COPY explorer ./explorer

# Install dependencies
RUN --mount=type=cache,id=pnpm,target=/pnpm/store pnpm install --frozen-lockfile --ignore-scripts

# Set Build-Time Env Variables
ARG NEXT_PUBLIC_RPC_URL=http://localhost:26657
ARG NEXT_PUBLIC_API_URL=http://localhost:8082
ARG NEXT_PUBLIC_FAUCET_URL=http://localhost:8000
ARG NEXT_PUBLIC_COSMOS_RPC_URL=https://rpc.yourchain.io
ARG NEXT_PUBLIC_COSMOS_RPC_WS=wss://rpc.yourchain.io/websocket
ARG NEXT_PUBLIC_EVM_RPC_URL=https://evm-rpc.yourchain.io
ARG NEXT_PUBLIC_EVM_RPC_WS=wss://evm-ws.yourchain.io
ARG NEXT_PUBLIC_COSMOS_GRPC=grpc.yourchain.io:9090
ARG NEXT_PUBLIC_COSMOS_REST=https://lcd.yourchain.io:1317
ARG NEXT_PUBLIC_COSMOS_CHAIN_ID=sovereign-1
ARG NEXT_PUBLIC_EVM_CHAIN_ID=7777
ARG NEXT_PUBLIC_CURRENCY_SYMBOL=SLT

ENV NEXT_PUBLIC_RPC_URL=$NEXT_PUBLIC_RPC_URL
ENV NEXT_PUBLIC_API_URL=$NEXT_PUBLIC_API_URL
ENV NEXT_PUBLIC_FAUCET_URL=$NEXT_PUBLIC_FAUCET_URL
ENV NEXT_PUBLIC_COSMOS_RPC_URL=$NEXT_PUBLIC_COSMOS_RPC_URL
ENV NEXT_PUBLIC_COSMOS_RPC_WS=$NEXT_PUBLIC_COSMOS_RPC_WS
ENV NEXT_PUBLIC_EVM_RPC_URL=$NEXT_PUBLIC_EVM_RPC_URL
ENV NEXT_PUBLIC_EVM_RPC_WS=$NEXT_PUBLIC_EVM_RPC_WS
ENV NEXT_PUBLIC_COSMOS_GRPC=$NEXT_PUBLIC_COSMOS_GRPC
ENV NEXT_PUBLIC_COSMOS_REST=$NEXT_PUBLIC_COSMOS_REST
ENV NEXT_PUBLIC_COSMOS_CHAIN_ID=$NEXT_PUBLIC_COSMOS_CHAIN_ID
ENV NEXT_PUBLIC_EVM_CHAIN_ID=$NEXT_PUBLIC_EVM_CHAIN_ID
ENV NEXT_PUBLIC_CURRENCY_SYMBOL=$NEXT_PUBLIC_CURRENCY_SYMBOL

# Build the project
RUN pnpm --filter explorer build

FROM base AS runner
WORKDIR /app
COPY --from=builder /app/package.json /app/pnpm-workspace.yaml /app/pnpm-lock.yaml ./
COPY --from=builder /app/packages/api-spec ./packages/api-spec
COPY --from=builder /app/explorer ./explorer
RUN --mount=type=cache,id=pnpm,target=/pnpm/store pnpm install --frozen-lockfile --prod --ignore-scripts

WORKDIR /app/explorer
EXPOSE 3001
ENV PORT=3001
CMD ["pnpm", "start"]
