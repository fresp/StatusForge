# StatusForge

StatusForge is a self-hosted status page platform built with a Go API/server and a React admin/public frontend. It combines service monitoring, incident and maintenance management, subscriber management, RBAC-protected admin workflows, MFA-aware authentication, and a customizable real-time public status page.

---

## Overview

- **Service Monitoring** - Track the status of your services in real-time
- **Incident Management** - Create, update, and resolve incidents with timelines
- **Role-Based Admin Access** - Enforce RBAC with `admin` and `operator` roles on admin APIs
- **Operator Scope Enforcement** - Limit operator access to incidents and maintenance workflows
- **Public Status Page** - Share service status with your users
- **Real-time Updates** - WebSocket-powered live updates
- **Members & Invitations Split View** - Manage active members and pending invitations in separate admin sections
- **Self-hosted** - Full control over your data and infrastructure
- **Lightweight Architecture** - Simple Go backend with React frontend
- **Webhook Notification Channels** - Configure webhooks to receive incident and maintenance updates
- **Custom Status Page Themes and Settings** - Customize the look and feel of your public status page

---

## Screenshots

| Dashboard | Public Status Page | Incident Timeline |
|-----------|-------------------|-------------------|
| ![Dashboard](docs/images/admin-dashboard.png) | ![Status Page](docs/images/public-statuspage.jpeg) | ![Incidents](docs/images/admin-incident.png) |

---

## Architecture

### Backend

- **Language**: Go
- **HTTP framework**: Gin
- **Database**: MongoDB
- **Cache / infra dependency**: Redis
- **Auth**: JWT + MFA verification flow
- **Realtime**: Gorilla WebSocket hub

### Frontend

- **Framework**: React 18
- **Build tool**: Vite
- **Language**: TypeScript
- **Styling**: Tailwind CSS

### Runtime shape

- Entry point: `cmd/server/main.go`
- Server bootstrap: `internal/server/server.go`
- API route registration: `internal/server/api_routes.go`
- Static asset serving: `internal/server/static.go`
- Background monitoring worker: `internal/server/worker.go`

---

## How It Works

At startup, the unified server:

1. Loads environment variables from `.env` if available
2. Builds runtime configuration from `configs/config.go`
3. Connects to MongoDB and Redis
4. Starts the WebSocket hub
5. Registers API routes and `/health`
6. Optionally starts the monitoring worker when `ENABLE_WORKER=true`
7. Seeds the default admin user if it does not exist
8. Serves the embedded frontend bundle

The monitoring worker periodically executes checks, records monitor logs, updates uptime summaries, and detects outages.

---

## Authentication and Authorization

Auth middleware lives in `internal/middleware/auth.go`.

- **Authentication**: Bearer JWT
- **Authorization**: role checks via `RequireRoles(...)`
- **MFA enforcement**: `RequireMFAVerified()` protects verified admin areas

Current roles:

- `admin`
- `operator`

Typical access model:

- Public endpoints are open
- Login and invitation activation are public
- Authenticated users can access profile and MFA endpoints
- MFA-verified admins/operators can manage incidents and maintenance
- MFA-verified admins can manage monitors, settings, components, subscribers, and users

---

## API Surface

### Public endpoints

- `GET /health`
- `GET /ws`
- `GET /api/status/summary`
- `GET /api/status/components`
- `GET /api/status/incidents`
- `GET /api/status/settings`
- `POST /api/subscribe`
- `POST /api/auth/login`
- `POST /api/users/invitations/activate`

### Authenticated endpoints

- `GET /api/auth/me`
- `PATCH /api/auth/me`
- `POST /api/auth/mfa/setup`
- `POST /api/auth/mfa/verify`
- `POST /api/auth/mfa/recovery/verify`
- `POST /api/auth/mfa/disable`

### Incident and maintenance endpoints

