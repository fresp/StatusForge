# Screenshot Guidelines

## Required Screenshots

### 1. public-statuspage.jpeg
- Shows: Public-facing status page
- Size: 1280x720 minimum
- Content: Component status, uptime bars, incident list

### 2. admin-dashboard.png
- Shows: Admin dashboard overview
- Size: 1280x720 minimum
- Content: Navigation, quick stats, recent activity

### 3. admin-component.png
- Shows: Component management interface
- Size: 1280x720 minimum
- Content: Component list, create/edit forms

### 4. admin-incident.png
- Shows: Incident detail with timeline
- Size: 1280x720 minimum
- Content: Incident updates, status transitions

### 5. admin-monitoring.png
- Shows: Monitor list and metrics
- Size: 1280x720 minimum
- Content: Monitor status, response times, uptime

### 6. admin-maintenance.png
- Shows: Maintenance scheduling interface
- Size: 1280x720 minimum
- Content: Calendar view, maintenance details

## How to Capture

1. Start the application: `docker compose up`
2. Access admin at: http://localhost:8080/admin/login
3. Create sample data: `go run scripts/seed.go`
4. Capture screenshots using browser dev tools
5. Save to this directory with names above

## Current Status

All 6 required screenshots are present and have been integrated into the README.md documentation.
