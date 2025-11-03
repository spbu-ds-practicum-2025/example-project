FROM golang:1.20 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/bank-card-adapter ./cmd/server

FROM gcr.io/distroless/static
COPY --from=builder /bin/bank-card-adapter /bin/bank-card-adapter
ENTRYPOINT ["/bin/bank-card-adapter"]
