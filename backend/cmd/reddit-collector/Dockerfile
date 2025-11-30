# Build stage
FROM golang:1.21-alpine AS builder

# Set working directory
WORKDIR /app

# Install git (needed for go mod download)
RUN apk add --no-cache git

# Copy go mod files from backend root
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the reddit-collector binary
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o reddit-collector ./cmd/reddit-collector

# Final stage
FROM alpine:3.19

# Install ca-certificates for HTTPS requests and timezone data
RUN apk --no-cache add ca-certificates tzdata

# Create app directory
WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/reddit-collector .

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup && \
    chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Run the application
ENTRYPOINT ["./reddit-collector"]
