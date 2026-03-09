# Fix Go Build Errors and Cleanup Codebase

## TL;DR

> **Quick Summary**: Fix Go embed pattern syntax, remove duplicate function declarations, and clean up unused code to make `go run cmd/server/main.go` run successfully.
>
> **Deliverables**:
> - Fixed embed.go with correct `//go:embed` pattern
> - Fixed static.go with correct path reference
> - Removed duplicate `common_api.go` file
> - Copied frontend build to correct location
> - Verified server startup
>
> **Estimated Effort**: Quick
> **Parallel Execution**: NO - sequential (foundation → cleanup → verify)
> **Critical Path**: Copy frontend → Fix embed.go → Fix static.go → Delete common_api.go → Verify build

---

## Context

### Original Request
Fix build errors and clean up the codebase so that `go run cmd/server/main.go` runs successfully without errors.

### Interview Summary
**Key Discussions**:
- **Root cause**: Go embed cannot use `../` in patterns; assets must be inside package directory tree
- **Duplicate code**: `common_api.go` and `api_routes.go` have identical function declarations
- **Correct implementation**: `api_routes.go` has proper imports, `common_api.go` is broken (missing imports)

**Research Findings**:
- Frontend build output exists at `apps/web/dist/` (index.html, assets/)
- `internal/embed/web/` directory exists but is empty
- `api_routes.go` has correct MongoDB/bcrypt imports
- `common_api.go` missing `mongo`, `bson`, `primitive`, `bcrypt`, `models` imports

### Metis Review
**Identified Gaps** (addressed):
- Build automation: Not included in scope (manual copy only)
- Docker considerations: Not in scope for this task
- Test files: Verified they don't reference `common_api.go`

**Questions Raised**:
1. Should build automation be added? → **No, out of scope for this task**
2. Should Docker be updated? → **No, separate concern**
3. Any other embed references? → **Only `server.go` and `static.go`**

---

## Work Objectives

### Core Objective
Fix all Go build errors so `go run cmd/server/main.go` starts successfully with MongoDB and Redis connections.

### Concrete Deliverables
- `internal/embed/web/dist/` with copied frontend build
- `internal/embed/embed.go` with fixed pattern
- `internal/server/static.go` with correct path
- Deleted `internal/server/common_api.go`
- Working `go run cmd/server/main.go`

### Definition of Done
- [ ] `go build ./...` exits with code 0
- [ ] `go run cmd/server/main.go` starts without errors
- [ ] Server listens on configured port (default 8080)
- [ ] Health endpoint returns 200 OK

### Must Have
- All build errors resolved
- Duplicate function declarations removed
- Frontend build accessible via embed

### Must NOT Have (Guardrails)
- No changes to frontend build process
- No build automation scripts (out of scope)
- No Docker configuration changes
- No unnecessary refactoring beyond fixing errors
- No changing API route definitions

---

## Verification Strategy (MANDATORY)

> **ZERO HUMAN INTERVENTION** — ALL verification is agent-executed. No exceptions.

