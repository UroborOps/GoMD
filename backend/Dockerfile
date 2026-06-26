# Build frontend
FROM node:20-alpine AS frontend-builder

WORKDIR /app
COPY frontend/package.json frontend/package-lock.json* ./
RUN npm ci
COPY frontend/. .
RUN npm run build

# Build Go binary with embedded frontend
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install Go dependencies
ENV GOTOOLCHAIN=local
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy Go source code
COPY backend/ .

# Copy frontend dist into the embed location (relative to static.go)
COPY --from=frontend-builder /app/dist ./internal/static/dist

# Build the single binary (embeds frontend)
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /gomd ./cmd/gomd

# Production stage
FROM alpine:3.21

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary
COPY --from=builder /gomd /app/gomd

# Create vault directory
RUN mkdir -p /app/vault

# Expose port
EXPOSE 3000

# Run the server
ENTRYPOINT ["/app/gomd"]
CMD ["--port", "3000", "--vault", "/app/vault"]