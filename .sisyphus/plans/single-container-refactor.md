# Single-Container Architecture Refactor

## TL;DR

> **Quick Summary**: Refactor multi-service Go application (API + Worker + Web) into a single binary running all services concurrently via goroutines, served from one Docker container.
> 
> **Deliverables**: 
> - Unified `cmd/server/main.go` entrypoint
> - Single Dockerfile for combined binary
> - Embedded React frontend in Go binary
> - Graceful shutdown handling (30s timeout)
> - Health check endpoint with database connectivity verification
> - Environment variable `ENABLE_WORKER` for API-only mode
> - Updated docker-compose.yml (simplified)
> 
> **Estimated Effort**: Medium
> **Parallel Execution**: YES - 5 waves
> **Critical Path**: Database validation → Embed frontend → Unified main → Docker setup → Verification

---

## Context

### Original Request
Refactor the multi-service architecture (API + Worker + Web) into a single Docker container with one unified binary running all services concurrently using goroutines.

**Goal**: Simplify deployment for single-node installations, Kubernetes, and small self-hosted environments.

**Constraints**:
- Do NOT break existing functionality
- Maintain feature parity
- Do NOT remove any business logic
- Only reorganize architecture and startup flow

### Current Architecture
**3 Separate Services**:
1. **API Service** (`apps/api/main.go`) - Gin HTTP server on port 8080, WebSocket hub, JWT auth, REST API
2. **Worker Service** (`apps/worker/main.go`) - Monitoring daemon with 10s ticker, HTTP/TCP/DNS/ICMP checks, auto-incident management
3. **Web Service** (`apps/web/`) - React frontend served via nginx on port 3000

**Infrastructure**: MongoDB, Redis, Docker Compose orchestration

### Interview Summary
**Key Decisions Made**:
- **Web Service**: Bundle into binary using `embed` package (serve static files from Go)
- **Worker Mode**: Environment variable `ENABLE_WORKER` (default: true) for flexible deployments
- **Graceful Shutdown**: SIGTERM/SIGINT with 30s timeout, completes in-flight requests
- **Health Check**: `/health` endpoint checks MongoDB + Redis connectivity
- **Testing**: Tests after implementation (not TDD)

**Research Findings**:
- **No service-to-service communication** - API and Worker only share databases
- **Redis not actually used** - Worker connects but doesn't query it
- **WebSocket hub already runs as goroutine** in API service (`go hub.Run()`)
- **Database singleton pattern** - Both services use `database.GetDB()` with global variables
- **Worker parallelism** - Uses `sync.WaitGroup` for parallel monitoring checks

### Metis Review
**Identified Gaps** (addressed in plan):
- **Database pooling**: Need to verify MongoDB driver handles concurrent goroutines (HTTP + Worker + WebSocket)
- **Resource cleanup**: Worker ticker must stop gracefully during shutdown
- **Logging identifiers**: Each log entry needs prefix (HTTP/WORKER/WS) for unified binary
- **Error isolation**: One service failure shouldn't crash entire binary
- **Edge cases**: Database failure on startup, multiple SIGTERM signals, overlapping worker tasks

**Guardrails Applied**:
- MUST NOT: Change database connection logic, add new API endpoints, modify worker monitoring logic
- MUST: Use existing singleton patterns, preserve all endpoint behavior, implement proper error handling
- MUST: Document startup order and goroutine dependencies

---

## Work Objectives

### Core Objective
Merge API, Worker, and Web services into a single Go binary that runs all components concurrently using goroutines, served from one Docker container, while maintaining feature parity and adding graceful shutdown.

### Concrete Deliverables
- `cmd/server/main.go` - Unified entrypoint with goroutine orchestration
- `cmd/server/api.go` - HTTP server setup (extracted from `apps/api/main.go`)
- `cmd/server/worker.go` - Worker setup (extracted from `apps/worker/main.go`)
- `cmd/server/signals.go` - Signal handling and graceful shutdown
- `cmd/server/health.go` - Health check handler with DB connectivity
- `cmd/server/static.go` - Static file serving with `embed` package
- Updated `configs/config.go` - Add `EnableWorker` field
- `Dockerfile` - Single Dockerfile for unified binary
- Updated `docker-compose.yml` - Simplified service definitions
- Tests for unified startup, graceful shutdown, health checks

### Definition of Done
- [ ] Binary starts and serves API (port 8080), Web (root path), WebSocket (`/ws`)
- [ ] Health check endpoint `/health` returns 200 OK with MongoDB + Redis status
- [ ] `ENABLE_WORKER=false` disables worker but HTTP server still works
- [ ] SIGTERM triggers graceful shutdown (HTTP completes requests, worker stops ticker, DB disconnects)
- [ ] All existing API endpoints work unchanged
- [ ] Docker image builds successfully and runs with single container
- [ ] All tests pass

