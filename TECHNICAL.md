# Technical overview

## What is OMRON MCP?

OMRON MCP is an MCP server for OMRON Sysmac Studio projects. MCP is an open protocol that allows an AI application to discover and call well-defined tools exposed by another application or service. Read the [MCP introduction](https://modelcontextprotocol.io/introduction) and the [official specification](https://modelcontextprotocol.io/specification/latest) for the protocol concepts and message flow.

The server is written in Go. Its responsibilities are deliberately split into three layers:

- **MCP transport and lifecycle** handle client initialization, protocol-version negotiation, capabilities, and normal requests.
- **Sysmac integration** discovers projects and reads or writes supported project data through the OMRON tooling available on the host.
- **Tool definitions** expose focused operations to the AI client, such as project discovery, inspection, structured-text access, and build-related operations.

## Client lifecycle

An MCP client must initialize the connection before it can make normal tool calls. During initialization, the client and server negotiate a protocol version and exchange capabilities. Once the handshake is complete, the client can list tools and invoke them. OMRON MCP follows that lifecycle so an AI client can discover what the server supports instead of relying on hard-coded assumptions.

In practice, the AI client configuration only needs to point to the OMRON MCP executable (or its configured launcher). The client starts the process, performs the MCP handshake, and presents the discovered tools in the AI session.

## Project flow

1. The server discovers available Sysmac projects on the host.
2. The user selects the project that should be used for the session.
3. The AI client invokes narrowly scoped MCP tools.
4. The server validates the request, performs the corresponding Sysmac operation, and returns structured results or an actionable error.
5. The user reviews any generated or changed PLC content before applying it to a controller.

The server is not a replacement for Sysmac Studio or for commissioning procedures. It provides an AI-accessible interface around project inspection and supported engineering operations.

## Safety and engineering practice

PLC changes can affect machines and people. Use source control, project backups, simulation or offline validation, and your existing review and commissioning procedures. Do not treat an AI-generated result as automatically safe to deploy.

## Attribution

OMRON MCP is created and maintained by **Roelof Jan Boer**. See the [project repository](https://github.com/rjboer/OMRON-MCP) for the source and issue tracker.
