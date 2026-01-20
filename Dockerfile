# Build Stage
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# Build both binaries
RUN go build -o /bin/gateway ./cmd/gateway
RUN go build -o /bin/storage ./cmd/storage

# Run Stage (We use a tiny Alpine Linux image)
FROM alpine:latest
WORKDIR /app
# Copy binaries from builder
COPY --from=builder /bin/gateway /app/gateway
COPY --from=builder /bin/storage /app/storage
# Create storage directory
RUN mkdir -p /data