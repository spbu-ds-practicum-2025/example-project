FROM golang:1.20 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/api-gateway ./cmd/server

FROM gcr.io/distroless/static
COPY --from=builder /bin/api-gateway /bin/api-gateway
ENTRYPOINT ["/bin/api-gateway"]
