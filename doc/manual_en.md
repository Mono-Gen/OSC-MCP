# OSC MCP Server Setup & Feature Manual

This manual provides instructions to setup, run, and integrate the OSC MCP Server (`osc-mcp`) with AI assistants (such as Claude Desktop or Gemini Antigravity), helping you control OSC-compatible software and hardware applications over UDP.

---

## 🔰 Getting Started: Core Concepts & Terminology

Before you start developing, let's clear up some key terminology:

*   **OSC (Open Sound Control)**: A protocol for communication among computers, sound synthesizers, and other multimedia devices. It is widely used in applications like VRChat (for avatar parameters), TouchOSC, and Max/MSP.
*   **UDP (User Datagram Protocol)**: A lightweight network communication protocol used to send data packets quickly. OSC packets are typically sent over UDP.
*   **MCP (Model Context Protocol)**: A protocol enabling AI models (like Claude) to securely interface with local servers and external hardware control layers.
*   **OSC Address**: A URL-like path used to identify a specific control target. It must start with a slash (e.g., `/avatar/parameters/Mute`).
*   **Type Tag**: A header string starting with a comma that details the types of arguments inside the packet (e.g., `,ifs` indicates an integer, a float, and a string argument).
*   **Bundle**: A package that groups multiple OSC messages with a 64-bit NTP time tag.

---

## Setup & Execution

### 1. Build Executable
Compile the source code using the Go compiler:

- **Windows (PowerShell)**:
  ```powershell
  $env:GOOS="windows"
  $env:GOARCH="amd64"
  go build -o osc-mcp.exe main.go
  ```

- **macOS (Apple Silicon)**:
  ```bash
  GOOS=darwin GOARCH=arm64 go build -o osc-mcp main.go
  ```

---

### 2. Integration with AI Assistants (MCP Clients)

Integrate this server with your preferred AI client.

#### Config File Paths
- **Claude Desktop**
  - **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`
  - **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Antigravity / Gemini MCP Clients**
  - **Universal**: `~/.gemini/antigravity-ide/mcp_settings.json` etc.

#### Configuration Snippets

##### ■ Windows Configuration
On Windows, launching the executable via `cmd.exe /c` ensures stable channel streams:

```json
{
  "mcpServers": {
    "osc-mcp": {
      "command": "cmd.exe",
      "args": [
        "/c",
        "C:\\path\\to\\osc-mcp.exe"
      ],
      "cwd": "C:\\path\\to\\project_directory"
    }
  }
}
```
*Note: Make sure to escape backslashes with double backslashes (`\\`).*

##### ■ macOS Configuration
```json
{
  "mcpServers": {
    "osc-mcp": {
      "command": "/path/to/osc-mcp",
      "args": [],
      "cwd": "/path/to/project_directory"
    }
  }
}
```

---

## Available Tools & Resources

The MCP server registers the following interfaces:

### 1. Tools

*   **`send_osc`**
    - **Description**: Sends a single OSC message to a designated IP, port, and address.
    - **Arguments**:
      - `host` (string, optional): Target IP address (default: `127.0.0.1`).
      - `port` (integer, required): Target UDP port (e.g., `9000` for VRChat).
      - `address` (string, required): OSC address pattern starting with `/`.
      - `args` (array, optional): List of typed arguments containing `type` (`"int"`, `"float"`, `"string"`, or `"bool"`) and `value`.

*   **`start_listen_osc`**
    - **Description**: Binds a local UDP port to start listening to incoming OSC messages.
    - **Arguments**:
      - `port` (integer, required): Local port to open.
      - `host` (string, optional): Local IP to bind (default: `127.0.0.1`. Use `0.0.0.0` to receive from external network devices).
    - **Behavior**: Spawns a reader loop in the background. Retains the connection if called multiple times on the same port. Stores the last 100 messages in a ring buffer.

*   **`stop_listen_osc`**
    - **Description**: Closes the socket listener for the given port and stops the receiver thread.
    - **Arguments**:
      - `port` (integer, required): Port to release.

### 2. Resources

*   **`osc://messages/{port}/recent`**
    - **Description**: Requests the recently cached OSC messages for the given port.
    - **Returns**: A JSON string listing recent messages with timestamps, sender IP/ports, and arguments.

---

## Troubleshooting

### Port Conflicts
If `start_listen_osc` fails with a port bind conflict, ensure the port is not used by another application.
On Windows:
```cmd
netstat -ano | findstr <port>
```

### JSON-RPC Stream Corruption
Do **not** print standard logs to `Stdout`. Any unexpected characters sent to standard output will corrupt the JSON-RPC stream and cause the MCP client to crash. Print all debugging logs to `Stderr`.
