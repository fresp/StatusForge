# Multi-stage build for StatusForge unified server
FROM node:20-alpine AS frontend-build

WORKDIR /app

# Copy frontend files
COPY apps/web/ .

# Install dependencies and build the frontend
RUN npm ci
RUN npm run build

# Build the Go backend
FROM golang:1.26-alpine AS backend-build

ARG TARGETARCH

# Install build tools
RUN apk add --no-cache ca-certificates git musl-dev gcc libcap-dev

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Copy built frontend assets
COPY --from=frontend-build /app/dist /app/internal/embed/dist

# Build binary (auto-detect arch for M1/ARM64 support)
RUN CGO_ENABLED=0 GOARCH=$TARGETARCH go build -a -installsuffix cgo -o server cmd/server/main.go

# Final minimal image
FROM alpine:latest

# Upgrade packages and install ca-certificates
RUN apk upgrade --no-cache && \
    apk --no-cache add ca-certificates

RUN adduser -D -u 1001 appuser

WORKDIR /app

COPY --from=backend-build --chown=appuser:appuser /app/server /app/server
RUN chmod +x /app/server

USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/server"]