# Build the manager binary
FROM golang:1.17 as builder
ARG VER=0.0.0-dev

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY dbaas_support.go dbaas_support.go
COPY apis/ apis/
COPY controllers/ controllers/
COPY internal/ internal/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -ldflags "-X main.operatorVersion=${VER}" -tags 'dbaas' -o manager

FROM registry.access.redhat.com/ubi8-minimal
RUN microdnf -y update && microdnf -y clean all

COPY LICENSE /licenses/LICENSE
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
