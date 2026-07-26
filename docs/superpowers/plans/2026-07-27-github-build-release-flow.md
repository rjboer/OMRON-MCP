# GitHub Build and Release Flow Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a least-privilege GitHub Actions flow that tests and builds OMRON-MCP on Windows and publishes stable tagged releases with a checksum.

**Architecture:** Two focused workflows share the same explicit Go and UCRT64 compiler setup but have separate triggers and permissions. A PowerShell acceptance test checks the repository-level workflow contract, while `test.ps1` remains the single full-suite entry point for local and CI use.

**Tech Stack:** GitHub Actions, PowerShell 7, Go 1.26 toolchain selection, CGo, MSYS2 UCRT64 GCC, Pester 3.4-compatible tests, GitHub CLI.

## Global Constraints

- CI runs on pull requests, pushes to `main`, and manual dispatch.
- Release runs only for pushed `v*.*.*` tags and accepts only `^v[0-9]+\.[0-9]+\.[0-9]+$`.
- CI has `contents: read`; release has `contents: write`.
- Both workflows run on `windows-2025`.
- Both workflows use `actions/checkout@v6`, `actions/setup-go@v6`, and `msys2/setup-msys2@v2`.
- MSYS2 installs `mingw-w64-ucrt-x86_64-gcc`; compiler paths come from `msys2-location`.
- CI artifact name is `omron-mcp-windows-amd64`, retained for 14 days.
- Release assets are `omron-mcp.exe` and `SHA256SUMS.txt`; release notes are generated.
- Application, GUI, MCP protocol, and MCP tool behavior remain unchanged.

---

### Task 1: Add Failing Buildflow Contract Tests

**Files:**
- Create: `tests/buildflow.Tests.ps1`

**Interfaces:**
- Consumes: repository files relative to `$PSScriptRoot\..`
- Produces: Pester acceptance tests covering workflow structure, script parameters, obsolete CI removal, and documentation

- [ ] **Step 1: Write the failing workflow contract tests**

Create `tests/buildflow.Tests.ps1` with Pester 3.4-compatible assertions:

```powershell
$repoRoot = Split-Path -Parent $PSScriptRoot

function Read-RepoFile([string]$RelativePath) {
    Get-Content -LiteralPath (Join-Path $repoRoot $RelativePath) -Raw
}

Describe "GitHub buildflow contract" {
    It "defines the CI triggers, runner, permissions, toolchain, test, build, and artifact" {
        $ci = Read-RepoFile ".github/workflows/ci.yml"
        $ci | Should Match "(?m)^  pull_request:"
        $ci | Should Match "(?m)^  push:"
        $ci | Should Match "(?m)^      - main$"
        $ci | Should Match "(?m)^  workflow_dispatch:"
        $ci | Should Match "(?m)^  contents: read$"
        $ci | Should Match "runs-on: windows-2025"
        $ci | Should Match "actions/checkout@v6"
        $ci | Should Match "actions/setup-go@v6"
        $ci | Should Match "cache-dependency-path: go.sum"
        $ci | Should Match "msys2/setup-msys2@v2"
        $ci | Should Match "mingw-w64-ucrt-x86_64-gcc"
        $ci | Should Match "go env GOVERSION"
        $ci | Should Match "test[.]ps1"
        $ci | Should Match "build[.]ps1"
        $ci | Should Match "actions/upload-artifact@v4"
        $ci | Should Match "name: omron-mcp-windows-amd64"
        $ci | Should Match "retention-days: 14"
        $ci | Should Match "if-no-files-found: error"
    }

    It "defines a stable-tag release with write permission and required assets" {
        $release = Read-RepoFile ".github/workflows/release.yml"
        $release | Should Match "'v[*][.][*][.][*]'"
        $release | Should Match "(?m)^  contents: write$"
        $release | Should Match "runs-on: windows-2025"
        $release | Should Match "actions/checkout@v6"
        $release | Should Match "actions/setup-go@v6"
        $release | Should Match "cache-dependency-path: go.sum"
        $release | Should Match "msys2/setup-msys2@v2"
        $release | Should Match "mingw-w64-ucrt-x86_64-gcc"
        $release | Should Match "go env GOVERSION"
        $release | Should Match ([regex]::Escape('^v[0-9]+\.[0-9]+\.[0-9]+$'))
        $release | Should Match "test[.]ps1"
        $release | Should Match "build[.]ps1"
        $release | Should Match "SHA256SUMS[.]txt"
        $release | Should Match "gh release create"
        $release | Should Match "--generate-notes"
    }

    It "exposes explicit test compiler parameters" {
        $testScript = Read-RepoFile "test.ps1"
        $testScript | Should Match '(?s)param[(].*[$]CCPath.*[$]CXXPath.*[$]GoToolchain'
        $testScript | Should Match "Supplying -CCPath and -CXXPath together is required"
        $testScript | Should Match "go test [.]/[.][.][.] -count=1"
    }

    It "removes obsolete GitLab CI and documents artifacts and tags" {
        Test-Path -LiteralPath (Join-Path $repoRoot ".gitlab-ci.yml") | Should Be $false
        $readme = Read-RepoFile "README.md"
        $readme | Should Match "actions/workflows/ci[.]yml/badge[.]svg"
        $readme | Should Match "omron-mcp-windows-amd64"
        $readme | Should Match "v1[.]2[.]3"
        $readme | Should Match "SHA256SUMS[.]txt"
    }
}
```

