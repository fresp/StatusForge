# StatusForge

A production-ready, self-hosted status page and monitoring platform — similar to Atlassian Statuspage, BetterStack, and UptimeRobot. Single-tenant, fully open-source.

This unified server combines API, Worker, and embeds Web frontend in a single binary using goroutines.

![Status Platform](https://img.shields.io/badge/Go-1.21-00ADD8?logo=go) ![React](https://img.shields.io/badge/React-18-61DAFB?logo=react) ![MongoDB](https://img.shields.io/badge/MongoDB-7-47A248?logo=mongodb) ![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?logo=docker)

## Features

- **Public Status Page** — Atlassian-style status page with 90-day uptime history bars, active incidents, and scheduled maintenance
- **Admin CMS Dashboard** — Manage components, incidents, maintenance windows, monitors, and subscribers
- **Automated Monitoring** — HTTP, TCP, DNS, and ICMP ping checks with configurable intervals
- **Auto Incident Management** — Automatically creates incidents after 3 consecutive failures; auto-resolves when healthy
- **Real-time Updates** — WebSocket push for instant status changes without page refresh
- **Incident Timeline** — Full update history with status transitions (investigating → identified → monitoring → resolved)
- **Email Subscribers** — Collect and manage subscriber emails for status notifications
- **JWT Authentication** — Secure admin-only routes with bcrypt password hashing
- **90-Day Uptime History** — Daily uptime aggregation with color-coded bars per component
- **Single Container Deployment** — Unified binary in a single Docker container

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend API | Go 1.21, Gin, JWT, bcrypt |
| Monitoring Worker | Go, goroutines, ICMP/TCP/DNS/HTTP with graceful shutdown |
| Frontend | React 18, Vite, TypeScript, Tailwind CSS (embedded in Go binary) |
| Database | MongoDB 7 |
| Cache / Pub-Sub | Redis |
| Real-time | WebSocket (gorilla/websocket) |
| Deployment | Single Docker Image |

## Architecture

This implementation consolidates the previous 3 separate services (API, Worker, Web) into a single binary:

- **Unified Server** (`cmd/server/main.go`) - runs all services concurrently in goroutines
- **Embedded Frontend** - React frontend statically embedded in the Go binary
- **Toggleable Worker** - Enable/disable via `ENABLE_WORKER` environment variable
- **Health Check** - `/health` endpoint with database connectivity verification
- **Graceful Shutdown** - 30s timeout with proper goroutine cleanup

## Quick Start

### Prerequisites

- Docker and Docker Compose

### 1. Prepare your configuration

```bash
cp .env.example .env
# Edit .env if you want to change defaults
```

### 2. Run the unified platform

```bash
docker compose up --build
```

This starts:
- **Server** on port `8080` - serves API, WebSocket, and embedded Web frontend
- **MongoDB** on port `27017`
- **Redis** on port `6379`

### 3. (Optional) Seed sample data

```bash
# After services are running
go run scripts/seed.go
```

This creates sample components, subcomponents, and a resolved incident with full timeline.

### 4. Access the platform

| URL | Description |
|-----|-------------|
| http://localhost:8080 | Public status page (served from embedded frontend) |
| http://localhost:8080/admin/login | Admin login |
| http://localhost:8080/api | API base |

**Default admin credentials:**
```
Email:    admin@statusplatform.com
Password: admin123
```

> Change these in `.env` before deploying to production.

## Configuration Options

| Environment Variable | Description | Default |
|---------------------|-------------|---------|
| `ENABLE_WORKER` | Whether to run the monitoring worker | `true` |
| `PORT` | Server listening port | `8080` |
| `MONGO_URI` | MongoDB connection string | `mongodb://mongo:27017` |
| `MONGO_DB_NAME` | Database name | `statusplatform` |
| `REDIS_ADDR` | Redis connection address | `redis:6379` | 
| `JWT_SECRET` | Secret for JWT generation | `change-me-in-production` |

## Health Check

The server exposes a health check endpoint to verify database connectivity:

- `GET http://localhost:8080/health`
- Returns 200 if healthy, 503 if MongoDB or Redis are unreachable
- Response includes detailed DB connectivity status

MIT