### Test Decision
- **Infrastructure exists**: YES (test files present in `internal/server/`)
- **Automated tests**: NO (existing tests don't cover embed system)
- **Framework**: Go testing package (`testing`)
- **Primary verification**: Agent-executed commands

### QA Policy
Every task includes agent-executed QA scenarios using Bash commands.
Evidence saved to `.sisyphus/evidence/task-{N}-{scenario-slug}.{ext}`.

- **Build verification**: `go build ./...` (exit code 0, no output)
- **Runtime verification**: `go run cmd/server/main.go` (startup logs)
- **API verification**: `curl http://localhost:8080/health` (200 OK response)

---

## Execution Strategy

### Parallel Execution Waves

```
Wave 1 (Foundation — copy assets):
└── Task 1: Copy frontend build to embed location [quick]

Wave 2 (Fix embed system):
├── Task 2: Fix embed.go pattern [quick]
└── Task 3: Fix static.go path [quick]

Wave 3 (Cleanup):
└── Task 4: Delete duplicate common_api.go [quick]

Wave 4 (Verification):
├── Task 5: Verify build succeeds [quick]
└── Task 6: Verify server startup [quick]

Wave FINAL (After ALL tasks — independent review, 2 parallel):
├── Task F1: Plan compliance audit (unspecified-high)
└── Task F2: Scope fidelity check (quick)

Critical Path: Task 1 → Task 2 → Task 3 → Task 4 → Task 5 → Task 6 → F1-F2
```

### Dependency Matrix
- **1**: — — 2, 3, 1
- **2**: 1 — 5, 1
- **3**: 1 — 5, 1
- **4**: — 5, 1
- **5**: 2, 3, 4 — 6, 1
- **6**: 5 — F1, F2, 1

### Agent Dispatch Summary
- **1**: **1** — T1 → `quick`
- **2**: **2** — T2 → `quick`, T3 → `quick`
- **3**: **1** — T4 → `quick`
- **4**: **2** — T5 → `quick`, T6 → `quick`
- **FINAL**: **2** — F1 → `unspecified-high`, F2 → `quick`

---

## TODOs

- [ ] 1. Copy Frontend Build to Embed Location

  **What to do**:
  - Create directory `internal/embed/web/dist` if it doesn't exist
  - Copy all files from `apps/web/dist/` to `internal/embed/web/dist/`
  - Verify files are present: index.html and assets/ directory

  **Must NOT do**:
  - Do NOT modify the frontend build files
  - Do NOT create symlinks (breaks in Docker)
  - Do NOT add build automation scripts (out of scope)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple file copy operation, no complex logic
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 1
  - **Blocks**: Task 2, Task 3
  - **Blocked By**: None (can start immediately)

  **References**:
  - `apps/web/dist/index.html` - Source file to copy
  - `apps/web/dist/assets/` - Source directory to copy
  - `internal/embed/web/` - Destination directory (exists but empty)

  **Acceptance Criteria**:
  - [ ] Directory `internal/embed/web/dist/` exists
  - [ ] File `internal/embed/web/dist/index.html` exists
  - [ ] Directory `internal/embed/web/dist/assets/` exists

  **QA Scenarios**:
  ```
  Scenario: Verify frontend files copied correctly
    Tool: Bash
    Preconditions: Command executed from project root
    Steps:
      1. ls -la internal/embed/web/dist/index.html
      2. ls -la internal/embed/web/dist/assets/
    Expected Result: Both commands succeed (exit 0), files exist
    Failure Indicators: "No such file or directory" error
    Evidence: .sisyphus/evidence/task-1-files-copied.txt
  ```

  **Commit**: YES (1)
  - Message: `fix(embed): copy frontend build to embed location`
  - Files: `internal/embed/web/dist/*`

- [ ] 2. Fix Embed Pattern in embed.go

  **What to do**:
  - Replace the invalid `//go:embed all:../../apps/web/dist/*` pattern
  - Update to valid pattern: `//go:embed web/dist`

  **Must NOT do**:
  - Do NOT use `../` in embed pattern (invalid)
  - Do NOT change the variable name `Assets`

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple code fix, single line change
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Task 3)
  - **Parallel Group**: Wave 2
  - **Blocks**: Task 5 (build verification)
  - **Blocked By**: Task 1 (frontend files must exist)

  **References**:
  - `internal/embed/embed.go:5` - Line to modify

  **Acceptance Criteria**:
  - [ ] File `internal/embed/embed.go` has pattern `//go:embed web/dist`
  - [ ] `go build ./internal/embed/...` succeeds

  **QA Scenarios**:
  ```
  Scenario: Verify embed pattern compiles
    Tool: Bash
    Preconditions: Frontend files copied (Task 1 complete)
    Steps:
      1. go build ./internal/embed/...
    Expected Result: Command exits with code 0, no errors
    Failure Indicators: "invalid pattern syntax" error
    Evidence: .sisyphus/evidence/task-2-embed-compiles.txt
  ```

  **Commit**: YES (2)
  - Message: `fix(embed): correct embed pattern syntax`
  - Files: `internal/embed/embed.go`

- [ ] 3. Fix Static File Path in static.go

  **What to do**:
  - Update `fs.Sub(embed.Assets, "../../apps/web/dist")` to use correct path
  - Change to `fs.Sub(embed.Assets, "web/dist")`

  **Must NOT do**:
  - Do NOT change the function structure or logic
  - Do NOT modify the React router fallback behavior

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Simple string replacement
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: YES (with Task 2)
  - **Parallel Group**: Wave 2
  - **Blocks**: Task 5 (build verification)
  - **Blocked By**: Task 1 (frontend files must exist)

  **References**:
  - `internal/server/static.go:15` - Line with incorrect path

  **Acceptance Criteria**:
  - [ ] File `internal/server/static.go` line 15 uses `"web/dist"`
  - [ ] `go build ./internal/server/...` succeeds

  **QA Scenarios**:
  ```
  Scenario: Verify static path compiles
    Tool: Bash
    Preconditions: Embed pattern fixed (Task 2 complete or parallel)
    Steps:
      1. go build ./internal/server/...
    Expected Result: Command exits with code 0, no errors
    Failure Indicators: "undefined" or compile errors
    Evidence: .sisyphus/evidence/task-3-static-compiles.txt
  ```

  **Commit**: YES (3)
  - Message: `fix(server): correct static file path reference`
  - Files: `internal/server/static.go`