- [ ] **Step 2: Run the tests to verify RED**

Run:

```powershell
Invoke-Pester -Script .\tests\buildflow.Tests.ps1
```

Expected: failures because `.github/workflows/ci.yml` and `.github/workflows/release.yml` do not exist, `test.ps1` lacks the parameters, `.gitlab-ci.yml` still exists, and the README lacks buildflow documentation.

- [ ] **Step 3: Commit the failing acceptance test**

```powershell
git add -- tests/buildflow.Tests.ps1
git commit -m "test: define GitHub buildflow contract"
```

### Task 2: Support Explicit Test Compiler Paths

**Files:**
- Modify: `test.ps1:1-36`
- Test: `tests/buildflow.Tests.ps1`

**Interfaces:**
- Consumes: `-CCPath <string>`, `-CXXPath <string>`, and existing `-GoToolchain <string>`
- Produces: a test process with validated `CC`, `CXX`, `GOOS=windows`, `GOARCH=amd64`, and `CGO_ENABLED=1`

- [ ] **Step 1: Implement the minimal parameter contract**

Change the parameter block to:

```powershell
param(
    [string]$CCPath = "",
    [string]$CXXPath = "",
    [string]$GoToolchain = "go1.26.5"
)
```

Before fallback discovery, add:

```powershell
$hasCC = -not [string]::IsNullOrWhiteSpace($CCPath)
$hasCXX = -not [string]::IsNullOrWhiteSpace($CXXPath)
if ($hasCC -xor $hasCXX) {
    throw "Supplying -CCPath and -CXXPath together is required."
}

if (-not $hasCC) {
    $CCPath = $candidates | Where-Object { Test-MingwCompiler $_ } | Select-Object -First 1
    if ([string]::IsNullOrWhiteSpace($CCPath)) {
        throw "No 64-bit MinGW gcc.exe found. The inherited CC/PATH is ignored."
    }
    $CXXPath = Join-Path (Split-Path -Parent $CCPath) "g++.exe"
}
if (-not (Test-MingwCompiler $CCPath)) {
    throw "C compiler is missing or does not target 64-bit MinGW: $CCPath"
}
if (-not (Test-MingwCompiler $CXXPath)) {
    throw "C++ compiler is missing or does not target 64-bit MinGW: $CXXPath"
}
```

Use `$CCPath` and `$CXXPath` for the environment and compiler-bin `PATH`.
Keep the existing full-suite command unchanged: `go test ./... -count=1`.

- [ ] **Step 2: Run the focused test**

Run:

```powershell
Invoke-Pester -Script .\tests\buildflow.Tests.ps1
```

Expected: the explicit compiler parameter test passes; workflow and documentation tests remain red.

- [ ] **Step 3: Exercise the real explicit compiler path**

Run:

```powershell
.\test.ps1 -CCPath "C:\TDM-GCC-64\bin\gcc.exe" -CXXPath "C:\TDM-GCC-64\bin\g++.exe"
```

Expected: all Go packages pass and the output identifies the supplied compiler.

- [ ] **Step 4: Commit the script change**

```powershell
git add -- test.ps1
git commit -m "build: accept explicit test compiler paths"
```

### Task 3: Add the CI Workflow

**Files:**
- Create: `.github/workflows/ci.yml`
- Test: `tests/buildflow.Tests.ps1`

**Interfaces:**
- Consumes: pull requests, pushes to `main`, and manual dispatch
- Produces: tested `bin/omron-mcp.exe` uploaded as `omron-mcp-windows-amd64`

- [ ] **Step 1: Create the CI workflow**

Create `.github/workflows/ci.yml`:

