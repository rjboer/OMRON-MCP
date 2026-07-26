# GitHub Build and Release Flow Design

Date: 2026-07-27
Status: Approved design, pending implementation

## Goal

Add a GitHub Actions build flow that verifies every proposed change, produces a downloadable Windows executable, and publishes a GitHub Release for stable version tags.

The flow must use an explicitly selected 64-bit C compiler. It must not depend on whichever `gcc.exe` happens to appear first on the runner's `PATH`, because the desktop application uses CGo through Fyne.

## Scope

The implementation will:

- add a continuous-integration workflow at `.github/workflows/ci.yml`;
- add a tag-driven release workflow at `.github/workflows/release.yml`;
- update `test.ps1` so CI and local users can supply explicit C and C++ compiler paths;
- document the build status, CI artifacts, and release procedure in `README.md`;
- remove the obsolete `.gitlab-ci.yml`, which still tests the removed Waterjet package.

No application, MCP protocol, GUI, or tool behavior changes are part of this work.

## Continuous Integration

### Triggers

`ci.yml` runs for:

- every pull request;
- every push to `main`;
- manual `workflow_dispatch` runs.

Concurrent CI runs for the same branch or pull request use a concurrency group. A newer run cancels an older in-progress run for that same ref.

### Permissions and runner

The workflow declares only:

```yaml
permissions:
  contents: read
```

It runs on `windows-2025`.

### Toolchain

The workflow:

1. checks out the repository with `actions/checkout@v6`;
2. installs the Go version declared by `go.mod` with `actions/setup-go@v6`, including the module cache keyed through `go.sum`;
3. installs an MSYS2 UCRT64 environment with `msys2/setup-msys2@v2` and the `mingw-w64-ucrt-x86_64-gcc` package;
4. derives the compiler paths from the action's `msys2-location` output:
   - `<msys2-location>\ucrt64\bin\gcc.exe`
   - `<msys2-location>\ucrt64\bin\g++.exe`

The compiler paths are passed explicitly to the PowerShell scripts. The workflow does not trust or rewrite the runner-wide compiler selection.

### Verification and artifact

The workflow invokes `test.ps1` with the explicit compiler paths. The script runs `go test ./...`, so every Go package is tested with CGo enabled.

After tests pass, the workflow invokes `build.ps1` with the same compiler paths and produces:

```text
bin/omron-mcp.exe
```

Every successful CI run uploads that executable using `actions/upload-artifact@v4`:

- artifact name: `omron-mcp-windows-amd64`;
- path: `bin/omron-mcp.exe`;
- retention: 14 days;
- missing-file behavior: fail the workflow.

Tests or build failures prevent artifact publication.

## Release Flow

### Trigger and version validation

`release.yml` runs only for pushed tags matching GitHub's `v*.*.*` filter.

Because the GitHub tag filter is a glob rather than a semantic-version parser, an early PowerShell step validates the complete tag against:

```text
^v[0-9]+\.[0-9]+\.[0-9]+$
```

This design publishes stable releases such as `v1.2.3`. Tags containing prerelease or build suffixes are rejected.

### Permissions and build

The workflow declares:

```yaml
permissions:
  contents: write
```

It otherwise uses the same Windows runner, Go setup, explicit UCRT64 compilers, full test command, and build script as CI. A release is never created from an untested executable.

### Release assets

After a successful build, PowerShell calculates the SHA-256 hash of `bin/omron-mcp.exe` and writes a release checksum file:

```text
bin/SHA256SUMS.txt
```

The file contains the lowercase hexadecimal hash, two spaces, and the filename `omron-mcp.exe` in a conventional, machine-readable line:

```text
<sha256>  omron-mcp.exe
```

The workflow uses the GitHub CLI supplied on GitHub-hosted runners and the workflow token to create the release:

```text
gh release create <tag> bin/omron-mcp.exe bin/SHA256SUMS.txt --generate-notes
```

The GitHub Release:

- uses the pushed tag as its tag and title source;
- contains `omron-mcp.exe`;
- contains `SHA256SUMS.txt`;
- uses GitHub-generated release notes;
- is not a draft and not marked as a prerelease.

If tag validation, tests, build, checksum generation, or release creation fails, the job fails and does not report a successful release.

## PowerShell Script Contract

`test.ps1` gains optional `CCPath` and `CXXPath` parameters matching the compiler-path contract already supported by `build.ps1`.

When both paths are supplied, `test.ps1` validates that both files exist and sets `CC` and `CXX` for the test process. When neither is supplied, the existing local 64-bit compiler discovery remains available. Supplying only one path is an error.

The script must reject a compiler that cannot target 64-bit Windows before running the tests. This preserves the existing protection against accidentally selecting Amesim's 32-bit MinGW compiler.

## Documentation

`README.md` gains:

- a build-status badge for `.github/workflows/ci.yml`;
- concise local test and build commands;
- an explanation of where CI artifacts can be downloaded;
- instructions to create a release by pushing a stable tag, for example `v1.2.3`;
- a note that releases contain the executable and its SHA-256 checksum.

## Acceptance Criteria

Implementation is complete when:

1. workflow files parse as valid YAML and contain the specified triggers and least-privilege permissions;
2. CI tests all packages and builds the Windows executable with explicit UCRT64 compiler paths;
3. CI uploads `omron-mcp-windows-amd64` after a successful run;
4. the release workflow rejects non-stable version tags that pass the broad GitHub glob;
5. a valid stable tag tests and builds before publishing the executable and checksum with generated notes;
6. `test.ps1` supports explicit compiler paths without breaking its local fallback;
7. `.gitlab-ci.yml` is removed;
8. `README.md` accurately documents the implemented flow;
9. local `go test ./...` and a local Windows build pass using the known 64-bit compiler.
