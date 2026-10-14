package output

import (
	"bytes"
	"strings"
	"testing"

	"github.com/pranshuparmar/witr/pkg/model"
)

func TestPrintMermaid(t *testing.T) {
	chain := []model.Process{
		{PID: 1, Command: "systemd"},
		{PID: 100, Command: "bash"},
		{PID: 200, Command: "node", Sockets: []model.Socket{{State: "LISTEN", Port: 3000}}},
	}

	var buf bytes.Buffer
	PrintMermaid(&buf, chain, nil)
	out := buf.String()

	if !strings.Contains(out, "```mermaid") {
		t.Errorf("expected ```mermaid header, got: %s", out)
	}
	if !strings.Contains(out, "p1[\"systemd (pid 1)\"]") {
		t.Errorf("expected systemd node, got: %s", out)
	}
	if !strings.Contains(out, "p1 --> p100") {
		t.Errorf("expected p1 --> p100, got: %s", out)
	}
	if !strings.Contains(out, ":3000") {
		t.Errorf("expected :3000 in node label, got: %s", out)
	}
}

func TestPrintDot(t *testing.T) {
	chain := []model.Process{
		{PID: 1, Command: "systemd"},
		{PID: 200, Command: "node"},
	}

	var buf bytes.Buffer
	PrintDot(&buf, chain, nil)
	out := buf.String()

	if !strings.Contains(out, "digraph witr {") {
		t.Errorf("expected digraph witr, got: %s", out)
	}
	if !strings.Contains(out, "\"1\" -> \"200\";") {
		t.Errorf("expected 1 -> 200 edge, got: %s", out)
	}
}
