# Linux Windows Continuous Build Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build and test OMRON-MCP on standard GitHub-hosted Ubuntu runners, publish a Windows AMD64 artifact for verification builds, and maintain an attested `continuous` prerelease after successful `main` builds.

**Architecture:** A shared strict Bash script owns the MinGW cross-build contract. A read-only CI workflow verifies `develop`, pull requests, and manual runs; a separate least-privilege publication workflow independently rebuilds `main`, attests the executable digest, and updates the moving `continuous` prerelease.

**Tech Stack:** GitHub Actions, `ubuntu-24.04`, Go 1.26 from `go.mod`, Bash, Ubuntu MinGW-w64, Xvfb, GitHub CLI, `actions/attest@v4`, actionlint.

## Global Constraints

- Use only standard GitHub-hosted `ubuntu-24.04` runners.
- Do not use self-hosted runners, custom VMs, custom Docker images, or Wine.
- Verification runs on pushes to `develop`, all pull requests, and `workflow_dispatch`.
- Publication runs only on pushes to `main`.
- CI has only `contents: read`.
- Publication has only `contents: write`, `id-token: write`, and `attestations: write`.
- Use `actions/checkout@v6`, `actions/setup-go@v6`, `actions/upload-artifact@v4`, and `actions/attest@v4`.
- Cross-build with `GOOS=windows`, `GOARCH=amd64`, `CGO_ENABLED=1`, `CC=x86_64-w64-mingw32-gcc-posix`, and `CXX=x86_64-w64-mingw32-g++-posix`.
- Output `dist/omron-mcp-windows-amd64.exe`.
- Artifact name is `omron-mcp-windows-amd64` with 14-day retention.
- Continuous release assets are the executable, `SHA256SUMS.txt`, and `omron-mcp-windows-amd64.attestation.json`.
- Application, GUI, MCP protocol, and MCP tool behavior remain unchanged.
- Test executable script behavior with controlled fake tools; do not test workflow YAML or human documentation by grepping source text.
- Validate workflow configuration with `actionlint` and its behavior with a real GitHub-hosted `ubuntu-24.04` run.

---

### Task 1: Define the Cross-Build Script Behavior

**Files:**
- Create: `scripts/build-windows_test.sh`

**Interfaces:**
- Consumes: `scripts/build-windows.sh` through a controlled `PATH` containing fake MinGW and Go executables
- Produces: behavioral tests proving correct target selection, environment propagation, output creation, and rejection of a non-AMD64 compiler target

- [ ] **Step 1: Write the failing behavioral tests**

