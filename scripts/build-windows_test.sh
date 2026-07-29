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
