FROM --platform=$BUILDPLATFORM golang:1.26.0 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETOS
ARG TARGETARCH

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -v -o app ./cmd/vihren


FROM oven/bun:1.1.34 AS ui-build

WORKDIR /src/frontend

COPY frontend/package.json frontend/bun.lock ./
RUN bun install --no-save

COPY frontend/ ./
RUN bun run build


FROM debian:12.10-slim

ARG TARGETARCH

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl perl && \
    rm -rf /var/lib/apt/lists/*

RUN mkdir -p /app /data/flamedb

COPY --from=build /src/app /app/vihren
COPY --from=ui-build /src/ui /app/ui
COPY sql /app/sql
COPY scripts /app/scripts
COPY replace.yaml /app/replace.yaml

ARG CHDB_VERSION=v4.0.2

RUN set -e; \
    echo "TARGETARCH=${TARGETARCH}"; \
    if [ "${TARGETARCH}" = "arm64" ]; then platform="linux-aarch64-libchdb.tar.gz"; else platform="linux-x86_64-libchdb.tar.gz"; fi; \
    url="https://github.com/chdb-io/chdb/releases/download/${CHDB_VERSION}/${platform}"; \
    echo "Downloading libchdb from ${url}"; \
    curl -L -o /tmp/libchdb.tar.gz "${url}"; \
    tar -xzf /tmp/libchdb.tar.gz -C /usr/local/lib; \
    chmod +x /usr/local/lib/libchdb.so; \
    rm -f /tmp/libchdb.tar.gz

ENV CHDB_LIB_PATH=/usr/local/lib/libchdb.so
ENV VIHREN_DB_FILENAME=/data/flamedb/chdb
ENV VIHREN_INDEXER_NORMALIZER_PATH=/app/replace.yaml

RUN useradd -m -s /bin/bash -u 888 non_root && \
    chown -R non_root:non_root /data /app

USER non_root


WORKDIR /app

EXPOSE 8080
ENTRYPOINT ["/app/vihren"]