### Must Have
- Single binary that runs HTTP server, WebSocket hub, Worker (optional)
- Environment variable `ENABLE_WORKER` to disable worker
- Graceful shutdown with 30s timeout
- Health check endpoint checking database connectivity
- Embedded React frontend served from root path
- All existing functionality preserved

### Must NOT Have (Guardrails from Metis)
- NO database schema changes
- NO new API endpoints beyond `/health`
- NO changes to worker monitoring logic (only add ENABLE_WORKER flag)
- NO new external dependencies (use existing libraries)
- NO configuration hot reloading (keep simple env-based)
- NO metrics endpoint (out of scope)
- NO circuit breakers (unless database issues occur)
- NO shared mutable state between goroutines

---

## Verification Strategy (MANDATORY)

> **ZERO HUMAN INTERVENTION** — ALL verification is agent-executed. No exceptions.

### Test Decision
- **Infrastructure exists**: YES (Go testing framework)
- **Automated tests**: Tests after implementation
- **Framework**: `go test` with standard library
- **Test categories**: 
  - Unit tests for graceful shutdown, health check
  - Integration tests for unified startup, API endpoints
  - E2E test with Docker container

### QA Policy
Every task MUST include agent-executed QA scenarios.

- **API/Backend**: Use Bash (curl) — Send requests, assert status + response fields
- **Docker**: Use Bash (docker) — Build image, run container, verify startup
- **Go Binary**: Use Bash (go build, go test) — Compile, run tests, check exit codes
- **Health Check**: Use Bash (curl) — Hit `/health`, verify MongoDB + Redis status

---

## Execution Strategy

### Parallel Execution Waves

> Maximize throughput by grouping independent tasks into parallel waves.
> Each wave completes before the next begins.
> Target: 5-8 tasks per wave.

```
Wave 1 (Foundation - can start immediately):
├── Task 1: Create configs.EnableWorker field [quick]
├── Task 2: Create cmd/server directory structure [quick]
├── Task 3: Extract worker setup to cmd/server/worker.go [quick]
├── Task 4: Create graceful shutdown handler in cmd/server/signals.go [quick]
└── Task 5: Create health check handler in cmd/server/health.go [quick]

Wave 2 (Core Implementation - depends on Wave 1):
├── Task 6: Extract API setup to cmd/server/api.go [quick]
├── Task 7: Create embed setup for React frontend in cmd/server/static.go [quick]
├── Task 8: Create unified main.go with goroutine orchestration [deep]
└── Task 9: Add logging identifiers (HTTP/WORKER/WS prefixes) [quick]

Wave 3 (Integration - depends on Wave 2):
├── Task 10: Update existing handlers to use logging prefixes [quick]
├── Task 11: Update existing worker code to use logging prefixes [quick]
├── Task 12: Verify database connection pooling handles concurrent access [deep]
└── Task 13: Create integration test for unified startup [deep]

Wave 4 (Docker & Deployment - depends on Wave 3):
├── Task 14: Create single Dockerfile for unified binary [quick]
├── Task 15: Update docker-compose.yml for simplified architecture [quick]
├── Task 16: Create .dockerignore for optimized builds [quick]
└── Task 17: Test Docker build and container startup [unspecified-high]

Wave 5 (Verification & Cleanup - depends on Wave 4):
├── Task 18: Create graceful shutdown test [deep]
├── Task 19: Create health check integration test [deep]
├── Task 20: Create worker mode toggle test (ENABLE_WORKER=false) [deep]
├── Task 21: Remove old apps/api and apps/worker directories [quick]
└── Task 22: Update README.md with new architecture [writing]

Wave FINAL (After ALL tasks — independent review, 4 parallel):
├── Task F1: Plan compliance audit (oracle)
├── Task F2: Code quality review (unspecified-high)
├── Task F3: Real manual QA (unspecified-high)
└── Task F4: Scope fidelity check (deep)

Critical Path: Task 1 → Task 8 → Task 13 → Task 17 → F1-F4
Parallel Speedup: ~60% faster than sequential
Max Concurrent: 5 (Waves 1 & 5)
```

### Dependency Matrix (abbreviated)

- **1-5**: — — 6-9, 1
- **6-7**: 1-5 — 8, 2
- **8**: 1-9 — 13, 3
- **10-13**: 6-9 — 14-17, 4
- **14-17**: 10-13 — 18-22, 5
- **18-22**: 14-17 — F1-F4

> Full matrix for all tasks provided in TODOs section.

