# Development Dockerfile with live reloading
FROM golang:1.21.3-alpine

# Install ca-certificates for SSL/TLS
RUN apk --no-cache add ca-certificates

WORKDIR /app

# Expose port (will be overridden by docker-compose)
EXPOSE 8080

# Default command - will be overridden by docker-compose
CMD ["go", "run", "cmd/main.go"]
