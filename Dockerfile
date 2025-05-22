# Multi-stage build for DDNSwitch
FROM golang:1.21-alpine AS builder

# Install git and ca-certificates (needed for go mod download)
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
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags "-X main.version=docker" -o ddnswitch .

# Final stage
FROM alpine:latest

# Install ca-certificates for HTTPS requests
RUN apk --no-cache add ca-certificates

# Create a non-root user
RUN addgroup -g 1001 -S ddnswitch && \
    adduser -u 1001 -S ddnswitch -G ddnswitch

# Set working directory
WORKDIR /home/ddnswitch

# Copy binary from builder stage
COPY --from=builder /app/ddnswitch /usr/local/bin/ddnswitch

# Change ownership
RUN chown ddnswitch:ddnswitch /usr/local/bin/ddnswitch

# Switch to non-root user
USER ddnswitch

# Create .ddnswitch directory
RUN mkdir -p /home/ddnswitch/.ddnswitch

# Set entrypoint
ENTRYPOINT ["ddnswitch"]

# Default command
CMD ["--help"]