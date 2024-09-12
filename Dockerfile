# Build stage
FROM golang:1.23.1-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o app ./cmd/app

# Final stage
FROM alpine:latest

WORKDIR /root/
COPY --from=builder /app/app .

CMD ["./app"]
