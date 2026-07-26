param(
    [string]$CCPath = "",
    [string]$CXXPath = "",
[string]$GoToolchain = "go1.26.5",
[string]$OutputDir = "bin",
[string]$OutputName = "",
[string]$Target = "./cmd/omron-mcp"
)

$ErrorActionPreference = "Stop"

function Test-MingwCompiler {
    param([string]$CompilerPath)

    if (-not (Test-Path -LiteralPath $CompilerPath -PathType Leaf)) { return $false }
    $machine = & $CompilerPath -dumpmachine 2>$null
    return $LASTEXITCODE -eq 0 -and $machine -match "^x86_64.*mingw"
}

function Get-BinutilsVersion {
    param([string]$CompilerPath)
    $ldPath = Join-Path (Split-Path -Parent $CompilerPath) "ld.exe"
    if (-not (Test-Path -LiteralPath $ldPath -PathType Leaf)) { throw "GNU linker was not found next to the compiler: $ldPath" }
    # Read the native command to completion before selecting its first line.
    # Piping ld.exe directly to Select-Object -First 1 can close stdout early,
    # causing ld.exe to report exit code -1 even though the version is valid.
    $versionOutput = @(& $ldPath --version 2>&1)
    $exitCode = $LASTEXITCODE
    $firstLine = [string]$versionOutput[0]
    if ($exitCode -ne 0) { throw "Could not determine the GNU linker version: $firstLine" }
    if ($firstLine -notmatch '([0-9]+)[.]([0-9]+)(?:[.]([0-9]+))?') { throw "Could not parse GNU linker version: $firstLine" }
    $patch = if ($Matches[3]) { $Matches[3] } else { "0" }
    return [version]"$($Matches[1]).$($Matches[2]).$patch"
}

$candidates = @(
	"C:\TDM-GCC-64\bin\gcc.exe",
	"C:\msys64\ucrt64\bin\gcc.exe",
	"C:\msys64\mingw64\bin\gcc.exe"
)

if ([string]::IsNullOrWhiteSpace($CCPath)) {
    foreach ($candidate in $candidates) {
        if (Test-MingwCompiler $candidate) {
            $CCPath = $candidate
            break
        }
    }
}
if ([string]::IsNullOrWhiteSpace($CCPath) -or -not (Test-MingwCompiler $CCPath)) {
	throw "No 64-bit MinGW gcc.exe found. The inherited CC/PATH is ignored; install TDM-GCC-64 or pass -CCPath explicitly."
}
if ([string]::IsNullOrWhiteSpace($CXXPath)) {
    $CXXPath = Join-Path (Split-Path -Parent $CCPath) "g++.exe"
}
if (-not (Test-MingwCompiler $CXXPath)) { throw "C++ compiler is missing or does not target 64-bit MinGW: $CXXPath" }

New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null
$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:GOAMD64 = "v1"
$env:CGO_ENABLED = "1"
$env:CC = $CCPath
$env:CXX = $CXXPath
$compilerBin = Split-Path -Parent $CCPath
$env:PATH = "$compilerBin;$env:PATH"
$binutilsVersion = Get-BinutilsVersion -CompilerPath $CCPath
Write-Host "Binutils: $binutilsVersion"
if ($binutilsVersion -lt [version]"2.37.0") {
    Write-Warning "Binutils $binutilsVersion is too old for default Go 1.25+ Windows cgo builds. Enabling GOEXPERIMENT=nodwarf5."
    $env:GOEXPERIMENT = "nodwarf5"
} else {
    Remove-Item Env:GOEXPERIMENT -ErrorAction SilentlyContinue
}
$env:GOTOOLCHAIN = "$GoToolchain+auto"
$env:GOTELEMETRY = "off"

if ([string]::IsNullOrWhiteSpace($OutputName)) {
    $targetName = Split-Path -Leaf $Target
    $OutputName = "$targetName.exe"
}
$outputPath = Join-Path $OutputDir $OutputName

$goVersion = & go version
if ($LASTEXITCODE -ne 0 -or $goVersion -notmatch [regex]::Escape($GoToolchain)) {
    throw "Requested Go toolchain $GoToolchain is not active. Actual: $goVersion"
}
Write-Host "Using compiler: $CCPath"
Write-Host "Using Go: $goVersion"
Write-Host "Writing executable: $outputPath"
go build -mod=readonly -ldflags "-H=windowsgui" -o $outputPath $Target
