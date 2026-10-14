package mcp

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestMCPInitialize(t *testing.T) {
	req := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05"}}` + "\n"
	in := strings.NewReader(req)
	var out bytes.Buffer

	err := RunServer(in, &out, "v0.3.4-test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resp jsonRPCResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}

	if resp.JSONRPC != "2.0" {
		t.Errorf("expected jsonrpc 2.0, got %s", resp.JSONRPC)
	}
	resMap, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("expected result map, got %T", resp.Result)
	}
	if resMap["protocolVersion"] != "2024-11-05" {
		t.Errorf("expected protocolVersion 2024-11-05, got %v", resMap["protocolVersion"])
	}
}

func TestMCPToolsList(t *testing.T) {
	req := `{"jsonrpc":"2.0","id":2,"method":"tools/list"}` + "\n"
	in := strings.NewReader(req)
	var out bytes.Buffer

	err := RunServer(in, &out, "v0.3.4-test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resp jsonRPCResponse
	if err := json.Unmarshal(out.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse json response: %v", err)
	}

	resMap := resp.Result.(map[string]interface{})
	toolsList, ok := resMap["tools"].([]interface{})
	if !ok || len(toolsList) < 4 {
		t.Errorf("expected at least 4 tools, got %d", len(toolsList))
	}
}
