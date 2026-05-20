# build/Dockerfile
# Stage 1: Frontend build
FROM node:24-bookworm-slim AS frontend

WORKDIR /src
COPY web/package.json web/package-lock.json ./web/
RUN cd web && npm ci

COPY web ./web
RUN cd web && npm run build

# Stage 2: Go build
FROM golang:1.26.3-bookworm AS builder

RUN apt-get update && apt-get install -y gcc libc6-dev && rm -rf /var/lib/apt/lists/*

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend /src/internal/master/api/frontend_dist ./internal/master/api/frontend_dist
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o /bin/vaultfleet-master ./cmd/master

# Stage 3: Runtime
FROM debian:bookworm-slim

ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates tzdata && rm -rf /var/lib/apt/lists/*

COPY --from=builder /bin/vaultfleet-master /usr/local/bin/vaultfleet-master

VOLUME ["/data"]
EXPOSE 8080

ENTRYPOINT ["vaultfleet-master"]
CMD ["--data-dir", "/data", "--addr", ":8080"]
