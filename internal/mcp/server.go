package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/internal/output"
	"github.com/pranshuparmar/witr/internal/pipeline"
	"github.com/pranshuparmar/witr/internal/target"
	"github.com/pranshuparmar/witr/pkg/model"
)

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Tool definition for MCP tools/list
type mcpTool struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema interface{} `json:"inputSchema"`
}

// RunServer runs a Model Context Protocol (MCP) server over stdin/stdout.
func RunServer(r io.Reader, w io.Writer, version string) error {
	scanner := bufio.NewScanner(r)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 4*1024*1024)

	tools := []mcpTool{
		{
			Name:        "witr_trace_port",
			Description: "Trace what process is listening on a network port and why it is running (causal tree back to supervisor/shell)",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"port": map[string]interface{}{
						"type":        "integer",
						"description": "Port number to trace (e.g. 3000, 8080)",
					},
				},
				"required": []string{"port"},
			},
		},
		{
			Name:        "witr_trace_pid",
			Description: "Trace a process by PID and explain why it exists, its supervisor ancestry, and open sockets",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"pid": map[string]interface{}{
						"type":        "integer",
						"description": "Process ID to trace",
					},
				},
				"required": []string{"pid"},
			},
		},
		{
			Name:        "witr_trace_file",
			Description: "Find which process is holding a specific file or lock open",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Absolute or relative file path (e.g. /var/lib/dpkg/lock)",
					},
				},
				"required": []string{"path"},
			},
		},
		{
			Name:        "witr_trace_name",
			Description: "Trace a process by executable or command name",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"name": map[string]interface{}{
						"type":        "string",
						"description": "Process name or substring",
					},
					"exact": map[string]interface{}{
						"type":        "boolean",
						"description": "Require exact match (no substring match)",
					},
				},
				"required": []string{"name"},
			},
		},
	}

	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		var req jsonRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			resp := jsonRPCResponse{
				JSONRPC: "2.0",
				Error:   &rpcError{Code: -32700, Message: "Parse error"},
			}
			send(w, resp)
			continue
		}

		switch req.Method {
		case "initialize":
			resp := jsonRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: map[string]interface{}{
					"protocolVersion": "2024-11-05",
					"capabilities": map[string]interface{}{
						"tools": map[string]interface{}{},
					},
					"serverInfo": map[string]interface{}{
						"name":    "witr",
						"version": version,
					},
				},
			}
			send(w, resp)

		case "notifications/initialized":
			// No response required for notifications

		case "tools/list":
			resp := jsonRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: map[string]interface{}{
					"tools": tools,
				},
			}
			send(w, resp)

		case "tools/call":
			var callParams struct {
				Name      string                 `json:"name"`
				Arguments map[string]interface{} `json:"arguments"`
			}
			if err := json.Unmarshal(req.Params, &callParams); err != nil {
				send(w, jsonRPCResponse{
					JSONRPC: "2.0",
					ID:      req.ID,
					Error:   &rpcError{Code: -32602, Message: "Invalid params"},
				})
				continue
			}

			outputStr, isErr := handleToolCall(callParams.Name, callParams.Arguments)
			resp := jsonRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result: map[string]interface{}{
					"content": []map[string]interface{}{
						{
							"type": "text",
							"text": outputStr,
						},
					},
					"isError": isErr,
				},
			}
			send(w, resp)

		case "ping":
			send(w, jsonRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  map[string]interface{}{},
			})

		default:
			send(w, jsonRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &rpcError{Code: -32601, Message: fmt.Sprintf("Method not found: %s", req.Method)},
			})
		}
	}

	return scanner.Err()
}

func send(w io.Writer, resp jsonRPCResponse) {
	data, _ := json.Marshal(resp)
	data = append(data, '\n')
	w.Write(data)
}

func handleToolCall(name string, args map[string]interface{}) (string, bool) {
	switch name {
	case "witr_trace_port":
		var port int
		if p, ok := args["port"].(float64); ok {
			port = int(p)
		} else if s, ok := args["port"].(string); ok {
			port, _ = strconv.Atoi(s)
		}
		if port <= 0 {
			return "Invalid port argument", true
		}
		t := model.Target{Type: model.TargetPort, Value: strconv.Itoa(port)}
		return executeTargetJSON(t, false)

	case "witr_trace_pid":
		var pid int
		if p, ok := args["pid"].(float64); ok {
			pid = int(p)
		} else if s, ok := args["pid"].(string); ok {
			pid, _ = strconv.Atoi(s)
		}
		if pid <= 0 {
			return "Invalid pid argument", true
		}
		t := model.Target{Type: model.TargetPID, Value: strconv.Itoa(pid)}
		return executeTargetJSON(t, false)

	case "witr_trace_file":
		path, _ := args["path"].(string)
		if strings.TrimSpace(path) == "" {
			return "Invalid file path argument", true
		}
		t := model.Target{Type: model.TargetFile, Value: path}
		return executeTargetJSON(t, false)

	case "witr_trace_name":
		pname, _ := args["name"].(string)
		if strings.TrimSpace(pname) == "" {
			return "Invalid process name argument", true
		}
		exact, _ := args["exact"].(bool)
		t := model.Target{Type: model.TargetName, Value: pname}
		return executeTargetJSON(t, exact)

	default:
		return fmt.Sprintf("Unknown tool: %s", name), true
	}
}

func executeTargetJSON(t model.Target, exact bool) (string, bool) {
	pids, err := target.Resolve(t, exact)
	if err != nil {
		return err.Error(), true
	}
	if len(pids) == 0 {
		return fmt.Sprintf("No process found for target: %s (%s)", t.Value, t.Type), true
	}

	// Analyze primary PID
	pid := pids[0]
	res, err := pipeline.AnalyzePID(pipeline.AnalyzeConfig{
		PID:    pid,
		Target: t,
	})
	if err != nil {
		return err.Error(), true
	}

	jsonStr, err := output.ToJSON(res)
	if err != nil {
		return err.Error(), true
	}
	return jsonStr, false
}