```yaml
name: CI

on:
  pull_request:
  push:
    branches:
      - main
  workflow_dispatch:

permissions:
  contents: read

concurrency:
  group: ci-${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: true

jobs:
  test-and-build:
    runs-on: windows-2025
    steps:
      - name: Check out repository
        uses: actions/checkout@v6

      - name: Set up Go
        uses: actions/setup-go@v6
        with:
          go-version-file: go.mod
          cache: true
          cache-dependency-path: go.sum

      - name: Set up UCRT64 compiler
        id: msys2
        uses: msys2/setup-msys2@v2
        with:
          msystem: UCRT64
          update: true
          install: mingw-w64-ucrt-x86_64-gcc

      - name: Test all packages
        shell: pwsh
        run: |
          $compilerRoot = "${{ steps.msys2.outputs.msys2-location }}\ucrt64\bin"
          $goToolchain = (go env GOVERSION).Trim()
          .\test.ps1 -CCPath "$compilerRoot\gcc.exe" -CXXPath "$compilerRoot\g++.exe" -GoToolchain $goToolchain

      - name: Build Windows executable
        shell: pwsh
        run: |
          $compilerRoot = "${{ steps.msys2.outputs.msys2-location }}\ucrt64\bin"
          $goToolchain = (go env GOVERSION).Trim()
          .\build.ps1 -CCPath "$compilerRoot\gcc.exe" -CXXPath "$compilerRoot\g++.exe" -GoToolchain $goToolchain

      - name: Upload Windows executable
        uses: actions/upload-artifact@v4
        with:
          name: omron-mcp-windows-amd64
          path: bin/omron-mcp.exe
          retention-days: 14
          if-no-files-found: error
```

- [ ] **Step 2: Run the focused test**

Run:

```powershell
Invoke-Pester -Script .\tests\buildflow.Tests.ps1
```

Expected: the CI test passes; release and documentation tests remain red.

- [ ] **Step 3: Commit the CI workflow**

```powershell
git add -- .github/workflows/ci.yml
git commit -m "ci: test and build Windows artifact"
```

### Task 4: Add the Release Workflow

**Files:**
- Create: `.github/workflows/release.yml`
- Test: `tests/buildflow.Tests.ps1`

**Interfaces:**
- Consumes: pushed tags matching `v*.*.*`, with exact stable-tag validation
- Produces: a non-draft GitHub Release containing `omron-mcp.exe` and `SHA256SUMS.txt`

- [ ] **Step 1: Create the release workflow**

Create `.github/workflows/release.yml`:

```yaml
name: Release

on:
  push:
    tags:
      - 'v*.*.*'

permissions:
  contents: write

jobs:
  release:
    runs-on: windows-2025
    steps:
      - name: Validate stable version tag
        shell: pwsh
        env:
          RELEASE_TAG: ${{ github.ref_name }}
        run: |
          if ($env:RELEASE_TAG -notmatch '^v[0-9]+\.[0-9]+\.[0-9]+$') {
            throw "Release tag must be a stable version such as v1.2.3: $env:RELEASE_TAG"
          }

      - name: Check out repository
        uses: actions/checkout@v6

      - name: Set up Go
        uses: actions/setup-go@v6
        with:
          go-version-file: go.mod
          cache: true
          cache-dependency-path: go.sum

      - name: Set up UCRT64 compiler
        id: msys2
        uses: msys2/setup-msys2@v2
        with:
          msystem: UCRT64
          update: true
          install: mingw-w64-ucrt-x86_64-gcc

      - name: Test all packages
        shell: pwsh
        run: |
          $compilerRoot = "${{ steps.msys2.outputs.msys2-location }}\ucrt64\bin"
          $goToolchain = (go env GOVERSION).Trim()
          .\test.ps1 -CCPath "$compilerRoot\gcc.exe" -CXXPath "$compilerRoot\g++.exe" -GoToolchain $goToolchain

      - name: Build Windows executable
        shell: pwsh
        run: |
          $compilerRoot = "${{ steps.msys2.outputs.msys2-location }}\ucrt64\bin"
          $goToolchain = (go env GOVERSION).Trim()
          .\build.ps1 -CCPath "$compilerRoot\gcc.exe" -CXXPath "$compilerRoot\g++.exe" -GoToolchain $goToolchain

      - name: Create SHA-256 checksum
        shell: pwsh
        run: |
          $hash = (Get-FileHash -Algorithm SHA256 -LiteralPath "bin\omron-mcp.exe").Hash.ToLowerInvariant()
          "$hash  omron-mcp.exe" | Set-Content -LiteralPath "bin\SHA256SUMS.txt" -Encoding ascii

      - name: Publish GitHub Release
        shell: pwsh
        env:
          GH_TOKEN: ${{ github.token }}
          RELEASE_TAG: ${{ github.ref_name }}
        run: |
          gh release create "$env:RELEASE_TAG" "bin\omron-mcp.exe" "bin\SHA256SUMS.txt" --generate-notes --title "$env:RELEASE_TAG"
```

