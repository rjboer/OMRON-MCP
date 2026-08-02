# OMRON MCP

Give your AI assistant a practical connection to an OMRON Sysmac Studio project.

OMRON MCP is a Go-based [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) service that lets an AI tool work with OMRON PLC projects. It can help you create and inspect PLC code, understand project structure, analyse data, and investigate faults without losing sight of the real Sysmac project on your machine.

The goal is simple: bring an AI assistant into the engineering workflow as a useful, reviewable pair of hands for OMRON PLC development.

## Get started

1. Create or open a project in **OMRON Sysmac Studio**.
2. Start OMRON MCP using the instructions for your platform and keep it available to your AI client.
3. Open an AI tool that supports MCP, such as Codex, and add/configure OMRON MCP as an MCP server. The client starts the MCP initialization handshake and then discovers the available tools.
4. Select the Sysmac project you just created when the tool asks you to choose a project.
5. Work with the project through natural language: generate or inspect PLC code, trace project data, debug a fault, and analyse the evidence returned by Sysmac Studio.

Treat generated changes as engineering changes: review them, test them in a safe environment, and download to a controller only through your normal approval process.

## What it is useful for

- Creating and refining OMRON PLC programs with AI assistance.
- Exploring Sysmac project structure and implementation details.
- Debugging faults and analysing variables, containers, and project data.
- Asking focused questions about an existing project instead of searching through files manually.

For the protocol and implementation details, see [Technical overview](TECHNICAL.md).

## Status

This project is in testing and active development. If something is unclear or does not work as expected, please [open an issue](https://github.com/rjboer/OMRON-MCP/issues).

## About

Created and maintained by **Roelof Jan Boer**.

Connect with Roelof Jan Boer on [LinkedIn](https://www.linkedin.com/in/rjboer/).
