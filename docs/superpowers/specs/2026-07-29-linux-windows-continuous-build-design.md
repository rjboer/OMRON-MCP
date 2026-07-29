# Linux Windows Continuous Build Design

Date: 2026-07-29
Status: Approved design, pending implementation

## Goal

Use standard GitHub-hosted Linux runners to test OMRON-MCP, cross-compile its Fyne/CGo desktop application into a Windows AMD64 executable, and publish a registered continuous build whenever `develop` is merged into `main`.

The flow must require no self-hosted runner, custom virtual machine, or custom Docker image.

## Git Branch Flow

The repository follows this build lifecycle:

1. Work is pushed to `develop`.
2. A pull request from `develop` to `main` runs verification but does not publish a release.
3. Merging the pull request creates a push to `main`.
4. The `main` push reruns all verification from the merged commit.
5. Only after successful tests, cross-compilation, checksum generation, and provenance attestation is the `continuous` prerelease updated.

A failed or cancelled job must never update the published release.

## Runner and Toolchain

Both workflows run on the fixed standard GitHub-hosted runner:

```yaml
runs-on: ubuntu-24.04
```

They use:

- `actions/checkout@v6`;
- `actions/setup-go@v6` with the Go version read from `go.mod` and caching based on `go.sum`;
- Ubuntu repository packages installed with `apt-get`;
- the Ubuntu MinGW-w64 cross-compiler for Windows AMD64.

The required Ubuntu packages are:

- `gcc`;
- `gcc-mingw-w64-x86-64`;
- `g++-mingw-w64-x86-64`;
- `libgl1-mesa-dev`;
- `xorg-dev`;
- `xvfb`.

The Windows cross-build uses explicit environment values:

```text
GOOS=windows
GOARCH=amd64
CGO_ENABLED=1
CC=x86_64-w64-mingw32-gcc-posix
CXX=x86_64-w64-mingw32-g++-posix
```

The workflow verifies the compiler target before building. It must not depend on a compiler selected implicitly from `PATH`.

## Shared Build Script

A repository script at `scripts/build-windows.sh` owns the cross-build command used by both workflows.

The script:

- runs with strict Bash error handling;
- validates that the selected C and C++ compilers exist;
- verifies that the C compiler reports an `x86_64` MinGW target;
- creates `dist/`;
- builds `./cmd/omron-mcp` with `-mod=readonly`;
- uses the Windows GUI linker flag `-H=windowsgui`;
- writes `dist/omron-mcp-windows-amd64.exe`;
- fails if the output is missing or empty.

Keeping the cross-build in a script ensures pull requests and releases cannot silently drift to different compiler settings.

## Verification Workflow

`.github/workflows/ci.yml` runs for:

- pushes to `develop`;
- all pull requests, including `develop` to `main`;
- manual `workflow_dispatch` runs.

It declares only:

```yaml
permissions:
  contents: read
```

The job:

1. checks out the exact commit;
2. installs Go and the Ubuntu build dependencies;
3. runs all Go tests under a virtual X display:

   ```text
   xvfb-run --auto-servernum go test ./... -count=1
   ```

4. invokes `scripts/build-windows.sh`;
5. uploads `dist/omron-mcp-windows-amd64.exe` with `actions/upload-artifact@v4`.

The workflow artifact is:

- named `omron-mcp-windows-amd64`;
- retained for 14 days;
- configured to fail when the expected executable is absent.

CI uses a per-ref concurrency group and cancels an older in-progress verification for the same ref.

## Continuous Publication Workflow

`.github/workflows/continuous-release.yml` runs only on pushes to `main`.

It declares:

```yaml
permissions:
  contents: write
  id-token: write
  attestations: write
```

The job repeats the complete Linux test and Windows cross-build process. It does not consume an executable produced by an earlier pull-request run.

After the build succeeds, the job:

