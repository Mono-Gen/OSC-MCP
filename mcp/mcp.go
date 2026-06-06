package mcp

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"regexp"
	"strconv"
	"sync"
	"time"

	"osc-mcp/osc"
)

// JSON-RPC 2.0 Types
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      any             `json:"id,omitempty"`
}

type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
	ID      any             `json:"id"`
}

type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

// MCP Protocol Types
type Tool struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	InputSchema any    `json:"inputSchema"`
}

type ToolListResult struct {
	Tools []Tool `json:"tools"`
}

type CallToolParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

type CallToolResult struct {
	Content []TextContent `json:"content"`
	IsError bool          `json:"isError,omitempty"`
}

type TextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type Resource struct {
	URI         string `json:"uri"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	MimeType    string `json:"mimeType,omitempty"`
}

type ResourceListResult struct {
	Resources []Resource `json:"resources"`
}

type ReadResourceParams struct {
	URI string `json:"uri"`
}

type ResourceContent struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text"`
}

type ReadResourceResult struct {
	Contents []ResourceContent `json:"contents"`
}

// Server Core State
type CacheMessage struct {
	Timestamp string         `json:"timestamp"`
	From      string         `json:"from"`
	Address   string         `json:"address"`
	Args      []osc.Argument `json:"args,omitempty"`
}

type Server struct {
	mu        sync.RWMutex
	listeners map[int]*net.UDPConn
	caches    map[int][]CacheMessage
	bufPool   sync.Pool
	logger    *log.Logger
}

const (
	MaxCacheSize = 100
	BufferSize   = 65535
)

// NewServer initializes and returns a new Server instance.
func NewServer() *Server {
	return &Server{
		listeners: make(map[int]*net.UDPConn),
		caches:    make(map[int][]CacheMessage),
		bufPool: sync.Pool{
			New: func() any {
				return make([]byte, BufferSize)
			},
		},
		logger: log.New(os.Stderr, "[mcp-server] ", log.LstdFlags),
	}
}

// StartListen opens a UDP port to start listening to incoming OSC messages.
func (s *Server) StartListen(host string, port int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.listeners[port]; ok {
		s.logger.Printf("Already listening on port %d, maintaining status", port)
		return nil
	}

	addrStr := fmt.Sprintf("%s:%d", host, port)
	udpAddr, err := net.ResolveUDPAddr("udp", addrStr)
	if err != nil {
		return fmt.Errorf("failed to resolve local address %s: %w", addrStr, err)
	}

	conn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		return fmt.Errorf("port conflict or permission issue on port %d: %w (ensure the port is not used by another application)", port, err)
	}

	s.listeners[port] = conn
	s.caches[port] = make([]CacheMessage, 0, MaxCacheSize)

	go s.listenLoop(port, conn)

	s.logger.Printf("Started listening on %s", addrStr)
	return nil
}

// StopListen closes the UDP listener socket for the given port.
func (s *Server) StopListen(port int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	conn, ok := s.listeners[port]
	if !ok {
		return fmt.Errorf("no listener found on port %d", port)
	}

	err := conn.Close()
	delete(s.listeners, port)
	delete(s.caches, port)

	s.logger.Printf("Stopped listening on port %d", port)
	return err
}

// CloseAll closes all open UDP sockets.
func (s *Server) CloseAll() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for port, conn := range s.listeners {
		conn.Close()
		delete(s.listeners, port)
		delete(s.caches, port)
	}
	s.logger.Println("Closed all active listeners")
}

func (s *Server) listenLoop(port int, conn *net.UDPConn) {
	for {
		buf := s.bufPool.Get().([]byte)
		n, remoteAddr, err := conn.ReadFromUDP(buf)
		if err != nil {
			s.bufPool.Put(buf)
			s.logger.Printf("Listener on port %d closed: %v", port, err)
			return
		}

		packetData := make([]byte, n)
		copy(packetData, buf[:n])
		s.bufPool.Put(buf)

		messages, err := osc.DecodePacket(packetData)
		if err != nil {
			s.logger.Printf("Failed to decode OSC packet on port %d: %v", port, err)
			continue
		}

		s.mu.Lock()
		cache, ok := s.caches[port]
		if ok {
			now := time.Now().Format(time.RFC3339)
			fromStr := remoteAddr.String()

			for _, msg := range messages {
				cm := CacheMessage{
					Timestamp: now,
					From:      fromStr,
					Address:   msg.Address,
					Args:      msg.Args,
				}

				if len(cache) >= MaxCacheSize {
					cache = cache[1:]
				}
				cache = append(cache, cm)
			}
			s.caches[port] = cache
		}
		s.mu.Unlock()
	}
}

