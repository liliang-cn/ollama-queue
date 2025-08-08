# Build stage
FROM golang:1.21-alpine AS builder

# Install git and ca-certificates (needed for dependencies)
RUN apk add --no-cache git ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags='-w -s' -o ollama-queue ./cmd/ollama-queue

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1000 -S appgroup && \
    adduser -u 1000 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/ollama-queue .

# Create data directory with correct permissions
RUN mkdir -p /app/data && \
    chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose default port (if needed for future web interface)
EXPOSE 7125

# Set default environment variables
ENV OLLAMA_HOST=http://host.docker.internal:11434
ENV QUEUE_STORAGE_PATH=/app/data
ENV QUEUE_MAX_WORKERS=4
ENV LOG_LEVEL=info

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ./ollama-queue status --help > /dev/null || exit 1

# Default command
ENTRYPOINT ["./ollama-queue"]
CMD ["--help"]