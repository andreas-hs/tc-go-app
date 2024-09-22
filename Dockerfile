# Base image for both dev and prod
FROM golang:1.23.1-alpine AS base
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

# Debug layer with Delve and reflex for rebuild
FROM base AS debug
RUN go install github.com/go-delve/delve/cmd/dlv@latest
RUN go install github.com/cespare/reflex@latest

FROM debug AS dev
COPY . .
CMD reflex -r '\.go$' -- sh -c 'go build -gcflags "all=-N -l" -o app ./cmd/app && dlv --listen=:2345 --headless=true --api-version=2 --accept-multiclient exec ./app'

# Prod stage
FROM base AS prod
COPY . .
RUN go build -o app ./cmd/app
CMD ["./app"]
