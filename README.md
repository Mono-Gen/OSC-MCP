# OSC Model Context Protocol (MCP) Server

`osc-mcp` is an MCP server that allows AI agents (such as Claude Desktop or Gemini Antigravity) to monitor and control OSC (Open Sound Control) compatible applications and devices (e.g., VRChat, TouchOSC, Max/MSP, or audio-lighting gear) over UDP network connections.

---

## Features

- **No Third-Party Dependencies**: High security, simple compilation, and robust performance by building exclusively on Go's standard library.
- **Flexible OSC Message Sending**: Send typed values (`int`, `float`, `string`, `bool`) to any target UDP IP and port with proper OSC 1.0 byte structures.
- **Dynamic Port Listening**: Open UDP sockets on designated ports dynamically to receive incoming parameters from external devices.
- **Message History Resource**: Expose recently received OSC messages on a specific port (up to 100 entries per port) to the AI client.
- **Thread Safety**: Concurrent socket management and cache manipulation protected by `sync.RWMutex`.
- **Protected Communication channel**: All logs and errors are outputted to `Stderr` to protect the stdio channel from JSON-RPC corruption.

---

## Requirements

- Go 1.21+ (if compiling from source)
- An network-reachable OSC application or device (e.g., VRChat running locally, TouchOSC on an iPad, etc.)

---

## Installation & Setup

### 1. Run/Compile the Server
To compile a binary for Windows:
```bash
$env:GOOS="windows"
$env:GOARCH="amd64"
go build -o osc-mcp.exe main.go
```

To compile a binary for macOS:
```bash
GOOS=darwin GOARCH=arm64 go build -o osc-mcp main.go
```

### 2. Claude Desktop Integration

Edit the Claude configuration file for your OS:

- **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`
- **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`

#### Windows

```json
{
  "mcpServers": {
    "osc-mcp": {
      "command": "cmd.exe",
      "args": [
        "/c",
        "C:\\path\\to\\osc-mcp.exe"
      ],
      "cwd": "C:\\path\\to\\config_directory"
    }
  }
}
```

#### macOS

```json
{
  "mcpServers": {
    "osc-mcp": {
      "command": "/path/to/osc-mcp",
      "args": [],
      "cwd": "/path/to/config_directory"
    }
  }
}
```

---

## Available Tools & Resources

The MCP server registers the following tools and resources to the AI agent:

### Tools
- `send_osc`: Sends an OSC message to a specified UDP target IP, port, and OSC address.
- `start_listen_osc`: Binds a local UDP port to start listening to incoming OSC messages.
- `stop_listen_osc`: Closes the socket listener for the given port and stops the receiver thread.

### Resources
- `osc://messages/{port}/recent`: Queries the cached OSC messages received on a specific port.

---

## Documentation

For detailed interface design, check:
- [English Manual](doc/manual_en.md)
- [Japanese Manual](doc/manual_ja.md)
- [English Specification](doc/spec_en.md)
- [OSC 1.0 Specification Notes](doc/osc_spec_notes_en.md)
