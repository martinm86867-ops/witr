package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

// PrintDot generates a Graphviz DOT digraph representing the process ancestry chain.
func PrintDot(w io.Writer, chain []model.Process, children []model.Process) {
	if len(chain) == 0 {
		return
	}

	fmt.Fprintln(w, "digraph witr {")
	fmt.Fprintln(w, "  rankdir=TB;")
	fmt.Fprintln(w, "  node [shape=box, style=\"rounded,filled\", fillcolor=\"#f8f9fa\", fontname=\"Helvetica\", fontsize=10];")
	fmt.Fprintln(w, "  edge [color=\"#6c757d\", arrowsize=0.8];")

	escape := func(s string) string {
		s = strings.ReplaceAll(s, `\`, `\\`)
		s = strings.ReplaceAll(s, `"`, `\"`)
		return s
	}

	for i, proc := range chain {
		name := escape(ChainName(proc))
		nodeID := fmt.Sprintf("\"%d\"", proc.PID)
		var label string
		if len(proc.Sockets) > 0 {
			var ports []string
			for _, s := range proc.Sockets {
				if s.State == "LISTEN" && s.Port > 0 {
					ports = append(ports, fmt.Sprintf(":%d", s.Port))
				}
			}
			if len(ports) > 0 {
				label = fmt.Sprintf("%s\\n(pid %d) %s", name, proc.PID, strings.Join(ports, " "))
			} else {
				label = fmt.Sprintf("%s\\n(pid %d)", name, proc.PID)
			}
		} else {
			label = fmt.Sprintf("%s\\n(pid %d)", name, proc.PID)
		}

		// Highlight target leaf process in green
		if i == len(chain)-1 {
			fmt.Fprintf(w, "  %s [label=\"%s\", fillcolor=\"#d4edda\", color=\"#28a745\", penwidth=2.0];\n", nodeID, label)
		} else {
			fmt.Fprintf(w, "  %s [label=\"%s\"];\n", nodeID, label)
		}

		if i > 0 {
			prevID := fmt.Sprintf("\"%d\"", chain[i-1].PID)
			if gap := ParentGap(chain, i); gap != "" {
				gapID := fmt.Sprintf("\"gap%d\"", i)
				fmt.Fprintf(w, "  %s [label=\"? (%s)\", style=\"dashed,filled\", fillcolor=\"#fff3cd\", color=\"#ffc107\"];\n", gapID, escape(gap))
				fmt.Fprintf(w, "  %s -> %s;\n", prevID, gapID)
				fmt.Fprintf(w, "  %s -> %s;\n", gapID, nodeID)
			} else {
				fmt.Fprintf(w, "  %s -> %s;\n", prevID, nodeID)
			}
		}
	}

	targetNode := fmt.Sprintf("\"%d\"", chain[len(chain)-1].PID)
	limit := 5
	for i, child := range children {
		if i >= limit {
			fmt.Fprintf(w, "  \"more\" [label=\"... and %d more\", style=\"dotted,filled\", fillcolor=\"#e2e3e5\"];\n", len(children)-limit)
			fmt.Fprintf(w, "  %s -> \"more\";\n", targetNode)
			break
		}
		childID := fmt.Sprintf("\"c%d\"", child.PID)
		fmt.Fprintf(w, "  %s [label=\"%s\\n(pid %d)\"];\n", childID, escape(ChainName(child)), child.PID)
		fmt.Fprintf(w, "  %s -> %s;\n", targetNode, childID)
	}

	fmt.Fprintln(w, "}")
}
