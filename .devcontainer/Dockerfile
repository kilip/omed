# syntax=docker/dockerfile:1
FROM golang:1.25-bookworm AS golang

FROM mcr.microsoft.com/devcontainers/base:bookworm

ARG NODE_MAJOR=24

# ---- system deps (build-essential dibutuhkan CGO go-sqlite3 & crate native) ----
RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get install -y --no-install-recommends \
       build-essential pkg-config libssl-dev libsqlite3-dev \
       postgresql-client unzip ca-certificates curl gnupg \
    && rm -rf /var/lib/apt/lists/*

# ---- Node ----
RUN curl -fsSL https://deb.nodesource.com/setup_${NODE_MAJOR}.x | bash - \
    && apt-get install -y --no-install-recommends nodejs \
    && rm -rf /var/lib/apt/lists/*

# ---- Playwright system libs (browser di-install per-user di setup.sh) ----
RUN npx -y playwright install-deps chromium && rm -rf /var/lib/apt/lists/*

# ---- Go ----
COPY --from=golang /usr/local/go /usr/local/go

ENV GOPATH=/home/vscode/go \
    CARGO_HOME=/home/vscode/.cargo \
    RUSTUP_HOME=/home/vscode/.rustup \
    BUN_INSTALL=/home/vscode/.bun
ENV PATH=/usr/local/go/bin:/home/vscode/go/bin:/home/vscode/.cargo/bin:/home/vscode/.bun/bin:${PATH}

USER vscode

# ---- Rust ----
RUN curl -fsSL https://sh.rustup.rs | sh -s -- -y --profile minimal \
      --component clippy --component rustfmt --component rust-analyzer

# ---- Bun ----
RUN curl -fsSL https://bun.sh/install | bash

# ---- Go tools ----
RUN go install github.com/air-verse/air@latest \
    && go install entgo.io/ent/cmd/ent@latest \
    && go install github.com/swaggo/swag/v2/cmd/swag@latest \
    && go install go.uber.org/mock/mockgen@latest \
    && go install honnef.co/go/tools/cmd/staticcheck@latest \
    && go install golang.org/x/tools/gopls@latest \
    && go install github.com/go-delve/delve/cmd/dlv@latest

# ---- sqlx-cli (compile sekali, ke-cache di layer) ----
RUN cargo install sqlx-cli --locked --no-default-features --features rustls,postgres

# dir cache untuk named volume (biar owner-nya vscode, bukan root)
RUN mkdir -p ~/go/pkg ~/.cargo/registry ~/.bun/install/cache

WORKDIR /workspaces/omed
