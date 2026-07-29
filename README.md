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
