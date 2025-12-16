# Build the manager binary
FROM sfbrdevhelmweacr.azurecr.io/golang:1.25.4_echo as builder

WORKDIR /workspace
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY webhook/ webhook/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o admctl-webhook main.go

FROM uipathsec.azurecr.io/uipath/parent/ubi9-v2/base:1.0

# Add Echo label for tracking
LABEL org.opencontainers.builder.image.vendor="echohq.com"

WORKDIR /
COPY --from=builder /workspace/admctl-webhook .

ENTRYPOINT ["/admctl-webhook"]
