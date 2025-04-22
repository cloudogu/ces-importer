# Build the manager binary
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go sources
COPY *.go .
COPY api api
COPY configuration configuration
COPY cron cron
COPY systeminfo systeminfo
COPY sync sync
COPY validate validate

# Build
RUN go mod vendor
RUN go build -mod=vendor -o target/ces-importer


FROM alpine:3.21.3
LABEL maintainer="hello@cloudogu.com" \
      NAME="ces-importer" \
      VERSION="0.0.1"

RUN apk update && apk upgrade && apk --no-cache add bash openssh rsync && \
    addgroup -S -g 65532 ces-importer && \
    adduser -S -h /home/ces-importer -s /bin/bash -G ces-importer -u 65532 ces-importer

# note that this app will start deployments that must run as root
# use numeric IDs to avoid clash with runAsNonRoot so that k8s can validate it as non-root user
USER 65532:65532

WORKDIR /
COPY --from=builder /app/target/ces-importer .


ENTRYPOINT ["/ces-importer"]