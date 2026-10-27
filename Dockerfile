# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

# SQLite requires gcc/musl-dev for cgo
RUN apk add --no-cache gcc musl-dev git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o /etsy-monitor ./cmd/etsy-monitor

# Runner stage
FROM alpine:3.20

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata sqlite-libs

COPY --from=builder /etsy-monitor /app/etsy-monitor
COPY --from=builder /app/config.example.yaml /app/config.example.yaml

ENTRYPOINT ["/app/etsy-monitor"]
