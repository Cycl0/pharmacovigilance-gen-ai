# Use a Go image with modules support
FROM golang:1.20 AS builder

WORKDIR /app

# Copy go.mod and go.sum first to leverage Docker cache
COPY go.mod go.sum ./

# Download dependencies
RUN go mod tidy && \
    go mod verify && \
    go mod download

# Copy the entire source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o main .

# Use a minimal runtime image
FROM alpine:3.19

# Create non-root user and directory
RUN adduser -D -g '' appuser
WORKDIR /app
RUN chown appuser:appuser /app
USER appuser

# Copy necessary files
COPY --from=builder --chown=appuser:appuser /app/main .
COPY --from=builder --chown=appuser:appuser /app/.env .

CMD ["./main"]