- `GET /api/incidents`
- `POST /api/incidents`
- `PATCH /api/incidents/:id`
- `POST /api/incidents/:id/update`
- `GET /api/incidents/:id/updates`
- `GET /api/maintenance`
- `POST /api/maintenance`
- `PATCH /api/maintenance/:id`

### Components and subcomponents

- `GET /api/components`
- `GET /api/components/:id/subcomponents`
- `GET /api/subcomponents`
- `POST /api/components`
- `PATCH /api/components/:id`
- `DELETE /api/components/:id`
- `POST /api/subcomponents`
- `PATCH /api/subcomponents/:id`

### Monitor endpoints

- `GET /api/monitors`
- `POST /api/monitors`
- `POST /api/monitors/test`
- `PUT /api/monitors/:id`
- `DELETE /api/monitors/:id`
- `GET /api/monitors/:id/logs`
- `GET /api/monitors/:id/uptime`
- `GET /api/monitors/:id/history`
- `GET /api/monitors/outages`

### Subscribers, users, and settings

- `GET /api/subscribers`
- `DELETE /api/subscribers/:id`
- `GET /api/settings/status-page`
- `PATCH /api/settings/status-page`
- `GET /api/users`
- `PATCH /api/users/:id`
- `DELETE /api/users/:id`
- `POST /api/users/invitations`
- `GET /api/users/invitations`
- `POST /api/users/invitations/:id/refresh`
- `DELETE /api/users/invitations/:id`

---

## Status Page Settings and Theme Customization

Status page settings are stored in MongoDB in the `settings` collection using the key `status_page`.

Relevant backend files:

- `internal/models/status_page_settings.go`
- `internal/handlers/status_page_settings.go`

Relevant frontend files:

- `apps/web/src/pages/admin/AdminSettings.tsx`
- `apps/web/src/pages/StatusPage.tsx`
- `apps/web/src/types/index.ts`

### What can be customized now

- **Head / SEO**
  - page title
  - meta description
  - meta keywords
  - favicon URL
  - arbitrary custom meta tags
- **Branding**
  - site name
  - logo URL
- **Theme**
  - primary color
  - background color
  - text color
- **Layout**
  - `classic`
  - `compact`
- **Footer**
  - footer text
  - powered-by visibility toggle
- **Custom CSS**
  - raw CSS applied at runtime to the public status page

### Runtime behavior

- Public clients load settings from `GET /api/status/settings`
- Admin users manage settings through `/admin/settings`
- Settings updates emit the WebSocket event `status_page_settings_updated`
- The public status page listens for that event and refreshes settings live
- Default fallback settings are applied when settings do not yet exist

### Custom status page themes checklist after this update

- [x] Persisted theme settings model in MongoDB
- [x] Public API for reading current status page settings
- [x] Admin API for updating settings
- [x] Admin settings screen for form-based editing
- [x] Runtime application of branding and theme values on the public page
- [x] Runtime title, favicon, and meta tag updates
- [x] Layout variant support (`classic`, `compact`)
- [x] Footer customization support
- [x] Custom CSS injection support
- [x] WebSocket-driven live refresh on settings updates
- [x] Default/fallback settings for first-run behavior

### Roadmap for custom status page themes

- [ ] Dark mode / multi-theme presets
- [ ] Richer semantic color tokens for statuses and incidents
- [ ] Typography and font selection controls
- [ ] Section-level layout controls beyond `classic` and `compact`
- [ ] Theme preview / draft mode before publish
- [ ] Import/export theme configurations
- [ ] Safer CSS validation or linting for custom CSS input
- [ ] Asset upload workflow for logos and favicons

---

## Project Structure