### Agent Dispatch Summary

- **Wave 1**: **5 tasks** — All `quick`
- **Wave 2**: **4 tasks** — T6-T7, T9 → `quick`, T8 → `deep`
- **Wave 3**: **4 tasks** — T10-T11, T13 → `deep`, T12 → `deep`
- **Wave 4**: **4 tasks** — T14-T16 → `quick`, T17 → `unspecified-high`
- **Wave 5**: **5 tasks** — T18-T20 → `deep`, T21 → `quick`, T22 → `writing`
- **FINAL**: **4 tasks** — F1 → `oracle`, F2-F3 → `unspecified-high`, F4 → `deep`

---

## TODOs

> Implementation + Test = ONE Task. Never separate.
> EVERY task MUST have: Recommended Agent Profile + Parallelization info + QA Scenarios.

---
## Final Verification Wave (MANDATORY — after ALL implementation tasks)

> 4 review agents run in PARALLEL. ALL must APPROVE. Rejection → fix → re-run.

- [ ] F1. **Plan Compliance Audit** — `oracle`
  Read the plan end-to-end. For each "Must Have": verify implementation exists (read file, curl endpoint, run command). For each "Must NOT Have": search codebase for forbidden patterns — reject with file:line if found. Check evidence files exist in .sisyphus/evidence/. Compare deliverables against plan.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [ ] F2. **Code Quality Review** — `unspecified-high`
  Run `go vet`, `go test -race`, and `golangci-lint` if available. Review all changed files for: `panic()` without recovery, global mutable state, missing error handling, goroutine leaks, unbuffered channels. Check AI slop: excessive comments, over-abstraction, generic names.
  Output: `Build [PASS/FAIL] | Lint [PASS/FAIL] | Tests [N pass/N fail] | Files [N clean/N issues] | VERDICT`

- [ ] F3. **Real Manual QA** — `unspecified-high`
  Start from clean state. Execute EVERY QA scenario from EVERY task — follow exact steps, capture evidence. Test cross-task integration (features working together, not isolation). Test edge cases: ENABLE_WORKER=false, database connection failure, SIGTERM during request. Save to `.sisyphus/evidence/final-qa/`.
  Output: `Scenarios [N/N pass] | Integration [N/N] | Edge Cases [N tested] | VERDICT`

- [ ] F4. **Scope Fidelity Check** — `deep`
  For each task: read "What to do", read actual diff (git log/diff). Verify 1:1 — everything in spec was built (no missing), nothing beyond spec was built (no creep). Check "Must NOT do" compliance. Detect cross-task contamination: Task N touching Task M's files. Flag unaccounted changes.
  Output: `Tasks [N/N compliant] | Contamination [CLEAN/N issues] | Unaccounted [CLEAN/N files] | VERDICT`

---

## Commit Strategy

- **Wave 1**: `refactor(config): add EnableWorker field for single-binary mode`
- **Wave 2**: `refactor(cmd/server): create unified entrypoint with goroutine orchestration`
- **Wave 3**: `test(integration): add tests for unified startup and health checks`
- **Wave 4**: `refactor(docker): create single Dockerfile for unified binary`
- **Wave 5**: `refactor(arch): remove old apps/ structure, update docs`
- **Each commit**: Run `go test ./...` before committing

---

## Success Criteria

### Verification Commands
```bash
# 1. Binary builds successfully
go build -o bin/server ./cmd/server
# Expected: Exit code 0, binary created

# 2. Health check responds
curl -s http://localhost:8080/health | jq '.'
# Expected: {"status":"healthy","mongodb":"connected","redis":"connected"}

# 3. Worker mode disabled
ENABLE_WORKER=false ./bin/server &
sleep 3
curl -s http://localhost:8080/api/status/summary
# Expected: Valid JSON response (worker not needed for API)

# 4. Graceful shutdown
pkill -SIGTERM server
# Expected: Process exits with code 0 within 30 seconds

# 5. Docker build succeeds
docker build -t statusforge:unified .
docker run -d -p 8080:8080 --name test-server statusforge:unified
sleep 5
curl -s http://localhost:8080/health
# Expected: {"status":"healthy",...}

# 6. All tests pass
go test ./... -race
# Expected: PASS, 0 failures
```

### Final Checklist
- [ ] All "Must Have" present
- [ ] All "Must NOT Have" absent
- [ ] All tests pass
- [ ] Docker image builds and runs
- [ ] Graceful shutdown works (verified with pkill)
- [ ] Health check endpoint functional
- [ ] Worker mode toggle works
- [ ] Frontend served from root path
- [ ] API endpoints unchanged
- [ ] No new dependencies added