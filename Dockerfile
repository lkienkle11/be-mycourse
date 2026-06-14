# Multi-stage CGO build — mirrors .github/workflows/deploy-dev.yml (libvips + HDF5).
ARG STAGE=dev

FROM golang:1.25.0-bookworm AS builder
ARG STAGE
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
    libvips-dev libhdf5-dev pkg-config \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=1 go build -trimpath -ldflags="-s -w" -o /out/mycourse-io-be-${STAGE} .

FROM debian:bookworm-slim AS runtime
ARG STAGE
ENV STAGE=${STAGE} \
    CGO_ENABLED=1 \
    MIGRATE=1
RUN apt-get update && DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends \
    ca-certificates curl \
    libvips42 libhdf5-103-1 \
    && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=builder /out/mycourse-io-be-${STAGE} /app/mycourse-io-be
COPY config/ /app/config/
COPY migrations/ /app/migrations/
EXPOSE 8080
HEALTHCHECK --interval=10s --timeout=5s --start-period=30s --retries=5 \
    CMD curl -fsS http://127.0.0.1:8080/api/v1/health || exit 1
CMD ["/app/mycourse-io-be"]
