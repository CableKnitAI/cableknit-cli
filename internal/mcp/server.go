package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
)

type Server struct {
	version string
	cache   *Cache
	writer  io.Writer
	mu      sync.Mutex
}

func NewServer(version string) *Server {
	return &Server{
		version: version,
		cache:   NewCache(),
	}
}

func (s *Server) Run(in io.Reader, out io.Writer) error {
	s.writer = out

	// Load manifest data (API → disk cache → embedded fallback)
	s.cache.Load()

	scanner := bufio.NewScanner(in)
	scanner.Buffer(make([]byte, 0, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req Request
		if err := json.Unmarshal(line, &req); err != nil {
			s.writeError(nil, -32700, "Parse error")
			continue
		}

		s.handle(&req)
	}

	return scanner.Err()
}

func (s *Server) handle(req *Request) {
	switch req.Method {

	// Lifecycle
	case "initialize":
		s.respond(req.ID, InitializeResult{
			ProtocolVersion: "2024-11-05",
			Capabilities: Capabilities{
				Tools:     &ToolsCapability{},
				Resources: &ResourcesCapability{},
				Prompts:   &PromptsCapability{},
			},
			ServerInfo: ServerInfo{
				Name:    "cableknit",
				Version: s.version,
			},
		})

	case "notifications/initialized":
		// No response for notifications

	case "ping":
		s.respond(req.ID, map[string]any{})

	// Tools
	case "tools/list":
		s.respond(req.ID, ToolsListResult{Tools: allTools()})

	case "tools/call":
		var params ToolCallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.writeError(req.ID, -32602, "Invalid params")
			return
		}
		result := s.callTool(params)
		s.respond(req.ID, result)

	// Resources
	case "resources/list":
		s.respond(req.ID, ResourcesListResult{Resources: allResources()})

	case "resources/read":
		var params ResourceReadParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.writeError(req.ID, -32602, "Invalid params")
			return
		}
		result := s.readResource(params)
		s.respond(req.ID, result)

	// Prompts
	case "prompts/list":
		s.respond(req.ID, PromptsListResult{Prompts: allPrompts()})

	case "prompts/get":
		var params PromptGetParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			s.writeError(req.ID, -32602, "Invalid params")
			return
		}
		result, err := s.getPrompt(params)
		if err != nil {
			s.writeError(req.ID, -32602, err.Error())
			return
		}
		s.respond(req.ID, result)

	default:
		if !req.IsNotification() {
			s.writeError(req.ID, -32601, fmt.Sprintf("Method not found: %s", req.Method))
		}
	}
}

func (s *Server) respond(id json.RawMessage, result any) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	s.write(resp)
}

func (s *Server) writeError(id json.RawMessage, code int, msg string) {
	resp := Response{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &RPCError{Code: code, Message: msg},
	}
	s.write(resp)
}

func (s *Server) write(resp Response) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.Marshal(resp)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mcp: marshal error: %v\n", err)
		return
	}
	data = append(data, '\n')
	s.writer.Write(data)
}
