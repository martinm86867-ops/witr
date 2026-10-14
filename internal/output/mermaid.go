package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

// PrintMermaid generates a Mermaid.js flowchart representing the process ancestry chain.
func PrintMermaid(w io.Writer, chain []model.Process, children []model.Process) {
	if len(chain) == 0 {
		return
	}

	fmt.Fprintln(w, "```mermaid")
	fmt.Fprintln(w, "graph TD")

	// Escape strings for Mermaid node labels
	escape := func(s string) string {
		s = strings.ReplaceAll(s, `"`, `'`)
		s = strings.ReplaceAll(s, `[`, `\\[`)
		s = strings.ReplaceAll(s, `]`, `\\]`)
		s = strings.ReplaceAll(s, `(`, `\\(`)
		s = strings.ReplaceAll(s, `)`, `\\)`)
		return s
	}

	for i, proc := range chain {
		name := escape(ChainName(proc))
		nodeID := fmt.Sprintf("p%d", proc.PID)
		var label string
		if len(proc.Sockets) > 0 {
			var ports []string
			for _, s := range proc.Sockets {
				if s.State == "LISTEN" && s.Port > 0 {
					ports = append(ports, fmt.Sprintf(":%d", s.Port))
				}
			}
			if len(ports) > 0 {
				label = fmt.Sprintf("%s (pid %d) %s", name, proc.PID, strings.Join(ports, " "))
			} else {
				label = fmt.Sprintf("%s (pid %d)", name, proc.PID)
			}
		} else {
			label = fmt.Sprintf("%s (pid %d)", name, proc.PID)
		}

		fmt.Fprintf(w, "  %s[\"%s\"]\n", nodeID, label)

		if i > 0 {
			prevID := fmt.Sprintf("p%d", chain[i-1].PID)
			if gap := ParentGap(chain, i); gap != "" {
				gapID := fmt.Sprintf("gap%d", i)
				fmt.Fprintf(w, "  %s[\"? (%s)\"]\n", gapID, escape(gap))
				fmt.Fprintf(w, "  %s --> %s\n", prevID, gapID)
				fmt.Fprintf(w, "  %s --> %s\n", gapID, nodeID)
			} else {
				fmt.Fprintf(w, "  %s --> %s\n", prevID, nodeID)
			}
		}
	}

	// Highlight target leaf process
	targetNode := fmt.Sprintf("p%d", chain[len(chain)-1].PID)
	fmt.Fprintf(w, "  style %s fill:#2ecc71,stroke:#27ae60,stroke-width:2px,color:#ffffff\n", targetNode)

	// Add immediate children if any
	limit := 5
	for i, child := range children {
		if i >= limit {
			fmt.Fprintf(w, "  %s --> more[\"... and %d more\"]\n", targetNode, len(children)-limit)
			break
		}
		childID := fmt.Sprintf("c%d", child.PID)
		fmt.Fprintf(w, "  %s[\"%s (pid %d)\"]\n", childID, escape(ChainName(child)), child.PID)
		fmt.Fprintf(w, "  %s --> %s\n", targetNode, childID)
	}

	fmt.Fprintln(w, "```")
}
