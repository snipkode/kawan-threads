# syntax=docker/dockerfile:1

# ------------------------------------------------------------------
# Stage 1: build the Go binary (api + worker share one builder)
# ------------------------------------------------------------------
FROM golang:1.23-alpine AS builder
WORKDIR /src

RUN apk add --no-cache ca-certificates tzdata

ARG TARGETOS=linux
ARG TARGETARCH=amd64

# Cache module downloads.
COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal

# Build both binaries.
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-s -w" -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -ldflags="-s -w" -o /out/worker ./cmd/worker

# ------------------------------------------------------------------
# Stage 2: api runtime image
# ------------------------------------------------------------------
FROM alpine:3.20 AS api
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S kawan && adduser -S kawan -G kawan
COPY --from=builder /out/api /usr/local/bin/kawan-api
USER kawan
EXPOSE 8080
ENTRYPOINT ["kawan-api"]

# ------------------------------------------------------------------
# Stage 3: worker runtime image
# ------------------------------------------------------------------
FROM alpine:3.20 AS worker
RUN apk add --no-cache ca-certificates tzdata && \
    addgroup -S kawan && adduser -S kawan -G kawan
COPY --from=builder /out/worker /usr/local/bin/kawan-worker
USER kawan
ENTRYPOINT ["kawan-worker"]