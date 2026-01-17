# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /auto-db-backups .

# Runtime stage
FROM alpine:3.20

# Install database clients
RUN apk add --no-cache \
    postgresql16-client \
    mysql-client \
    mongodb-tools \
    ca-certificates \
    tzdata

# Copy the binary from builder
COPY --from=builder /auto-db-backups /auto-db-backups

# Set the entrypoint
ENTRYPOINT ["/auto-db-backups"]
