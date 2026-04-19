# syntax=docker/dockerfile:1.7

# -----------------------------------------------------------------------------
# Build stage
# -----------------------------------------------------------------------------
ARG GO_VERSION=1.26
FROM --platform=${BUILDPLATFORM} golang:${GO_VERSION}-alpine AS builder

RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /src

# Prime module cache — copy only manifests first for maximum layer reuse.
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH
ARG VERSION=dev

# Fully static binary: CGO off, stripped symbols, reproducible paths.
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} \
    go build \
        -trimpath \
        -tags=netgo,osusergo \
        -ldflags="-s -w -X gokit/pkg/version.Version=${VERSION}" \
        -o /out/api ./cmd/http

# -----------------------------------------------------------------------------
# Runtime stage — distroless static, non-root by default.
# -----------------------------------------------------------------------------
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

WORKDIR /app

# Runtime assets loaded via genkit.WithPromptDir("./prompts").
COPY --from=builder --chown=nonroot:nonroot /src/prompts ./prompts
COPY --from=builder /out/api /app/api

USER nonroot:nonroot

EXPOSE 4040

ENV SERVER_HOST=0.0.0.0 \
    SERVER_PORT=4040 \
    LOG_FORMAT=json

ENTRYPOINT ["/app/api"]