- [ ] 4. Delete Duplicate common_api.go

  **What to do**:
  - Delete the file `internal/server/common_api.go`
  - This file has duplicate declarations and missing imports
  - The correct implementation is in `api_routes.go`

  **Must NOT do**:
  - Do NOT delete `api_routes.go` (that's the correct file)
  - Do NOT try to fix `common_api.go` (just delete it)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Single file deletion
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 3
  - **Blocks**: Task 5 (build verification)
  - **Blocked By**: None

  **References**:
  - `internal/server/common_api.go` - File to delete
  - `internal/server/api_routes.go` - Correct implementation (keep)

  **Acceptance Criteria**:
  - [ ] File `internal/server/common_api.go` does not exist
  - [ ] File `internal/server/api_routes.go` still exists
  - [ ] No "redeclared" errors in build

  **QA Scenarios**:
  ```
  Scenario: Verify duplicate declarations removed
    Tool: Bash
    Preconditions: None
    Steps:
      1. test ! -f internal/server/common_api.go && echo "File deleted"
      2. test -f internal/server/api_routes.go && echo "api_routes.go exists"
    Expected Result: Both checks pass
    Failure Indicators: First check fails or second fails
    Evidence: .sisyphus/evidence/task-4-duplicate-removed.txt
  ```

  **Commit**: YES (4)
  - Message: `chore(server): remove duplicate api routes file`
  - Files: `internal/server/common_api.go` (deleted)

- [ ] 5. Verify Build Succeeds

  **What to do**:
  - Run `go build ./...` to verify all compile errors are resolved
  - Run `go vet ./...` to check for code issues

  **Must NOT do**:
  - Do NOT run `go run` yet (that's Task 6)
  - Do NOT modify any code based on vet warnings (unless critical)

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Verification commands only
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 4
  - **Blocks**: Task 6
  - **Blocked By**: Task 2, Task 3, Task 4

  **References**:
  - Go build: `go help build`

  **Acceptance Criteria**:
  - [ ] `go build ./...` exits with code 0
  - [ ] `go vet ./...` exits with code 0 (or only minor warnings)

  **QA Scenarios**:
  ```
  Scenario: Verify complete build succeeds
    Tool: Bash
    Preconditions: All previous tasks complete
    Steps:
      1. go build ./... 2>&1
      2. echo "Build exit code: $?"
    Expected Result: Exit code 0, no error messages
    Failure Indicators: Non-zero exit code or errors
    Evidence: .sisyphus/evidence/task-5-build-succeeds.txt
  ```

  **Commit**: NO (verification only)

- [ ] 6. Verify Server Startup

  **What to do**:
  - Run `go build -o /tmp/server-test ./cmd/server/` to verify binary builds
  - Confirm no compile errors in the process

  **Must NOT do**:
  - Do NOT make code changes
  - Do NOT require external services for this verification

  **Recommended Agent Profile**:
  - **Category**: `quick`
    - Reason: Runtime verification
  - **Skills**: []

  **Parallelization**:
  - **Can Run In Parallel**: NO
  - **Parallel Group**: Wave 4 (after Task 5)
  - **Blocks**: Final verification
  - **Blocked By**: Task 5

  **References**:
  - `cmd/server/main.go` - Entry point

  **Acceptance Criteria**:
  - [ ] `go build ./cmd/server/` succeeds
  - [ ] No "undefined" or "redeclared" errors

  **QA Scenarios**:
  ```
  Scenario: Verify server binary builds
    Tool: Bash
    Preconditions: Build succeeds (Task 5 complete)
    Steps:
      1. go build -o /tmp/server-test ./cmd/server/ 2>&1
      2. echo "Build exit code: $?"
      3. rm -f /tmp/server-test
    Expected Result: Exit code 0, binary created
    Failure Indicators: Build errors
    Evidence: .sisyphus/evidence/task-6-server-binary.txt
  ```

  **Commit**: NO (verification only)

---

## Final Verification Wave (MANDATORY — after ALL implementation tasks)

- [ ] F1. **Plan Compliance Audit** — `unspecified-high`
  Read the plan end-to-end. For each "Must Have": verify implementation exists. For each "Must NOT Have": search codebase for forbidden patterns. Check evidence files exist. Compare deliverables against plan.
  Output: `Must Have [N/N] | Must NOT Have [N/N] | Tasks [N/N] | VERDICT: APPROVE/REJECT`

- [ ] F2. **Scope Fidelity Check** — `quick`
  For each task: read "What to do", read actual diff. Verify 1:1 — everything in spec was built, nothing beyond spec was built. Check "Must NOT do" compliance.
  Output: `Tasks [N/N compliant] | Contamination [CLEAN/N issues] | VERDICT`

---

## Commit Strategy

- **1**: `fix(embed): copy frontend build to embed location` — internal/embed/web/dist/*
- **2**: `fix(embed): correct embed pattern syntax` — internal/embed/embed.go
- **3**: `fix(server): correct static file path reference` — internal/server/static.go
- **4**: `chore(server): remove duplicate api routes file` — internal/server/common_api.go (deleted)
- **5-6**: `chore: verify build and server startup` — no file changes

---

## Success Criteria

### Verification Commands
```bash
go build ./...                           # Expected: exit 0, no output
go run cmd/server/main.go &              # Expected: "Server starting on port 8080"
sleep 3
curl -s http://localhost:8080/health     # Expected: {"status":"ok"} or 200 OK
```

### Final Checklist
- [ ] All "Must Have" present
- [ ] All "Must NOT Have" absent
- [ ] All tests pass (if any existing tests)
- [ ] Server starts successfully
- [ ] Health endpoint returns 200 OK