```text
.
├── cmd/server/                 # Application entry point
├── configs/                    # Environment-backed runtime config
├── internal/
│   ├── database/               # MongoDB and Redis connection helpers
│   ├── embed/                  # Embedded frontend assets
│   ├── handlers/               # HTTP and WebSocket handlers
│   ├── middleware/             # Auth / RBAC / MFA middleware
│   ├── models/                 # Mongo-backed domain models
│   ├── server/                 # Bootstrap, routes, worker, static serving
│   └── utils/                  # Monitoring utility checks
├── apps/web/                   # React/Vite frontend
├── docs/                       # Supporting docs and images
├── Dockerfile                  # Multi-stage production image
├── docker-compose.yml          # Local deployment stack
├── Makefile                    # Docker lifecycle helpers
└── .env.example                # Required environment template
```

---

## Environment Variables

StatusForge reads configuration from environment variables and falls back to defaults in `configs/config.go`.

### Required / important variables

```env
MONGO_URI=mongodb://localhost:27017
MONGO_DB_NAME=statusplatform
REDIS_ADDR=localhost:6379
JWT_SECRET=change-this-jwt-secret
MFA_SECRET_KEY=change-this-mfa-secret
PORT=8080
ADMIN_EMAIL=admin@statusplatform.com
ADMIN_USERNAME=admin
ADMIN_PASSWORD=admin123
ENABLE_WORKER=true
```

### Notes

- `ADMIN_*` values are used for bootstrap seeding of the default admin account
- `ENABLE_WORKER=false` disables background monitor execution
- `JWT_SECRET` should be replaced in production
- `MFA_SECRET_KEY` should be set explicitly in production

---

## Quick Start with Docker Compose

The quickest way to run the full stack locally:

```bash
cp .env.example .env
docker compose up --build
```

Or with Make:

```bash
make up-build
```

Access the app at:

- Public status page: `http://localhost:8080`
- Admin panel: `http://localhost:8080/admin`
- Health check: `http://localhost:8080/health`

Default bootstrap credentials:

```text
Email: admin@statusplatform.com
Password: admin123
```

Change those credentials before any real deployment.

---

## Local Development

### Prerequisites

- Go 1.26+
- Node.js 20+
- MongoDB
- Redis

### Backend

```bash
cp .env.example .env
go run cmd/server/main.go
```

### Frontend

```bash
cd apps/web
npm install
npm run dev
```

The frontend Vite dev server is useful during UI work, while the production runtime embeds built frontend assets into the Go server.

---

## Build and Verification

### Frontend

```bash
cd apps/web
npm run build
```

### Backend

```bash
go build ./...
go test ./...
```

### Current known limitation

At the time of the latest verification, `go test ./...` exposed a pre-existing failure in:

- `internal/handlers/admin_members_test.go`
- `TestCreateUserInvitationRejectsInvalidRoleBeforeDBAccess`

This is unrelated to the status page settings/theme work.

---

## Docker and Deployment Files

- `Dockerfile` builds the frontend first, embeds the built assets into the Go binary, and produces a minimal Alpine runtime image
- `docker-compose.yml` provisions:
  - `server`
  - `mongo`
  - `redis`
- `Makefile` wraps common Docker Compose workflows such as:
  - `make up`
  - `make up-build`
  - `make down`
  - `make logs`
  - `make ps`

---

## Operational Notes

- Redis connection failures are logged as warnings rather than fatal startup errors
- MongoDB connection failures are fatal
- The monitoring worker runs every 10 seconds when enabled
- Static frontend files are served with SPA fallback behavior
- The server seeds the admin user on startup if the configured account does not exist

---

## Roadmap

- [x] Basic role-based multi-user admin system (`admin` / `operator`)
- [x] Custom status page settings and initial theme support
- [x] Notification channels (Webhooks)
- [ ] Advanced monitoring checks (for example SSL expiry)
- [ ] Multi-database support
- [ ] Analytics and reporting

---

## Contributing

Contributions are welcome.

- Open issues for bugs and feature requests
- Submit pull requests for fixes and improvements
- Improve docs, tests, or deployment workflows

If you change runtime behavior, APIs, or infrastructure, update this README as part of the change.

---

## License

[MIT](LICENSE)
