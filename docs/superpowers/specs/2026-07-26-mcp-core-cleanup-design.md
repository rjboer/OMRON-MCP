# MCP Core Cleanup Design

**Date:** 2026-07-26

## Goal

Align the desktop GUI with the approved five-tab product design, replace the
handwritten MCP protocol implementation with the official Go SDK, decompose
the concentrated Sysmac tool layer, and remove product-specific Waterjet logic
from the generic Sysmac MCP product.

## Scope

This cleanup changes product composition and internal architecture without
removing any of the 18 generic Sysmac MCP tools.

The approved GUI contains exactly these tabs, in this order:

1. Project Discovery
2. Project Explorer
3. Validation / Build
4. GitLab
5. MCP Connection

The following GUI features are removed:

- Entity Editor
- Change Activity
- Structured Text preview and apply controls
- Native Structured Text reads and writes initiated by the GUI
- GUI-only editor, preview, selection-capability, and activity state

Structured Text operations remain available through the MCP tool interface.

## MCP Protocol Architecture

The server will use the official
[`modelcontextprotocol/go-sdk`](https://github.com/modelcontextprotocol/go-sdk)
instead of maintaining a handwritten JSON-RPC/MCP lifecycle implementation.
The implementation will use Go SDK `v1.6.0`, which supports MCP revision `2025-11-25`
and the older revisions documented by the SDK compatibility matrix.

The official SDK owns:

- stdio transport framing
- initialization lifecycle state
- protocol-version negotiation
- client and server capability exchange
- rejection of invalid pre-initialization operations
- JSON-RPC request and notification handling

The application owns:

- server identity and instructions
- declaration of the tools capability
- registration of the Sysmac tools
- conversion between SDK request/result types and Sysmac domain operations
- domain-level validation and execution errors

The server advertises only capabilities it implements. It must not claim
resources, prompts, sampling, elicitation, logging, tasks, or other optional
features unless those features are added and tested separately.

The normative lifecycle is:

1. The client sends `initialize` with `protocolVersion`, `capabilities`, and
   `clientInfo`.
2. Client and server agree on one supported protocol version.
3. The server returns its implementation information and tools capability.
4. The client sends `notifications/initialized`.
5. Normal requests such as `tools/list` and `tools/call` may proceed.

This follows the official
[`2025-11-25` lifecycle specification](https://modelcontextprotocol.io/specification/2025-11-25/basic/lifecycle).

## Health Probe

The GUI's MCP health probe remains a real end-to-end stdio check. It will use a
standards-compliant client flow rather than emitting incomplete JSON by hand:

1. Start the configured MCP executable in MCP mode.
2. Send an initialize request containing the current supported protocol
   version, empty client capabilities, and probe client information.
3. Validate the negotiated version and returned server information.
4. Send `notifications/initialized`.
5. Send `tools/list`.
6. Report connected status, negotiated protocol version, server identity, and
   tool count.

Failure at any stage produces a disconnected/error health result with the
underlying error preserved.

## Sysmac Tool Organization

All existing tool names and behaviors are retained. Tool definitions and
handlers will be registered by focused components rather than returned from
one list and dispatched by one 18-case switch.

### Build and project tools

- `sysmac_nexcc_build`
- `sysmac_project_build`
- `sysmac_project_snapshot`

### Inspection and Structured Text tools

- `sysmac_project_inspect`
- `sysmac_structured_text_read`
- `sysmac_structured_text_preview`
- `sysmac_structured_text_update`

### SLWD tools

- `sysmac_slwd_read`
- `sysmac_slwd_write`

### Entity tools

- `sysmac_entity_mutate`
- `sysmac_entity_payload_read`
- `sysmac_entity_payload_write`

### Transaction tools

- `sysmac_transaction_start`
- `sysmac_transaction_stage_file`
- `sysmac_transaction_delete_file`
- `sysmac_transaction_preview`
- `sysmac_transaction_commit`
- `sysmac_transaction_rollback`

Each component registers typed SDK handlers and contains only the argument
types, result types, schemas, and execution logic for its own tool group.
Shared result conversion and transaction storage remain in small common files.
Tool execution errors remain MCP tool results with `isError: true`; malformed
protocol requests and unknown tools are handled as protocol errors by the SDK.

Existing mutation safeguards remain unchanged:

- direct writes require explicit confirmation
- protected-root checks remain enforced
- atomic backup behavior remains enforced
- transaction commits validate source hashes
- previews remain read-only

## Waterjet Removal

The generic product will delete:

- `internal/waterjet`
- `functions/waterjet_pressure_setpoint.go`
- `functions/waterjet_oil_pid.go`
- `functions/waterjet_phase_monitor.go`
- `functions/waterjet_permissives.go`
- `functions/waterjet_pump_sequence.go`
- `internal/mcp/waterjet_native_integration_test.go`

No Waterjet function is currently registered as an MCP tool, so this removal
does not alter the 18-tool MCP interface.

Generic Sysmac tests may continue to use `Waterjet` as inert sample project
data where the name has no product-specific behavior.

## GUI Composition

GUI construction will be separated enough to test the approved tab inventory
without launching a native window. The Project Explorer becomes read-only:
selecting an entity displays its generic identity and type metadata in the
explorer, without source content, editing controls, or write actions.

The Validation / Build and GitLab tabs keep their current capability-boundary
messages. The MCP Connection tab retains executable resolution, health status,
and the manual connection test. No GUI control may invoke a native project
mutation.

## Testing

Implementation follows test-driven development.

Required automated coverage:

- the GUI exposes exactly the approved five tabs in the approved order
- GUI construction contains no Entity Editor or Change Activity tab
- the health probe completes initialize, initialized notification, and
  `tools/list` in order
- supported client protocol revisions negotiate successfully
- when a client requests an unsupported revision, the server selects its
  latest supported revision and the client rejects the connection if it does
  not support that response
- normal tool requests cannot run before initialization completes
- the server advertises the tools capability and no unsupported capabilities
- all 18 existing Sysmac tools remain registered
- representative calls from each tool group preserve current behavior
- mutation confirmation, backup, protected-root, preview, and transaction
  safeguards continue to pass
- the repository contains no Waterjet package or public wrapper

The full verification command on the current Windows workstation is:

```powershell
$env:CC = "C:\TDM-GCC-64\bin\gcc.exe"
$env:CXX = "C:\TDM-GCC-64\bin\g++.exe"
go test ./...
```

The explicit compiler selection is required because the default local `gcc`
resolves to Simcenter Amesim's 32-bit MinGW compiler, which cannot build the
64-bit Fyne/cgo dependency.

## Compatibility and Non-Goals

- No generic Sysmac MCP tool is removed or renamed.
- No Sysmac domain behavior is deliberately changed.
- No new GUI editing workflow is introduced.
- Waterjet is deleted rather than moved into an example or plugin.
- HTTP MCP transport is not added; the product remains a local stdio server.
- PLC online operations remain outside this product.
- GitLab provider implementation remains outside this cleanup.
- Full controller build capability is not expanded beyond the existing
  adapter.

## Acceptance Criteria

The cleanup is complete when:

1. The GUI contains exactly the approved five tabs and no native write path.
2. The MCP server uses the official Go SDK and negotiates the current stable
   `2025-11-25` revision and supported older revisions.
3. MCP initialization must complete before normal tool operation.
4. All 18 generic Sysmac tools remain discoverable and operational.
5. No single file contains all tool definitions and a monolithic dispatch
   switch.
6. Waterjet production code, wrappers, and product-specific integration test
   are absent.
7. The complete test suite passes with the configured 64-bit C compiler.
