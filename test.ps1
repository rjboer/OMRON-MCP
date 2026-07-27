param(
    [string]$CCPath = "",
    [string]$CXXPath = "",
    [string]$GoToolchain = "go1.26.5"
)

$ErrorActionPreference = "Stop"

function Test-MingwCompiler {
    param([string]$CompilerPath)

    if (-not (Test-Path -LiteralPath $CompilerPath -PathType Leaf)) { return $false }
    $machine = & $CompilerPath -dumpmachine 2>$null
    return $LASTEXITCODE -eq 0 -and $machine -match "^x86_64.*mingw"
}

$candidates = @(
    "C:\TDM-GCC-64\bin\gcc.exe",
    "C:\msys64\ucrt64\bin\gcc.exe",
    "C:\msys64\mingw64\bin\gcc.exe"
)

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

$env:CC = $CCPath
$env:CXX = $CXXPath
$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "1"
$env:GOTOOLCHAIN = "$GoToolchain+auto"
$env:GOTELEMETRY = "off"
$env:PATH = "$(Split-Path -Parent $CCPath);$env:PATH"

Write-Host "Using compiler: $CCPath"
Write-Host "Using Go: $(& go version)"
go test ./... -count=1
