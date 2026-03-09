# Multi-stage build for StatusForge unified server
FROM node:20-alpine AS frontend-build

WORKDIR /app

# Copy frontend files
COPY apps/web/ .

# Install dependencies and build the frontend
RUN npm ci
RUN npm run build

# Build the Go backend
FROM golang:1.21-alpine AS backend-build

# Install build tools
RUN apk add --no-cache ca-certificates git musl-dev gcc libcap-dev

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Copy built frontend assets
COPY --from=frontend-build /app/dist /app/apps/web/dist

# Build static binary
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o server cmd/server/main.go

# Create user for running the binary
RUN adduser -D -u 1001 appuser

# Final minimal image
FROM alpine:latest

# Install ca-certificates
RUN apk --no-cache add ca-certificates

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