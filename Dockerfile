# Build the manager binary
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Copy the Go Modules manifests
COPY go.mod go.mod
#COPY go.sum go.sum

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY configuration configuration
COPY sync sync

# Build
RUN go mod vendor
RUN go build -mod=vendor -o target/ces-importer


# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM alpine:3.21.2
LABEL maintainer="hello@cloudogu.com" \
      NAME="ces-importer" \
      VERSION="0.0.1"

RUN apk update && apk upgrade && apk --no-cache add bash openssh rsync && \
    addgroup -S -g 1000 ces-importer && \
    adduser -S -h /home/ces-importer -s /bin/bash -G ces-importer -u 1000 ces-importer

USER ces-importer

WORKDIR /
COPY --from=builder /app/target/ces-importer .


ENTRYPOINT ["/ces-importer"]