Create `scripts/build-windows_test.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
script="$repo_root/scripts/build-windows.sh"

fail() {
  echo "FAIL: $*" >&2
  exit 1
}

make_fake_tools() {
  local fake_bin="$1"
  mkdir -p "$fake_bin"

  cat >"$fake_bin/x86_64-w64-mingw32-gcc-posix" <<'EOF'
#!/usr/bin/env bash
if [[ "${1:-}" == "-dumpmachine" ]]; then
  printf '%s\n' "${FAKE_COMPILER_TARGET:-x86_64-w64-mingw32}"
fi
EOF

  cat >"$fake_bin/x86_64-w64-mingw32-g++-posix" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF

  cat >"$fake_bin/go" <<'EOF'
#!/usr/bin/env bash
printf 'GOOS=%s GOARCH=%s CGO_ENABLED=%s CC=%s CXX=%s ARGS=%s\n' \
  "$GOOS" "$GOARCH" "$CGO_ENABLED" "$CC" "$CXX" "$*" >"$TEST_LOG"
output=""
previous=""
for argument in "$@"; do
  if [[ "$previous" == "-o" ]]; then
    output="$argument"
    break
  fi
  previous="$argument"
done
[[ -n "$output" ]] || exit 2
mkdir -p "$(dirname "$output")"
printf 'MZ' >"$output"
EOF

  chmod +x "$fake_bin"/*
}

test_successful_build_contract() (
  local temp_dir fake_bin work_dir log
  temp_dir="$(mktemp -d)"
  trap 'rm -rf "$temp_dir"' EXIT
  fake_bin="$temp_dir/bin"
  work_dir="$temp_dir/work"
  log="$temp_dir/go.log"
  mkdir -p "$work_dir"
  make_fake_tools "$fake_bin"

  (
    cd "$work_dir"
    PATH="$fake_bin:$PATH" TEST_LOG="$log" "$script"
  )

  [[ -s "$work_dir/dist/omron-mcp-windows-amd64.exe" ]] ||
    fail "build script did not create a non-empty Windows executable"
  grep -Fq 'GOOS=windows GOARCH=amd64 CGO_ENABLED=1' "$log" ||
    fail "build script did not pass the Windows CGo environment"
  grep -Fq -- '-mod=readonly -ldflags -H=windowsgui -o dist/omron-mcp-windows-amd64.exe ./cmd/omron-mcp' "$log" ||
    fail "build script invoked go with unexpected arguments"
)

test_wrong_compiler_target_is_rejected() (
  local temp_dir fake_bin work_dir output
  temp_dir="$(mktemp -d)"
  trap 'rm -rf "$temp_dir"' EXIT
  fake_bin="$temp_dir/bin"
  work_dir="$temp_dir/work"
  mkdir -p "$work_dir"
  make_fake_tools "$fake_bin"

  if output="$(
    cd "$work_dir"
    PATH="$fake_bin:$PATH" \
      TEST_LOG="$temp_dir/go.log" \
      FAKE_COMPILER_TARGET="i686-w64-mingw32" \
      "$script" 2>&1
  )"; then
    fail "build script accepted a non-AMD64 compiler target"
  fi
  grep -Fq 'does not target Windows AMD64' <<<"$output" ||
    fail "build script returned the wrong compiler-target error"
  [[ ! -e "$temp_dir/go.log" ]] ||
    fail "build script invoked go after rejecting the compiler"
)

test_successful_build_contract
test_wrong_compiler_target_is_rejected
echo "build-windows.sh behavior: PASS"
```

- [ ] **Step 2: Run the behavioral tests to verify RED**

Run:

```powershell
& 'C:\Program Files\Git\bin\bash.exe' scripts/build-windows_test.sh
```

Expected: FAIL because `scripts/build-windows.sh` does not exist.

- [ ] **Step 3: Commit the failing tests**

```powershell
git add -- scripts/build-windows_test.sh
git update-index --chmod=+x scripts/build-windows_test.sh
git commit -m "test: define Windows cross-build behavior"
```

### Task 2: Add the Shared Windows Cross-Build Script

**Files:**
- Create: `scripts/build-windows.sh`
- Test: `scripts/build-windows_test.sh`

**Interfaces:**
- Consumes: optional `CC` and `CXX` environment overrides, Go toolchain from `PATH`, package `./cmd/omron-mcp`
- Produces: non-empty `dist/omron-mcp-windows-amd64.exe`

- [ ] **Step 1: Create the minimal strict cross-build script**

Create `scripts/build-windows.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail

CC="${CC:-x86_64-w64-mingw32-gcc-posix}"
CXX="${CXX:-x86_64-w64-mingw32-g++-posix}"
OUTPUT="dist/omron-mcp-windows-amd64.exe"

command -v "$CC" >/dev/null 2>&1 || {
  echo "C compiler not found: $CC" >&2
  exit 1
}
command -v "$CXX" >/dev/null 2>&1 || {
  echo "C++ compiler not found: $CXX" >&2
  exit 1
}

compiler_target="$("$CC" -dumpmachine)"
case "$compiler_target" in
  x86_64-w64-mingw32*) ;;
  *)
    echo "C compiler does not target Windows AMD64: $compiler_target" >&2
    exit 1
    ;;
esac

mkdir -p dist
export GOOS=windows
export GOARCH=amd64
export CGO_ENABLED=1
export CC
export CXX

go build -mod=readonly -ldflags "-H=windowsgui" -o "$OUTPUT" ./cmd/omron-mcp
test -s "$OUTPUT" || {
  echo "Windows build did not produce a non-empty executable: $OUTPUT" >&2
  exit 1
}

echo "Built $OUTPUT with $compiler_target"
```

- [ ] **Step 2: Run the focused test**

Run:

```powershell
& 'C:\Program Files\Git\bin\bash.exe' scripts/build-windows_test.sh
```

Expected: `build-windows.sh behavior: PASS`.

- [ ] **Step 3: Validate Bash syntax**

Run from an environment with Bash:

