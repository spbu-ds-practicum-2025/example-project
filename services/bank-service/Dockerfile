FROM golang:1.20 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/bank-service ./cmd/server

FROM gcr.io/distroless/static
COPY --from=builder /bin/bank-service /bin/bank-service
ENTRYPOINT ["/bin/bank-service"]
