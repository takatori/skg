# syntax=docker/dockerfile:1
FROM golang:bookworm AS base
ENV GOTOOLCHAIN=auto

WORKDIR /app

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=bind,source=go.sum,target=go.sum \
    --mount=type=bind,source=go.mod,target=go.mod \
    go mod download -x \
    && apt-get update

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=bind,target=. \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldfloags="-s -w" -o /bin/main ./cmd/main.go


FROM golang:bookworm AS dev
ENV GOTOOLCHAIN=auto

WORKDIR /app

RUN go install github.com/air-verse/air@latest
CMD ["air", "-c", ".air.toml"]
