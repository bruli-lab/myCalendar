# syntax=docker/dockerfile:1.20

############################
# Etapa de build ARM64
############################
FROM --platform=$BUILDPLATFORM golang:1.26.5 AS builder
WORKDIR /src

ENV GOPROXY=https://proxy.golang.org,direct

# 1) Deps (capa estable + cache)
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# 2) Codi
COPY . .

ARG TARGETOS
ARG TARGETARCH

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o /out/myCalendar ./cmd/myCalendar
# --- runtime ---
FROM alpine:3.24

RUN apk add --no-cache tzdata
# copia el teu binari
COPY --from=builder /out/myCalendar /usr/local/bin/myCalendar

ARG APP_UID=10001
ARG APP_GID=10001

RUN apk add --no-cache tzdata \
    && addgroup \
        -S \
        -g "${APP_GID}" \
        app \
    && adduser \
        -S \
        -D \
        -H \
        -u "${APP_UID}" \
        -G app \
        app

COPY --from=builder \
    --chown=${APP_UID}:${APP_GID} \
    --chmod=0555 \
    /out/myCalendar \
    /usr/local/bin/myCalendar

USER ${APP_UID}:${APP_GID}

ENTRYPOINT ["/usr/local/bin/myCalendar"]
