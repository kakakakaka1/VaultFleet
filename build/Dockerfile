# build/Dockerfile
# Stage 1: Frontend build
FROM node:24-bookworm-slim AS frontend

WORKDIR /src
COPY web/package.json web/package-lock.json ./web/
RUN --mount=type=cache,target=/root/.npm cd web && npm ci

COPY web ./web
RUN cd web && npm run build

# Stage 2: Go build
FROM golang:1.26.3-bookworm AS builder

WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

ARG VERSION=dev
COPY . .
COPY --from=frontend /src/internal/master/api/frontend_dist ./internal/master/api/frontend_dist
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w -X main.version=${VERSION}" -o /bin/vaultfleet-master ./cmd/master

# Stage 3: Runtime
FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tzdata rclone iproute2 && rm -rf /var/lib/apt/lists/*

COPY --from=builder /bin/vaultfleet-master /usr/local/bin/vaultfleet-master
RUN printf '%s\n' \
  '#!/bin/sh' \
  'set -e' \
  'if ! getent hosts host.docker.internal >/dev/null 2>&1; then' \
  '  gateway="$(ip route | awk '\''/default/ {print $3; exit}'\'')" || true' \
  '  if [ -n "$gateway" ]; then echo "$gateway host.docker.internal host-gateway" >> /etc/hosts; fi' \
  'fi' \
  'if [ "${1#-}" != "$1" ]; then set -- vaultfleet-master "$@"; fi' \
  'exec "$@"' \
  > /usr/local/bin/vaultfleet-entrypoint && chmod +x /usr/local/bin/vaultfleet-entrypoint

VOLUME ["/data"]
EXPOSE 8080

ENTRYPOINT ["vaultfleet-entrypoint"]
CMD ["--data-dir", "/data", "--addr", ":8080"]
