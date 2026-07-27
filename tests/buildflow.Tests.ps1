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
