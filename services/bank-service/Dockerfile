# Build stage
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags='-w -s' \
    -o /bin/bank-service \
    ./cmd/server/main.go

# Runtime stage
FROM gcr.io/distroless/static-debian12:nonroot

# Copy CA certificates from builder
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy migrations
COPY --from=builder /build/migrations /migrations

# Copy binary
COPY --from=builder /bin/bank-service /bin/bank-service

# Use non-root user
USER nonroot:nonroot

EXPOSE 50051

ENTRYPOINT ["/bin/bank-service"]