```bash
bash -n scripts/build-windows.sh
```

Expected: exit code 0 with no output.

- [ ] **Step 4: Commit the script**

```powershell
git add -- scripts/build-windows.sh
git update-index --chmod=+x scripts/build-windows.sh
git commit -m "build: add Linux Windows cross-build script"
```

### Task 3: Add the Read-Only Verification Workflow

**Files:**
- Create: `.github/workflows/ci.yml`
- Verify: actionlint and the draft pull request's real GitHub-hosted runner

**Interfaces:**
- Consumes: pushes to `develop`, all pull requests, and `workflow_dispatch`
- Produces: verified Actions artifact `omron-mcp-windows-amd64`

- [ ] **Step 1: Create `.github/workflows/ci.yml`**

```yaml
name: CI

on:
  push:
    branches:
      - develop
  pull_request:
  workflow_dispatch:

permissions:
  contents: read

concurrency:
  group: ci-${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true

jobs:
  test-and-build:
    runs-on: ubuntu-24.04
    steps:
      - name: Check out repository
        uses: actions/checkout@v6

      - name: Set up Go
        uses: actions/setup-go@v6
        with:
          go-version-file: go.mod
          cache: true
          cache-dependency-path: go.sum

      - name: Install Linux and Windows build dependencies
        run: |
          sudo apt-get update
          sudo apt-get install --yes --no-install-recommends \
            gcc \
            gcc-mingw-w64-x86-64 \
            g++-mingw-w64-x86-64 \
            libgl1-mesa-dev \
            xorg-dev \
            xvfb

      - name: Test all packages
        run: xvfb-run --auto-servernum go test ./... -count=1

      - name: Test cross-build behavior
        run: ./scripts/build-windows_test.sh

      - name: Cross-build Windows executable
        run: ./scripts/build-windows.sh

      - name: Upload Windows executable
        uses: actions/upload-artifact@v4
        with:
          name: omron-mcp-windows-amd64
          path: dist/omron-mcp-windows-amd64.exe
          retention-days: 14
          if-no-files-found: error
```

- [ ] **Step 2: Validate the CI workflow**

```powershell
go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12 .github/workflows/ci.yml
```

Expected: exit code 0 with no workflow diagnostics.

- [ ] **Step 3: Commit the CI workflow**

```powershell
git add -- .github/workflows/ci.yml
git commit -m "ci: cross-build Windows artifact on Linux"
```

### Task 4: Add Attested Continuous Publication

**Files:**
- Create: `.github/workflows/continuous-release.yml`
- Verify: actionlint and the post-merge `main` workflow run

**Interfaces:**
- Consumes: successful pushes to `main`
- Produces: workflow artifact, GitHub provenance attestation, moving `continuous` tag, and three-asset prerelease

- [ ] **Step 1: Create `.github/workflows/continuous-release.yml`**

