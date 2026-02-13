# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files
COPY go.mod go.sum ./
COPY auth/ ./auth/
RUN go mod download

# Copy source code
COPY . .

# Run go mod tidy to fix dependencies
RUN go mod tidy

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o locky ./cmd/locky

# Final stage
FROM alpine:latest

WORKDIR /app

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates

# Copy binary from builder
COPY --from=builder /app/locky .

# Expose port
EXPOSE 8080

# Run the binary
CMD ["./locky"]
