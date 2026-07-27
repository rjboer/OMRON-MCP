# OMRON-MCP

[![CI](https://github.com/rjboer/OMRON-MCP/actions/workflows/ci.yml/badge.svg)](https://github.com/rjboer/OMRON-MCP/actions/workflows/ci.yml)

OMRON-MCP is a Go-based MCP server and desktop application for inspecting and working with OMRON Sysmac Studio projects.

The project is under active development. Please open a GitHub issue when you encounter a problem.

## Requirements

- Windows on AMD64
- Go as declared in `go.mod`
- A 64-bit MinGW compiler for the Fyne/CGo desktop application

The build scripts discover TDM-GCC-64 or a 64-bit MSYS2 compiler locally. You can also pass the compiler paths explicitly, which avoids selecting an incompatible 32-bit compiler from `PATH`.

## Test

Run all Go package tests with explicit compiler paths:

```powershell
.\test.ps1 -CCPath "C:\TDM-GCC-64\bin\gcc.exe" -CXXPath "C:\TDM-GCC-64\bin\g++.exe"
```

## Build

Build the Windows executable:

```powershell
.\build.ps1 -CCPath "C:\TDM-GCC-64\bin\gcc.exe" -CXXPath "C:\TDM-GCC-64\bin\g++.exe"
```

The executable is written to `bin\omron-mcp.exe`.

## Continuous integration

GitHub Actions tests every package and builds the Windows executable for pull requests, pushes to `main`, and manual workflow runs.

After a successful run, download the `omron-mcp-windows-amd64` artifact from that workflow run. CI artifacts are retained for 14 days.

## Releases

Push a stable version tag to create a GitHub Release:

```powershell
git tag v1.2.3
git push origin v1.2.3
```

The release workflow tests and rebuilds the project before publishing `omron-mcp.exe` and `SHA256SUMS.txt` with generated release notes. Tags with prerelease or build suffixes are not accepted.
