# Build the manager binary
FROM golang:1.22-alpine AS builder

WORKDIR /workspace

# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum

# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY configuration configuration
COPY prometheus prometheus
COPY proxy proxy
COPY serviceaccount serviceaccount

# Build
RUN go mod vendor
RUN go build -mod=vendor -o target/k8s-prometheus-auth


# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM gcr.io/distroless/static:nonroot
LABEL maintainer="hello@cloudogu.com" \
      NAME="k8s-prometheus-auth" \
      VERSION="57.1.1-7"

WORKDIR /
COPY --from=builder /workspace/target/k8s-prometheus-auth .

# the linter has a problem with the valid colon-syntax
# dockerfile_lint - ignore
USER 65532:65532

EXPOSE 8086
EXPOSE 8087

ENTRYPOINT ["/k8s-prometheus-auth"]