// HandleRequest processes an incoming JSON-RPC Request and writes the Response.
func (s *Server) HandleRequest(w io.Writer, req Request) {
	var res Response
	res.JSONRPC = "2.0"
	res.ID = req.ID

	result, err := s.dispatch(req)
	if err != nil {
		res.Error = err
	} else {
		res.Result = result
	}

	jsonBytes, errMarshal := json.Marshal(res)
	if errMarshal != nil {
		s.logger.Printf("Failed to marshal JSON-RPC response: %v", errMarshal)
		return
	}

	fmt.Fprintln(w, string(jsonBytes))
}

func (s *Server) dispatch(req Request) (json.RawMessage, *RPCError) {
	switch req.Method {
	case "initialize":
		initResult := map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities": map[string]any{
				"tools":     map[string]any{},
				"resources": map[string]any{},
			},
			"serverInfo": map[string]string{
				"name":    "osc-mcp-server",
				"version": "1.0.0",
			},
		}
		b, _ := json.Marshal(initResult)
		return b, nil

	case "tools/list":
		toolList := ToolListResult{
			Tools: []Tool{
				{
					Name:        "send_osc",
					Description: "指定された宛先（IP・ポート・アドレス）に対して、型指定された引数を伴うOSCメッセージを送信します。",
					InputSchema: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"host": map[string]any{
								"type":        "string",
								"description": "宛先IPアドレス。デフォルトは '127.0.0.1'。",
							},
							"port": map[string]any{
								"type":        "integer",
								"description": "宛先ポート番号。",
							},
							"address": map[string]any{
								"type":        "string",
								"description": "OSCアドレス。例: '/avatar/parameters/Mute'。必ず '/' で始まる必要があります。",
							},
							"args": map[string]any{
								"type":        "array",
								"description": "送信する引数のリスト。",
								"items": map[string]any{
									"type": "object",
									"properties": map[string]any{
										"type": map[string]any{
											"type": "string",
											"enum": []string{"int", "float", "string", "bool"},
										},
										"value": map[string]any{
											"type": "any",
										},
									},
									"required": []string{"type", "value"},
								},
							},
						},
						"required": []string{"port", "address"},
					},
				},
				{
					Name:        "start_listen_osc",
					Description: "指定されたポートとホストでOSCメッセージの待ち受けを開始します。",
					InputSchema: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"port": map[string]any{
								"type":        "integer",
								"description": "待ち受けを行うローカルポート番号。",
							},
							"host": map[string]any{
								"type":        "string",
								"description": "待ち受けを行うローカルIPアドレス。デフォルトは '127.0.0.1'。",
							},
						},
						"required": []string{"port"},
					},
				},
				{
					Name:        "stop_listen_osc",
					Description: "指定されたポートの待ち受けを終了し、UDPソケットを解放します。",
					InputSchema: map[string]any{
						"type": "object",
						"properties": map[string]any{
							"port": map[string]any{
								"type":        "integer",
								"description": "停止するポート番号。",
							},
						},
						"required": []string{"port"},
					},
				},
			},
		}
		b, _ := json.Marshal(toolList)
		return b, nil

	case "tools/call":
		var params CallToolParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, &RPCError{Code: -32602, Message: "Invalid parameters"}
		}
		return s.handleCallTool(params)

	case "resources/list":
		s.mu.RLock()
		defer s.mu.RUnlock()

		resources := make([]Resource, 0, len(s.listeners))
		for port := range s.listeners {
			resources = append(resources, Resource{
				URI:         fmt.Sprintf("osc://messages/%d/recent", port),
				Name:        fmt.Sprintf("OSC Port %d History", port),
				Description: fmt.Sprintf("Recent OSC messages received on port %d", port),
				MimeType:    "application/json",
			})
		}
		resList := ResourceListResult{Resources: resources}
		b, _ := json.Marshal(resList)
		return b, nil

	case "resources/read":
		var params ReadResourceParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return nil, &RPCError{Code: -32602, Message: "Invalid parameters"}
		}
		return s.handleReadResource(params)

	default:
		return nil, &RPCError{Code: -32601, Message: fmt.Sprintf("Method not found: %s", req.Method)}
	}
}

