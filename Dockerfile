# syntax=docker/dockerfile:1.7

# ─── UI builder ───────────────────────────────────────────────────────────────
FROM node:20-alpine AS ui

WORKDIR /src/ui
RUN corepack enable

COPY ui/package.json ui/pnpm-lock.yaml ./
RUN pnpm install --frozen-lockfile

COPY ui/ ./
RUN pnpm run build

# ─── Go builder ───────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS build

ARG VERSION=dev
ARG GIT_COMMIT=none
ARG BUILD_DATE=

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=ui /src/ui/dist/ ./internal/ui/ui-dist/

ENV CGO_ENABLED=0 GOOS=linux
RUN go build \
      -trimpath \
      -ldflags "-s -w \
        -X github.com/drsoft-oss/proxymetrics/internal/cli.Version=${VERSION} \
        -X github.com/drsoft-oss/proxymetrics/internal/cli.GitCommit=${GIT_COMMIT} \
        -X github.com/drsoft-oss/proxymetrics/internal/cli.BuildDate=${BUILD_DATE}" \
      -o /out/proxymetrics \
      ./cmd/proxymetrics

# ─── Runtime ──────────────────────────────────────────────────────────────────
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata \
 && addgroup -S -g 65532 proxymetrics \
 && adduser  -S -u 65532 -G proxymetrics proxymetrics \
 && mkdir -p /data \
 && chown -R proxymetrics:proxymetrics /data

COPY --from=build /out/proxymetrics /usr/local/bin/proxymetrics

USER 65532:65532
WORKDIR /data
VOLUME ["/data"]

EXPOSE 8080 8081

ENTRYPOINT ["/usr/local/bin/proxymetrics"]
CMD ["serve"]
