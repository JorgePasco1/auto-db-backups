# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install build dependencies including gcc for CGO
RUN apk add --no-cache git gcc musl-dev

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application with CGO enabled for proper crypto support
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-w -s" -o /auto-db-backups .

# Runtime stage
FROM alpine:3.20

# Install database clients, SSL certificates, and required libraries
RUN apk add --no-cache \
    postgresql16-client \
    mysql-client \
    mongodb-tools \
    ca-certificates \
    tzdata \
    libc6-compat && \
    update-ca-certificates

# Copy the binary from builder
COPY --from=builder /auto-db-backups /auto-db-backups

# Set the entrypoint
ENTRYPOINT ["/auto-db-backups"]