func (s *Server) handleCallTool(params CallToolParams) (json.RawMessage, *RPCError) {
	switch params.Name {
	case "send_osc":
		var args struct {
			Host    string         `json:"host"`
			Port    int            `json:"port"`
			Address string         `json:"address"`
			Args    []osc.Argument `json:"args"`
		}
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			return nil, &RPCError{Code: -32602, Message: fmt.Sprintf("Invalid arguments for send_osc: %v", err)}
		}

		if args.Host == "" {
			args.Host = "127.0.0.1"
		}

		msg := osc.Message{
			Address: args.Address,
			Args:    args.Args,
		}

		if err := osc.SendOSC(args.Host, args.Port, msg); err != nil {
			res := CallToolResult{
				Content: []TextContent{{Type: "text", Text: fmt.Sprintf("Failed to send OSC message: %v", err)}},
				IsError: true,
			}
			b, _ := json.Marshal(res)
			return b, nil
		}

		res := CallToolResult{
			Content: []TextContent{{Type: "text", Text: "OSC message sent successfully"}},
		}
		b, _ := json.Marshal(res)
		return b, nil

	case "start_listen_osc":
		var args struct {
			Port int    `json:"port"`
			Host string `json:"host"`
		}
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			return nil, &RPCError{Code: -32602, Message: fmt.Sprintf("Invalid arguments for start_listen_osc: %v", err)}
		}

		if args.Host == "" {
			args.Host = "127.0.0.1"
		}

		if err := s.StartListen(args.Host, args.Port); err != nil {
			res := CallToolResult{
				Content: []TextContent{{Type: "text", Text: fmt.Sprintf("Failed to start listening: %v", err)}},
				IsError: true,
			}
			b, _ := json.Marshal(res)
			return b, nil
		}

		res := CallToolResult{
			Content: []TextContent{{Type: "text", Text: fmt.Sprintf("Started listening on %s:%d", args.Host, args.Port)}},
		}
		b, _ := json.Marshal(res)
		return b, nil

	case "stop_listen_osc":
		var args struct {
			Port int `json:"port"`
		}
		if err := json.Unmarshal(params.Arguments, &args); err != nil {
			return nil, &RPCError{Code: -32602, Message: fmt.Sprintf("Invalid arguments for stop_listen_osc: %v", err)}
		}

		if err := s.StopListen(args.Port); err != nil {
			res := CallToolResult{
				Content: []TextContent{{Type: "text", Text: fmt.Sprintf("Failed to stop listening: %v", err)}},
				IsError: true,
			}
			b, _ := json.Marshal(res)
			return b, nil
		}

		res := CallToolResult{
			Content: []TextContent{{Type: "text", Text: fmt.Sprintf("Stopped listening on port %d", args.Port)}},
		}
		b, _ := json.Marshal(res)
		return b, nil

	default:
		return nil, &RPCError{Code: -32601, Message: fmt.Sprintf("Tool not found: %s", params.Name)}
	}
}

var uriRegex = regexp.MustCompile(`^osc://messages/(\d+)/recent$`)

func (s *Server) handleReadResource(params ReadResourceParams) (json.RawMessage, *RPCError) {
	matches := uriRegex.FindStringSubmatch(params.URI)
	if len(matches) != 2 {
		return nil, &RPCError{Code: -32602, Message: fmt.Sprintf("Invalid resource URI: %s", params.URI)}
	}

	port, err := strconv.Atoi(matches[1])
	if err != nil {
		return nil, &RPCError{Code: -32602, Message: fmt.Sprintf("Invalid port in URI: %s", matches[1])}
	}

	s.mu.RLock()
	cache, ok := s.caches[port]
	s.mu.RUnlock()

	if !ok {
		return nil, &RPCError{Code: 404, Message: fmt.Sprintf("No active listener or history found for port %d", port)}
	}

	history := map[string]any{
		"port":        port,
		"retrievedAt": time.Now().Format(time.RFC3339),
		"messages":    cache,
	}

	historyBytes, err := json.Marshal(history)
	if err != nil {
		return nil, &RPCError{Code: -32603, Message: "Failed to serialize history"}
	}

	res := ReadResourceResult{
		Contents: []ResourceContent{
			{
				URI:      params.URI,
				MimeType: "application/json",
				Text:     string(historyBytes),
			},
		},
	}
	b, _ := json.Marshal(res)
	return b, nil
}
