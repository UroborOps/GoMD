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

ENV GOTOOLCHAIN=auto
COPY backend/go.mod backend/go.sum ./
RUN go mod download

COPY backend/ .

COPY --from=frontend-builder /app/dist ./internal/static/dist
ARG TARGETOS
ARG TARGETARCH

RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH:-amd64} go build -ldflags="-s -w" -o /gomd ./cmd/gomd

# Production stage
FROM alpine:3.21

RUN apk --no-cache add ca-certificates git openssh-client

WORKDIR /app

COPY --from=builder /gomd /app/gomd

RUN mkdir -p /app/vault

EXPOSE 3000

ENTRYPOINT ["/app/gomd"]
CMD ["--port", "3000", "--vault", "/app/vault"]