```yaml
name: Continuous release

on:
  push:
    branches:
      - main

permissions:
  contents: write
  id-token: write
  attestations: write

concurrency:
  group: continuous-release
  cancel-in-progress: false

jobs:
  build-attest-publish:
    runs-on: ubuntu-24.04
    steps:
      - name: Check out repository
        uses: actions/checkout@v6

      - name: Set up Go
        uses: actions/setup-go@v6
        with:
          go-version-file: go.mod
          cache: true
          cache-dependency-path: go.sum

      - name: Install Linux and Windows build dependencies
        run: |
          sudo apt-get update
          sudo apt-get install --yes --no-install-recommends \
            gcc \
            gcc-mingw-w64-x86-64 \
            g++-mingw-w64-x86-64 \
            libgl1-mesa-dev \
            xorg-dev \
            xvfb

      - name: Test all packages
        run: xvfb-run --auto-servernum go test ./... -count=1

      - name: Test cross-build behavior
        run: ./scripts/build-windows_test.sh

      - name: Cross-build Windows executable
        run: ./scripts/build-windows.sh

      - name: Create SHA-256 checksum
        run: |
          cd dist
          sha256sum omron-mcp-windows-amd64.exe > SHA256SUMS.txt

      - name: Upload verified workflow artifact
        uses: actions/upload-artifact@v4
        with:
          name: omron-mcp-windows-amd64
          path: |
            dist/omron-mcp-windows-amd64.exe
            dist/SHA256SUMS.txt
          retention-days: 14
          if-no-files-found: error

      - name: Register build provenance
        id: attest
        uses: actions/attest@v4
        with:
          subject-checksums: dist/SHA256SUMS.txt

      - name: Prepare attestation bundle
        env:
          BUNDLE_PATH: ${{ steps.attest.outputs.bundle-path }}
        run: cp "$BUNDLE_PATH" dist/omron-mcp-windows-amd64.attestation.json

      - name: Publish continuous prerelease
        env:
          GH_TOKEN: ${{ github.token }}
          SOURCE_SHA: ${{ github.sha }}
          RUN_URL: ${{ github.server_url }}/${{ github.repository }}/actions/runs/${{ github.run_id }}
        run: |
          notes=$(printf 'Built from `%s` by [GitHub Actions](%s).\n\nVerify with:\n\n```text\ngh attestation verify omron-mcp-windows-amd64.exe --repo %s\n```\n' \
            "$SOURCE_SHA" "$RUN_URL" "$GITHUB_REPOSITORY")
          assets=(
            "dist/omron-mcp-windows-amd64.exe"
            "dist/SHA256SUMS.txt"
            "dist/omron-mcp-windows-amd64.attestation.json"
          )
          current_main_sha=$(gh api \
            "repos/${GITHUB_REPOSITORY}/git/ref/heads/main" \
            --jq '.object.sha')
          if [[ "$current_main_sha" != "$SOURCE_SHA" ]]; then
            echo "Skipping publication: $SOURCE_SHA is no longer the current main commit ($current_main_sha)."
            exit 0
          fi

          if gh release view continuous >/dev/null 2>&1; then
            gh release upload continuous "${assets[@]}" --clobber
            gh api --method PATCH \
              "repos/${GITHUB_REPOSITORY}/git/refs/tags/continuous" \
              -f sha="$SOURCE_SHA" \
              -F force=true
            gh release edit continuous \
              --prerelease \
              --title "Continuous Windows build" \
              --notes "$notes"
          else
            gh release create continuous "${assets[@]}" \
              --target "$SOURCE_SHA" \
              --prerelease \
              --title "Continuous Windows build" \
              --notes "$notes"
          fi
```

- [ ] **Step 2: Validate both workflows**

```powershell
go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12
```

Expected: exit code 0 with no workflow diagnostics.

- [ ] **Step 3: Commit the publication workflow**

```powershell
git add -- .github/workflows/continuous-release.yml
git commit -m "ci: publish attested continuous Windows build"
```

### Task 5: Remove Obsolete CI and Document the Flow

**Files:**
- Delete: `.gitlab-ci.yml`
- Modify: `README.md`

**Interfaces:**
- Consumes: exact workflow, artifact, release, and verification names from Tasks 3 and 4
- Produces: contributor instructions and consumer verification instructions

- [ ] **Step 1: Remove `.gitlab-ci.yml`**

Delete the file because it still invokes the removed `./internal/waterjet` package and is not part of the GitHub buildflow.

- [ ] **Step 2: Replace the README with the implemented workflow documentation**

Use this content:

````markdown
# OMRON-MCP

