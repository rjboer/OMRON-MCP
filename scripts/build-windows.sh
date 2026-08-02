#!/usr/bin/env bash
set -euo pipefail

CC="${CC:-x86_64-w64-mingw32-gcc-posix}"
CXX="${CXX:-x86_64-w64-mingw32-g++-posix}"
OUTPUT="dist/omron-mcp-windows-amd64.exe"
BUILD_COMMIT="${BUILD_COMMIT:-dev}"

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

go build -mod=readonly -ldflags "-H=windowsgui -X main.buildCommit=$BUILD_COMMIT" -o "$OUTPUT" ./cmd/omron-mcp
test -s "$OUTPUT" || {
  echo "Windows build did not produce a non-empty executable: $OUTPUT" >&2
  exit 1
}

echo "Built $OUTPUT with $compiler_target"
