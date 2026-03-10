# StatusForge

An open-source, self-hosted status page platform for monitoring services and managing incidents. Similar to Atlassian Statuspage and other commercial status page providers.

---

## Features

- Service status monitoring 
- Incident management with timeline
- Public status page for transparency
- Real-time updates (WebSocket)
- Self-hosted deployment with full control  
- Simple, unified architecture

---

## Screenshots

Dashboard | Public Status Page | Incident Timeline
--- | --- | ---
![Admin Dashboard](docs/images/admin-dashboard.png) | ![Public Status Page](docs/images/public-statuspage.jpeg) | ![Incident Timeline](docs/images/admin-incident.png)

---

## Quick Start

### Docker Compose (Recommended)

```bash
docker-compose up --build
```

Access the platform:
- Status page: http://localhost:8080
- Admin panel: http://localhost:8080/admin

Default credentials:
```
Email: admin@statusplatform.com 
Password: admin123
```

### Manual Installation

See installation guide below.

---

## Tech Stack

### Backend
- **Go** - Main backend language with concurrency
- **Gin** - Web framework
- **MongoDB** - Primary database 
- **Redis** - Cache and pub-sub

### Frontend
- **React** - UI framework
- **Vite** - Build tooling
- **Tailwind CSS** - Styling

---

## Installation

Clone the repository:
```bash
git clone https://github.com/fresp/StatusForge.git
cd StatusForge
```

### Prerequisites
- Docker & Docker Compose

### Run with Docker Compose
```bash
docker-compose up --build
```

### Configuration

Copy `.env.example` to `.env` and customize:
````
PORT=8080
MONGO_URI=mongodb://mongo:27017
MONGO_DB_NAME=statusplatform
REDIS_ADDR=redis:6379
JWT_SECRET=super-secret-jwt-key-change-in-production
ADMIN_EMAIL=admin@statusplatform.com
ADMIN_PASSWORD=admin123
```

### Development
- Backend: `go run cmd/server/main.go`
- Frontend: `cd apps/web && npm run dev`

---

## Roadmap

- Multi database support (PostgreSQL, MySQL)
- Role-based multi-user admin system
- Enhanced notification system (Slack, Teams, Webhooks)
- Advanced monitoring checks (ICMP, SSL expiry, etc.)
- Custom status page themes
- Email/SMS/Webhook notifications
- Maintenance window management
- Reporting and analytics

---

## Contributing

We welcome contributions! This is an open-source project looking for community support.

- Report issues and suggest features
- Contribute code and documentation
- Help improve the platform for everyone

Detailed developer documentation is located in `/docs`.

---

## License

MIT