[![CI](https://github.com/rjboer/OMRON-MCP/actions/workflows/ci.yml/badge.svg)](https://github.com/rjboer/OMRON-MCP/actions/workflows/ci.yml)

OMRON-MCP is a Go-based MCP server and desktop application for inspecting and working with OMRON Sysmac Studio projects.

The project is under active development. Please open a GitHub issue when you encounter a problem.

## Development flow

Development happens on `develop`. Pull requests into `main` are tested and cross-built on a standard GitHub-hosted Ubuntu runner. Merging into `main` repeats the complete build and updates the registered continuous Windows release.

## Local Windows verification

Run all Go package tests with a detected 64-bit MinGW compiler:

```powershell
.\test.ps1
```

Build the local Windows executable:

```powershell
.\build.ps1
```

The local executable is written to `bin\omron-mcp.exe`.

## GitHub build artifacts

Pushes to `develop`, pull requests, and manual CI runs test every package and cross-build Windows AMD64 on `ubuntu-24.04`.

After a successful run, download the `omron-mcp-windows-amd64` artifact from the workflow run. Verification artifacts are retained for 14 days.

## Continuous Windows release

Every successful push or merge to `main` updates the [`continuous` prerelease](https://github.com/rjboer/OMRON-MCP/releases/tag/continuous) with:

- `omron-mcp-windows-amd64.exe`;
- `SHA256SUMS.txt`;
- `omron-mcp-windows-amd64.attestation.json`.

Verify the checksum with a SHA-256 tool, or verify the registered GitHub build provenance:

```powershell
gh attestation verify .\omron-mcp-windows-amd64.exe --repo rjboer/OMRON-MCP
```

The attestation binds the executable digest to its repository, workflow, source commit, and triggering event.
````

- [ ] **Step 3: Verify the cleanup and documented targets**

```powershell
if (Test-Path -LiteralPath .gitlab-ci.yml) { throw '.gitlab-ci.yml still exists' }
Select-String -LiteralPath README.md -Pattern 'develop','main','omron-mcp-windows-amd64','continuous','SHA256SUMS.txt','gh attestation verify'
```

Expected: `.gitlab-ci.yml` is absent and every documented identifier is found.

- [ ] **Step 4: Commit cleanup and documentation**

```powershell
git add -- README.md .gitlab-ci.yml
git commit -m "docs: document continuous Windows artifacts"
```

### Task 6: Validate Locally and on GitHub

**Files:**
- Verify: `scripts/build-windows.sh`
- Verify: `scripts/build-windows_test.sh`
- Verify: `.github/workflows/ci.yml`
- Verify: `.github/workflows/continuous-release.yml`
- Verify: `README.md`

**Interfaces:**
- Consumes: completed branch
- Produces: local verification evidence and a draft pull request whose CI performs the real `ubuntu-24.04` cross-build

- [ ] **Step 1: Validate both workflows**

```powershell
go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12
```

Expected: exit code 0 with no workflow diagnostics.

- [ ] **Step 2: Validate Bash syntax**

```bash
bash -n scripts/build-windows.sh scripts/build-windows_test.sh
```

Expected: exit code 0 with no output.

- [ ] **Step 3: Run the cross-build behavior tests**

```powershell
& 'C:\Program Files\Git\bin\bash.exe' scripts/build-windows_test.sh
```

Expected: `build-windows.sh behavior: PASS`.

- [ ] **Step 4: Run the complete Go suite locally**

On the current Windows workstation:

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File .\test.ps1
```

Expected: every Go package passes using the detected 64-bit TDM-GCC compiler.

- [ ] **Step 5: Confirm repository state**

```powershell
git diff --check
git status --short
git log --oneline origin/main..HEAD
```

Expected: no whitespace errors, a clean worktree, and only the intended design/buildflow commits.

- [ ] **Step 6: Publish the branch for real Linux verification**

Use the `superpowers:finishing-a-development-branch` and `github:yeet` skills. Push `develop/linux-windows-continuous-build`, create a draft pull request to `main`, and wait for the `CI / test-and-build` check.

Expected GitHub evidence:

- the job runs on `ubuntu-24.04`;
- all Go tests pass under Xvfb;
- MinGW cross-compilation succeeds;
- `omron-mcp-windows-amd64` is downloadable and contains a non-empty `.exe`.

- [ ] **Step 7: Validate publication only after user-approved merge**

Do not merge automatically. After the user approves and merges the pull request to `main`, monitor `Continuous release / build-attest-publish` and verify:

```powershell
gh release view continuous --repo rjboer/OMRON-MCP
gh release download continuous --repo rjboer/OMRON-MCP --pattern "omron-mcp-windows-amd64.exe" --pattern "SHA256SUMS.txt" --pattern "omron-mcp-windows-amd64.attestation.json"
gh attestation verify .\omron-mcp-windows-amd64.exe --repo rjboer/OMRON-MCP
```

Expected: the release points to the merged `main` commit, all three assets download, the checksum matches, and GitHub verifies the provenance.

- [ ] **Step 8: Establish the `develop` branch after integration**

The remote currently has no `develop` branch. After the buildflow is merged and `main` publication is verified, create `develop` from the verified `main` commit:

```powershell
git fetch origin
git push origin origin/main:refs/heads/develop
```

Expected: `origin/develop` and `origin/main` initially point to the same verified commit, enabling the documented `develop` to `main` flow.
