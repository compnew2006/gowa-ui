# Build stage
FROM golang:1.24.5-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o whatomate ./cmd/whatomate

# Final stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create a non-root user and group
RUN addgroup -S whatomate && adduser -S whatomate -G whatomate

# Copy binary from builder
COPY --from=builder /app/whatomate .

# Copy config example (will be overridden by env vars in production)
COPY --from=builder /app/config.example.toml ./config.toml

# Create uploads directory and set ownership
RUN mkdir -p /app/uploads && chown -R whatomate:whatomate /app

# Switch to non-root user
USER whatomate

# Expose port
EXPOSE 8080

# Run the binary
CMD ["./whatomate", "server", "-config", "config.toml", "-migrate"]
