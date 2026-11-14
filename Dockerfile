# syntax=docker/dockerfile:1.28

FROM --platform=$BUILDPLATFORM golang:1.27.4 AS builder
WORKDIR /src

ENV GOPROXY=https://proxy.golang.org,direct

COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH

RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/go/pkg/mod \
    CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -tags timetzdata -ldflags="-s -w" -o /out/myCalendar ./cmd/myCalendar \
    && mkdir -p /out/data

FROM gcr.io/distroless/static-debian12:nonroot

ARG APP_UID=10001
ARG APP_GID=10001

COPY --from=builder --chown=${APP_UID}:${APP_GID} --chmod=0555 /out/myCalendar /usr/local/bin/myCalendar
COPY --from=builder --chown=${APP_UID}:${APP_GID} /out/data /data

USER ${APP_UID}:${APP_GID}

ENTRYPOINT ["/usr/local/bin/myCalendar"]
