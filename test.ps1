param(
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
$ccPath = $candidates | Where-Object { Test-MingwCompiler $_ } | Select-Object -First 1
if ([string]::IsNullOrWhiteSpace($ccPath)) {
    throw "No 64-bit MinGW gcc.exe found. The inherited CC/PATH is ignored."
}
$cxxPath = Join-Path (Split-Path -Parent $ccPath) "g++.exe"
if (-not (Test-MingwCompiler $cxxPath)) {
    throw "The matching 64-bit g++.exe was not found: $cxxPath"
}

$env:CC = $ccPath
$env:CXX = $cxxPath
$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "1"
$env:GOTOOLCHAIN = "$GoToolchain+auto"
$env:GOTELEMETRY = "off"
$env:PATH = "$(Split-Path -Parent $ccPath);$env:PATH"

Write-Host "Using compiler: $ccPath"
Write-Host "Using Go: $(& go version)"
go test ./... -count=1
