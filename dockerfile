ARG GO_VERSION=1.23
ARG APP_NAME=go-serverci

FROM golang:${GO_VERSION}-alpine AS builder
ARG APP_NAME

RUN apk add --no-cache build-base
WORKDIR /app

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -o /out/${APP_NAME} ./cmd

FROM debian:bookworm
ARG APP_NAME
ENV APP_NAME=${APP_NAME}

RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        texlive-full latexmk ghostscript poppler-utils \
        ca-certificates curl wget make && \
    rm -rf /var/lib/apt/lists/*

COPY --from=builder /out/${APP_NAME} /usr/local/bin/${APP_NAME}

WORKDIR /app
COPY . /app

EXPOSE 8080
CMD ["sh", "-lc", "$APP_NAME --serve"]
