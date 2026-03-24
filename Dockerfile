# =========================
# Frontend Build
# =========================
FROM node:20-alpine AS frontend-build

WORKDIR /app

# Copy frontend files
COPY apps/web/ .

# Install dependencies and build the frontend
RUN npm ci
RUN npm run build


# =========================
# Backend Build
# =========================
FROM golang:1.26-alpine AS backend-build

ARG TARGETARCH

# Install build tools
RUN apk add --no-cache ca-certificates git

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Copy built frontend assets
COPY --from=frontend-build /app/dist /app/internal/embed/dist

# Build binary
RUN CGO_ENABLED=0 GOARCH=$TARGETARCH go build -o server cmd/server/main.go


# =========================
# Final Image
# =========================
FROM alpine:latest

# Install runtime deps
RUN apk upgrade --no-cache && \
    apk add --no-cache ca-certificates

# ✅ Create non-root user FIRST
RUN adduser -D -u 1001 appuser

# Set working directory
WORKDIR /app

# ✅ Copy binary with correct ownership
COPY --from=backend-build --chown=appuser:appuser /app/server /app/server

# ✅ Prepare writable directory for SQLite
RUN mkdir -p /app/data && chown appuser:appuser /app/data

# Allow setup flow to persist .env in /app as non-root
RUN chown appuser:appuser /app

# Ensure binary executable
RUN chmod +x /app/server

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget -qO- http://localhost:8080/health || exit 1

# Start server
ENTRYPOINT ["/app/server"]
