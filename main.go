package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/signal"
	"syscall"

	"osc-mcp/mcp"
)

func main() {
	// Redirect default log output to os.Stderr to protect os.Stdout for MCP communication.
	log.SetOutput(os.Stderr)
	log.SetPrefix("[osc-mcp-main] ")

	server := mcp.NewServer()

	// Signal handling for clean shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received termination signal, shutting down gracefully...")
		server.CloseAll()
		os.Exit(0)
	}()

	reader := bufio.NewReader(os.Stdin)
	log.Println("OSC-MCP Server is running. Listening on stdin...")

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				log.Println("Stdin reached EOF. Exiting...")
				break
			}
			log.Printf("Error reading stdin: %v\n", err)
			break
		}

		// Skip empty lines
		if len(line) <= 1 {
			continue
		}

		var req mcp.Request
		if err := json.Unmarshal(line, &req); err != nil {
			log.Printf("Failed to unmarshal JSON-RPC request line: %v\n", err)
			sendErrorResponse(os.Stdout, nil, -32700, "Parse error")
			continue
		}

		server.HandleRequest(os.Stdout, req)
	}

	server.CloseAll()
	log.Println("OSC-MCP Server stopped.")
}

func sendErrorResponse(w io.Writer, id any, code int, message string) {
	res := mcp.Response{
		JSONRPC: "2.0",
		Error: &mcp.RPCError{
			Code:    code,
			Message: message,
		},
		ID: id,
	}
	if bytes, err := json.Marshal(res); err == nil {
		fmt.Fprintln(w, string(bytes))
	}
}
