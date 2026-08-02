# OMRON MCP

## AI-assisted engineering for OMRON Sysmac Studio

**Give your AI assistant a controlled, practical connection to real OMRON PLC projects.**

OMRON MCP is a Go-based [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) server and Windows workbench for OMRON Sysmac Studio projects. It allows compatible AI tools to inspect project structure, understand PLC code, analyse variables, investigate faults, generate Structured Text, and propose or apply reviewed changes directly to the project on your machine.

The objective is to make AI a capable engineering partner rather than a separate chatbot. Instead of working from isolated code fragments, the AI can operate with the context of the actual Sysmac project: its programs, entities, variables, metadata, and build results.

> [!IMPORTANT]
> OMRON MCP requires a valid OMRON Sysmac Studio license.

**Like my work?** [Add Roelof Jan Boer on LinkedIn](https://www.linkedin.com/in/rjboer/).

<p align="center">
  <img
    width="1105"
    height="750"
    alt="OMRON Sysmac MCP Workbench"
    src="https://github.com/user-attachments/assets/0148391e-b4d5-4ad3-8968-1c60e8c20645"
  />
</p>

---

## Get started

1. Create or open a project in **OMRON Sysmac Studio**.
2. Start OMRON MCP using the instructions for your platform and make it available to your AI client. OMRON MCP has been tested with Codex and Antigravity.
3. Open an AI tool that supports MCP, such as Codex, Cursor, or Antigravity, and add or configure OMRON MCP as an MCP server. The client starts the MCP initialization handshake and then discovers the available tools.
4. Select the Sysmac project you just created when the tool asks you to choose a project.
5. Work with the project through natural language: generate or inspect PLC code, trace project data, debug a fault, and analyse the evidence returned by Sysmac Studio.
6. Review the proposed code and project changes in your editor, such as Codex or Antigravity.
7. Apply the approved changes in your editor.
8. Validate the result in Sysmac Studio.

> [!WARNING]
> Treat generated changes as engineering changes: review them, test them in a safe environment, and download to a controller only through your normal approval process.

---

## What it is useful for

- Creating and refining OMRON PLC programs with AI assistance.
- Exploring Sysmac project structure and implementation details.
- Debugging faults and analysing variables, containers, and project data.
- Asking focused questions about an existing project instead of searching through files manually.

For protocol and implementation details, see the [technical overview](TECHNICAL.md).

OMRON MCP brings AI into the existing automation-engineering workflow while keeping the real project, engineering decisions, and final approval under the control of the PLC engineer.

> [!CAUTION]
> **Project status:** OMRON MCP is under active development. Use project copies or version-controlled snapshots when testing write operations.

This project is in testing and active development. If something is unclear or does not work as expected, please [open an issue](https://github.com/rjboer/OMRON-MCP/issues).

> OMRON MCP is an independent open-source project and is not affiliated with or endorsed by OMRON Corporation.

---

## Development flow

The repository uses two permanent branches:

- **`develop`** is the integration branch. Commit working changes here; GitHub Actions runs the full test suite and Windows build checks on every push.
- **`main`** is the release branch. Promote the tested `develop` state to `main`; a push to `main` runs the release workflow, signs the Windows artifact, and publishes the continuous GitHub release.

Feature branches are optional and should be merged back into `develop` before a release is promoted to `main`.

---

## About

Created and maintained by **Roelof Jan Boer**.

Like my work? [Add Roelof Jan Boer on LinkedIn](https://www.linkedin.com/in/rjboer/) and join the conversation.

> OMRON MCP is an independent open-source project and is not affiliated with or endorsed by OMRON Corporation.
