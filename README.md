# Statora

![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)
![React](https://img.shields.io/badge/React-18-61DAFB?logo=react&logoColor=black)
![License](https://img.shields.io/github/license/fresp/Statora)
![Last Commit](https://img.shields.io/github/last-commit/fresp/Statora)

Statora is a self-hosted status page and uptime monitoring platform.

It helps you monitor services, publish incidents, schedule maintenance, and share real-time status updates from one place.

## Overview

The name Statora comes from "state" and "awareness", reflecting its role as a central system that continuously understands service conditions and turns them into clear, reliable communication.

With Statora, you can:

- Show a clean public status page for your services
- Track incidents and post updates as things happen
- Schedule planned maintenance ahead of time
- Monitor endpoints and infrastructure from one dashboard
- Manage everything from a dedicated admin area
- Self-host the platform in your own environment

## Features

### Public Status Experience
- Public-facing status page for services, components, and subcomponents
- Incident history for transparent communication over time
- Service detail views with uptime and status context
- Real-time updates via WebSocket for a more responsive status experience
- Category-based filtering for organizing services

### Incident & Maintenance Management
- Create, update, and resolve incidents from the admin area
- Automatic incident creation when monitors detect outages that are not already covered by an active incident
- Publish scheduled maintenance to prepare users in advance
- Automatic maintenance status transitions (scheduled, in-progress, completed)
- Keep status communication centralized and consistent

### Monitoring & Reliability
- Built-in active monitoring for HTTP, TCP, DNS, Ping, and SSL checks
- Configurable check intervals and timeouts per monitor
- Warning support for SSL and domain expiry monitoring
- Worker-driven status updates tied to monitoring results
- Automatic outage detection after 3 consecutive failures
- Daily uptime tracking and outage history

### Administration & Access Control
- Dedicated admin dashboard for operational management
- Role-aware access with `admin` and `operator` roles
- User invitation system for onboarding new team members
- MFA-aware protected flows for sensitive actions (TOTP-based)
- SSO callback support for external authentication
- Centralized settings for branding and platform behavior

### Realtime & Integrations
- WebSocket-powered live refresh for status updates
- Webhook channel management for external notifications
- Subscriber management for status communication workflows
- Public API endpoints for status data

## Screenshots

### Public Experience

| Status Page | Incident History | Service Details |
|---|---|---|
| ![Public Status Page](docs/screenshots/public-statuspage.png) | ![Incident History](docs/screenshots/incident-history.png) | ![Service Info](docs/screenshots/public-service-info.png) |

### Admin Experience

| Dashboard | Monitoring | Maintenance | Settings |
|---|---|---|---|
| ![Admin Dashboard](docs/screenshots/admin-dashboard.png) | ![Admin Monitoring](docs/screenshots/admin-monitoring.png) | ![Admin Maintenance](docs/screenshots/admin-maintenance.png) | ![Admin Settings](docs/screenshots/admin-settings.png) |

## What You Can Do

- Run a public-facing status page
- Manage incidents and maintenance in one place
- Monitor services with active checks
- Keep internal operators and external users aligned
- Self-host your uptime and status workflow

## Quick Start

The fastest way to run Statora locally is with Docker Compose.

### Run with Docker Compose

```bash
git clone https://github.com/fresp/Statora.git
cd Statora
cp .env.example .env
docker compose up --build
```

### Default Local Endpoints

- Public status page: `http://localhost:8080/`
- Admin area: `http://localhost:8080/admin`
- Health endpoint: `http://localhost:8080/health`
- WebSocket: `ws://localhost:8080/ws`

### Default Bootstrap Admin

Values come from `.env.example`:

- `ADMIN_EMAIL=admin@statusplatform.com`
- `ADMIN_USERNAME=admin`
- `ADMIN_PASSWORD=admin123`

Change these immediately for any shared or persistent environment.

## Tech Stack

- **Backend:** Go 1.26, Gin web framework
- **Frontend:** React 18, TypeScript 5, Vite 5, Tailwind CSS 3
- **Database:** MongoDB 7
- **Supporting runtime dependency:** Redis 7
- **Realtime:** Gorilla WebSocket
- **Authentication:** JWT (golang-jwt/v5) with MFA (pquerna/otp)
- **Deployment:** Docker, Docker Compose

## Self-Hosted by Design

Statora is designed to give teams control over their status workflow, monitoring setup, and public communication without depending on a hosted third-party service.

## Roadmap

Planned improvements include:

- Stronger production hardening for realtime and CORS behavior
- Richer API and developer documentation
- More scalable worker deployment patterns
- Broader observability support

## Contributing

Contributions are welcome. Open an issue or submit a pull request with a clear scope and validation notes.

## License

Licensed under the [MIT License](LICENSE).
