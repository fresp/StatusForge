# StatusForge

A production-ready, self-hosted status page and monitoring platform — similar to Atlassian Statuspage, BetterStack, and UptimeRobot. Single-tenant, fully open-source.

This unified server combines API, Worker, and embeds Web frontend in a single binary using goroutines.

## 🚀 Features

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

## 🛠️ Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend API | Go 1.21, Gin, JWT, bcrypt |
| Monitoring Worker | Go, goroutines, ICMP/TCP/DNS/HTTP with graceful shutdown |
| Frontend | React 18, Vite, TypeScript, Tailwind CSS (embedded in Go binary) |
| Database | MongoDB 7 |
| Cache / Pub-Sub | Redis |
| Real-time | WebSocket (gorilla/websocket) |
| Deployment | Single Docker Image |

## 🏗️ Architecture

This implementation consolidates the previous 3 separate services (API, Worker, Web) into a single binary:

- **Unified Server** (`cmd/server/main.go`) - runs all services concurrently in goroutines
- **Embedded Frontend** - React frontend statically embedded in the Go binary
- **Toggleable Worker** - Enable/disable via `ENABLE_WORKER` environment variable
- **Health Check** - `/health` endpoint with database connectivity verification
- **Graceful Shutdown** - 30s timeout with proper goroutine cleanup

## 📸 Screenshots

### Public Status Page
![Public Status Page](docs/images/public-statuspage.jpeg)

Clean, professional status page showing component health and incident history

### Admin Dashboard
![Admin Dashboard](docs/images/admin-dashboard.png)

Main admin interface with navigation and quick overview

### Component Management
![Component Management](docs/images/admin-component.png)

Create and manage components and their sub-components

### Incident Timeline
![Incident Timeline](docs/images/admin-incident.png)

Full incident lifecycle with status updates and timeline

### Monitoring Overview
![Monitoring Overview](docs/images/admin-monitoring.png)

Real-time monitoring status and uptime metrics

### Maintenance Scheduling
![Maintenance Scheduling](docs/images/admin-maintenance.png)

Schedule and manage planned maintenance windows

## 📋 Dependencies

- **Backend**: Go 1.21+
- **Database**: MongoDB 7+
- **Cache**: Redis
- **Frontend**: React 18, Vite, TypeScript, Tailwind CSS
- **DevOps**: Docker, Docker Compose

## 🖥️ Platforms Supported

- Linux
- macOS
- Windows (via WSL)

## 🗂️ Key Directories

- `cmd/server/` - Main server executable
- `internal/` - Core application code
  - `database/` - Database connection logic
  - `handlers/` - API request handlers
  - `middleware/` - Authentication middleware
  - `models/` - Data models
  - `server/` - Server initialization logic
  - `embed/` - Embedded frontend assets
- `apps/web/` - React frontend
- `configs/` - Configuration management
- `scripts/` - Utility scripts

## 🔧 Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| PORT | Server listening port | "8080" |
| MONGO_URI | MongoDB connection string | "mongodb://localhost:27017" |
| MONGO_DB_NAME | Database name | "statusplatform" |
| REDIS_ADDR | Redis connection address | "localhost:6379" |
| JWT_SECRET | Secret for JWT generation | "super-secret-jwt-key-change-in-production" |
| ADMIN_EMAIL | Default admin email | "admin@statusplatform.com" |
| ADMIN_PASSWORD | Default admin password | "admin123" |
| ENABLE_WORKER | Enable monitoring worker | "true" |

## 🖨️ Running Mode

### Development
- Frontend: React development server with hot reloading (`npm run dev`)
- Backend: Go server with live reload
- Both services must run separately

### Production
- Built-in React frontend served from within the Go binary
- Single Go binary handles API, Web UI, and websocket connections
- Automated monitoring performed by integrated worker
- Deploy as a single container with MongoDB and Redis side-car

## 💻 API Endpoints

### Public Endpoints (No Auth Required)
- `GET /api/status/summary` - Overall platform status summary
- `GET /api/status/components` - All components with status and sub-components
- `GET /api/status/incidents` - Current and past incidents
- `POST /api/subscribe` - Subscribe email for notifications
- `GET /health` - Health check with database connectivity info
- `GET /` - Static React frontend

### Protected Endpoints (JWT Required)
- `POST /api/auth/login` - Authenticate admin user
- `GET /api/auth/me` - Get current authenticated user info
- **Components**
  - `GET /api/components` - Get all components
  - `POST /api/components` - Create component
  - `PATCH /api/components/:id` - Update component
  - `DELETE /api/components/:id` - Delete component
  - `GET /api/components/:id/subcomponents` - Get subcomponents for a specific component
- **Subcomponents**
  - `GET /api/subcomponents` - Get all subcomponents
  - `POST /api/subcomponents` - Create subcomponent
  - `PATCH /api/subcomponents/:id` - Update subcomponent
- **Monitoring**
  - `GET /api/monitors` - Get all monitors
  - `POST /api/monitors` - Create monitor
  - `DELETE /api/monitors/:id` - Delete monitor
  - `GET /api/monitors/:id/logs` - Get monitor logs
  - `GET /api/monitors/:id/uptime` - Get 90-day uptime history
  - `GET /api/monitors/:id/history` - Get enhanced monitor logs
  - `GET /api/monitors/outages` - Get all outage records
- **Incidents**
  - `GET /api/incidents` - Get all incidents
  - `POST /api/incidents` - Create incident
  - `PATCH /api/incidents/:id` - Update incident
  - `POST /api/incidents/:id/update` - Add incident update
  - `GET /api/incidents/:id/updates` - Get incident updates
- **Maintenance**
  - `GET /api/maintenance` - Get all maintenance windows
  - `POST /api/maintenance` - Create scheduled maintenance
  - `PATCH /api/maintenance/:id` - Update maintenance window
- **Subscribers**
  - `GET /api/subscribers` - Get all subscribers
  - `DELETE /api/subscribers/:id` - Delete subscriber

### WebSocket Endpoint
- `GET /ws` - WebSocket connection for real-time updates

## 💭 Design Philosophy

StatusForge follows the principle of "single-binary simplicity" where traditionally separate services (API backend, frontend web server, monitoring daemon, worker tasks) are combined into a single executable for easier deployment and management. It uses Go's excellent concurrency model via goroutines to handle multiple long-running processes.

## 🚀 Quick Start

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

## 🛠️ Configuration Options

| Environment Variable | Description | Default |
|---------------------|-------------|---------|
| `ENABLE_WORKER` | Whether to run the monitoring worker | `true` |
| `PORT` | Server listening port | `8080` |
| `MONGO_URI` | MongoDB connection string | `mongodb://mongo:27017` |
| `MONGO_DB_NAME` | Database name | `statusplatform` |
| `REDIS_ADDR` | Redis connection address | `redis:6379` | 
| `JWT_SECRET` | Secret for JWT generation | `change-me-in-production` |

## 🏥 Health Check

The server exposes a health check endpoint to verify database connectivity:

- `GET http://localhost:8080/health`
- Returns 200 if healthy, 503 if MongoDB or Redis are unreachable
- Response includes detailed DB connectivity status

## 🚢 Deployment

StatusForge is designed for containerized deployment:
- Single Docker image containing both server and static frontend assets
- Docker Compose configuration for orchestrating with MongoDB and Redis
- Production-optimized build process with static embedding
- Graceful shutdown handling for container orchestrators
- Healthcheck built into the image

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Add tests if applicable
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

## 📄 License

MIT