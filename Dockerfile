# syntax=docker/dockerfile:1

# -----------------------------------------------------------------------------
# Build stage
# -----------------------------------------------------------------------------
FROM golang:1.26-alpine AS builder

WORKDIR /src

RUN apk add --no-cache ca-certificates git tzdata

# Module files first for better layer caching
COPY go.mod go.sum* ./
RUN go mod download

COPY . .

ARG VERSION=0.1.0
ARG COMMIT=unknown
ARG BUILT_AT=unknown

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w \
      -X github.com/magicvr/allinme.core-api/pkg/version.Version=${VERSION} \
      -X github.com/magicvr/allinme.core-api/pkg/version.Commit=${COMMIT} \
      -X github.com/magicvr/allinme.core-api/pkg/version.BuiltAt=${BUILT_AT}" \
    -o /out/allinme.core-api ./cmd/server

# -----------------------------------------------------------------------------
# Runtime stage (minimal, non-root)
# -----------------------------------------------------------------------------
FROM alpine:3.21 AS runtime

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S app \
    && adduser -S -G app -H -s /sbin/nologin app

WORKDIR /app

COPY --from=builder /out/allinme.core-api /app/allinme.core-api

USER app

ENV APP_ENV=production \
    HTTP_ADDR=:8080 \
    LOG_LEVEL=info

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -qO- http://127.0.0.1:8080/healthz >/dev/null || exit 1

ENTRYPOINT ["/app/allinme.core-api"]
