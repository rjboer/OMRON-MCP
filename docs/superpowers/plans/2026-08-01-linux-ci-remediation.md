# Linux CI Remediation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make PR #2 pass on the standard GitHub-hosted Ubuntu runner and verify that it produces the requested Windows AMD64 artifact.

**Architecture:** Keep application behavior unchanged. Make the launcher tests express the existing host-native `filepath` contract, and explicitly select GLFW's X11 backend for the Xvfb-based Linux test step so Wayland headers are not required. Then validate locally, push the focused fix, and inspect the exact-head GitHub artifact.

**Tech Stack:** Go 1.26.5, Go `path/filepath`, GitHub Actions `ubuntu-24.04`, Xvfb/X11, Git Bash 5.2, GitHub CLI 2.96, actionlint 1.7.12.

## Global Constraints

- Do not change application, GUI, MCP protocol, launcher, or MCP tool behavior.
- Continue using only standard GitHub-hosted `ubuntu-24.04` runners.
- Keep CI and continuous-release test commands identical.
- Preserve the existing Windows AMD64 cross-build and artifact contract.
- Use the existing linked worktree and feature branch.

---

### Task 1: Make Launcher Path Tests Host-Native

**Files:**
- Modify: `internal/mcp/launcher_test.go`
- Test: `internal/mcp/launcher_test.go`

**Interfaces:**
- Consumes: `ResolveExecutable(configured, applicationExecutable string)` and Go's host-native `filepath` rules
- Produces: the same relative-root, no-double-`bin`, and absolute-path assertions on Windows and Linux

- [ ] **Step 1: Record the existing RED evidence**

Use GitHub run `30498335030`, where the current tests fail on Linux because `C:\\...` and backslashes are not Linux path syntax.

- [ ] **Step 2: Replace hardcoded Windows paths with host-native paths**

Import `path/filepath`. In each test, construct the application root with `t.TempDir()`, build configured and expected paths with `filepath.Join`, and use a path under `t.TempDir()` for the absolute-path case.

- [ ] **Step 3: Run the focused tests**

```powershell
go test ./internal/mcp -run ResolveExecutable -count=1
```

Expected: PASS on Windows; the exact-head Ubuntu run later proves Linux behavior.

- [ ] **Step 4: Review the diff**

Confirm only test inputs and expectations changed and `internal/mcp/launcher.go` remains untouched.

- [ ] **Step 5: Commit the portable tests and remediation plan**

```powershell
git add -- internal/mcp/launcher_test.go docs/superpowers/plans/2026-08-01-linux-ci-remediation.md
git commit -m "test: make launcher paths portable"
```

### Task 2: Select the X11 Backend for Xvfb Tests

**Files:**
- Modify: `.github/workflows/ci.yml`
- Modify: `.github/workflows/continuous-release.yml`

**Interfaces:**
- Consumes: GLFW's `x11` build tag and the already-installed `xorg-dev`/`xvfb` packages
- Produces: identical Linux test commands that compile only the X11 backend before the Windows cross-build

- [ ] **Step 1: Record the existing RED evidence**

Use GitHub run `30498335030`, where GLFW's default Linux constraints compile the Wayland source and fail at missing `wayland-client-core.h`.

- [ ] **Step 2: Apply the minimal workflow configuration change**

Change the `Test all packages` command in both workflows to:

```yaml
run: xvfb-run --auto-servernum go test -tags x11 ./... -count=1
```

- [ ] **Step 3: Validate workflow syntax**

```powershell
go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12
```

Expected: exit code 0 with no diagnostics.

- [ ] **Step 4: Verify command parity**

Read both `Test all packages` steps and confirm they are byte-for-byte identical.

- [ ] **Step 5: Commit the workflow fix**

```powershell
git add -- .github/workflows/ci.yml .github/workflows/continuous-release.yml
git commit -m "ci: select X11 for Linux tests"
```

### Task 3: Verify, Publish, and Inspect the Exact Build

**Files:**
- Verify: `internal/mcp/launcher_test.go`
- Verify: `.github/workflows/ci.yml`
- Verify: `.github/workflows/continuous-release.yml`
- Verify: `scripts/build-windows.sh`
- Verify: `scripts/build-windows_test.sh`

**Interfaces:**
- Consumes: the focused remediation diff and PR #2
- Produces: a green exact-head Ubuntu check and a downloaded non-empty PE executable

- [ ] **Step 1: Run local verification**

```powershell
& 'C:\\Program Files\\Git\\bin\\bash.exe' -n scripts/build-windows.sh scripts/build-windows_test.sh
& 'C:\\Program Files\\Git\\bin\\bash.exe' scripts/build-windows_test.sh
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\\test.ps1
git diff --check
```

- [ ] **Step 2: Confirm the focused commits and push the remediation**

```powershell
git status --short
git log --oneline origin/develop/linux-windows-continuous-build..HEAD
git push origin develop/linux-windows-continuous-build
```

- [ ] **Step 3: Monitor the exact-head PR check**

Use `gh pr checks 2 --watch --interval 10` and inspect logs if any step fails.

- [ ] **Step 4: Download and inspect the workflow artifact**

Download `omron-mcp-windows-amd64`, assert that the `.exe` is non-empty, verify its first two bytes are `MZ`, and calculate its SHA-256 digest.

- [ ] **Step 5: Preserve the merge boundary**

Do not merge PR #2. Continuous release and attestation verification remain post-merge work because that workflow runs only on `main`.