- [ ] **Step 2: Run the focused test**

Run:

```powershell
Invoke-Pester -Script .\tests\buildflow.Tests.ps1
```

Expected: CI, release, and script tests pass; only obsolete-CI/documentation assertions remain red.

- [ ] **Step 3: Commit the release workflow**

```powershell
git add -- .github/workflows/release.yml
git commit -m "ci: publish stable tagged releases"
```

### Task 5: Remove Obsolete CI and Document the Buildflow

**Files:**
- Delete: `.gitlab-ci.yml`
- Modify: `README.md:1-11`
- Test: `tests/buildflow.Tests.ps1`

**Interfaces:**
- Consumes: the implemented workflow names, artifact name, and stable tag format
- Produces: accurate contributor and release instructions

- [ ] **Step 1: Delete the obsolete GitLab configuration**

Delete `.gitlab-ci.yml`; it references `./internal/waterjet`, which is no longer present.

- [ ] **Step 2: Replace the README with concise buildflow documentation**

Write sections for project purpose, requirements, local testing, local build, CI artifacts, and releases. Include:

```markdown
[![CI](https://github.com/rjboer/OMRON-MCP/actions/workflows/ci.yml/badge.svg)](https://github.com/rjboer/OMRON-MCP/actions/workflows/ci.yml)
```

Document the verified local commands:

```powershell
.\test.ps1 -CCPath "C:\TDM-GCC-64\bin\gcc.exe" -CXXPath "C:\TDM-GCC-64\bin\g++.exe"
.\build.ps1 -CCPath "C:\TDM-GCC-64\bin\gcc.exe" -CXXPath "C:\TDM-GCC-64\bin\g++.exe"
```

State that successful CI runs expose `omron-mcp-windows-amd64`, and that pushing `v1.2.3` publishes `omron-mcp.exe` plus `SHA256SUMS.txt`.

- [ ] **Step 3: Run all acceptance tests to verify GREEN**

Run:

```powershell
Invoke-Pester -Script .\tests\buildflow.Tests.ps1
```

Expected: 4 tests passed, 0 failed.

- [ ] **Step 4: Commit the cleanup and documentation**

```powershell
git add -- README.md .gitlab-ci.yml
git commit -m "docs: document GitHub build and releases"
```

### Task 6: Validate the Complete Flow Locally

**Files:**
- Verify: `.github/workflows/ci.yml`
- Verify: `.github/workflows/release.yml`
- Verify: `test.ps1`
- Verify: `README.md`

**Interfaces:**
- Consumes: completed repository state
- Produces: fresh evidence that tests, build, executable, checksum format, and repository contract are correct

- [ ] **Step 1: Validate GitHub Actions syntax**

Run the official-source actionlint tool at the reviewed version:

```powershell
go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12
```

Expected: exit code 0 with no workflow errors.

- [ ] **Step 2: Run the contract suite**

```powershell
Invoke-Pester -Script .\tests\buildflow.Tests.ps1
```

Expected: 4 tests passed, 0 failed.

- [ ] **Step 3: Run all Go tests through the production test script**

```powershell
.\test.ps1 -CCPath "C:\TDM-GCC-64\bin\gcc.exe" -CXXPath "C:\TDM-GCC-64\bin\g++.exe"
```

Expected: every package passes.

- [ ] **Step 4: Rebuild the executable through the production build script**

```powershell
.\build.ps1 -CCPath "C:\TDM-GCC-64\bin\gcc.exe" -CXXPath "C:\TDM-GCC-64\bin\g++.exe"
```

Expected: exit code 0 and `bin\omron-mcp.exe` exists.

- [ ] **Step 5: Verify output and repository cleanliness**

```powershell
Get-Item -LiteralPath .\bin\omron-mcp.exe | Select-Object FullName,Length,LastWriteTime
Get-FileHash -Algorithm SHA256 -LiteralPath .\bin\omron-mcp.exe
git diff --check
git status --short
```

Expected: a non-empty executable, a SHA-256 hash, no whitespace errors, and no uncommitted tracked changes.

- [ ] **Step 6: Review the implementation against every acceptance criterion**

Confirm the nine criteria in `docs/superpowers/specs/2026-07-27-github-build-release-flow-design.md` directly against the final files and fresh command output. Do not create or push a release tag during local verification.
