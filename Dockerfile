# Build the manager binary
FROM golang:1.22 AS builder
WORKDIR /workspace

# Copy the Go Modules manifests
COPY go.mod go.sum* ./
RUN go mod download || true

# Copy the go source
COPY cmd/ cmd/
COPY pkg/ pkg/
COPY api/ api/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o manager cmd/controller/main.go

# Production minimal distroless image
FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /workspace/manager .
USER 65532:65532

ENTRYPOINT ["/manager"]