1. creates `dist/SHA256SUMS.txt` in standard `sha256sum` format;
2. uploads the executable and checksum as the `omron-mcp-windows-amd64` workflow artifact;
3. invokes `actions/attest@v4` with `subject-checksums: dist/SHA256SUMS.txt`;
4. copies the returned Sigstore bundle to the stable release filename `dist/omron-mcp-windows-amd64.attestation.json`;
5. creates or updates the GitHub prerelease tagged `continuous`.

The `continuous` release contains:

- `omron-mcp-windows-amd64.exe`;
- `SHA256SUMS.txt`;
- `omron-mcp-windows-amd64.attestation.json`.

The release title is `Continuous Windows build`. Its notes identify the source commit and workflow run. Existing assets with the same names are replaced.

The `continuous` tag is intentionally movable and is updated to the latest successfully verified `main` commit. The provenance registered by GitHub remains tied to the immutable artifact digest and source commit even after a newer continuous build replaces the release assets.

## Artifact Verification

Consumers can verify the downloaded executable in two ways:

1. compare its SHA-256 digest with `SHA256SUMS.txt`;
2. verify its GitHub attestation:

   ```text
   gh attestation verify omron-mcp-windows-amd64.exe --repo rjboer/OMRON-MCP
   ```

The attestation records the repository, workflow, source commit, triggering event, and binary digest.

## Failure and Recovery Behavior

- Test failure prevents the cross-build.
- Cross-build failure prevents checksum generation, attestation, and publication.
- Attestation failure prevents release publication.
- Release update failure marks the workflow failed and leaves the run available for retry.
- Rerunning a failed `main` workflow is safe because release assets use stable names and are replaced idempotently.
- Pull-request workflows never receive release or attestation write permissions.

Updating a moving Git tag and replacing release assets is not an atomic GitHub operation. Publication therefore happens only after every immutable output and attestation exists. If GitHub fails during the final release update, the run fails visibly and a rerun reconciles the tag, notes, and assets.

## Repository Changes

Implementation will:

- add `.github/workflows/ci.yml`;
- add `.github/workflows/continuous-release.yml`;
- add `scripts/build-windows.sh`;
- add Linux-compatible contract tests for workflow triggers, permissions, toolchain settings, artifact naming, attestation, and continuous release behavior;
- remove the obsolete `.gitlab-ci.yml`, which references the deleted Waterjet package;
- update `README.md` with the branch flow, artifact download, continuous release, checksum, and attestation verification instructions.

The local Windows `test.ps1` and `build.ps1` scripts remain available and are not used by GitHub Actions.

No application, GUI, MCP protocol, or MCP tool behavior changes are part of this work.

## Validation

Before publication, implementation must pass:

- the repository's full `go test ./...` suite;
- the Linux-compatible workflow contract tests;
- `actionlint` for both workflow files;
- `bash -n scripts/build-windows.sh`;
- a real Windows AMD64 cross-build on the standard `ubuntu-24.04` GitHub runner;
- confirmation that the generated executable and checksum are non-empty and consistent;
- confirmation that GitHub records an attestation for the executable digest;
- confirmation that the `continuous` prerelease contains all three named assets and points to the merged `main` commit.

## Acceptance Criteria

The work is complete when:

1. pushes to `develop`, pull requests, and manual verification runs use `ubuntu-24.04`, test the project, cross-build the Windows executable, and upload a temporary artifact;
2. a merge or direct push to `main` independently repeats the full verified build;
3. only a successful `main` run can update the `continuous` prerelease;
4. the published executable is named `omron-mcp-windows-amd64.exe`;
5. the executable has a matching `SHA256SUMS.txt`;
6. GitHub stores verifiable provenance for the executable through `actions/attest@v4`;
7. the attestation bundle is available as a release asset;
8. the `continuous` tag points to the successfully built `main` commit;
9. CI has read-only repository permissions and publication alone has the required write permissions;
10. the obsolete GitLab workflow is removed and the README documents the implemented flow.
