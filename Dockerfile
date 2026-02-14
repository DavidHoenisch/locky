# Build admin UI (Svelte)
FROM node:20-alpine AS ui-builder
WORKDIR /app
COPY . .
WORKDIR /app/ui
RUN npm ci && npm run build

# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app

# Install dependencies
RUN apk add --no-cache git

# Copy go mod files and source
COPY go.mod go.sum ./
COPY auth/ ./auth/
COPY cmd/ ./cmd/

# Overlay built admin UI so embed_ui can include it
COPY --from=ui-builder /app/auth/ui/dist ./auth/ui/dist

RUN go mod download

# Run go mod tidy to fix dependencies
RUN go mod tidy

# Build with embed_ui so the admin UI is served when ENABLE_ADMIN_UI is set
RUN CGO_ENABLED=0 GOOS=linux go build -tags embed_ui -o locky ./cmd/locky

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
