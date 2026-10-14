// Package witr carries the canonical witr agent skill as an embedded asset so
// the binary can hand it to any agent harness on demand.
//
// The Markdown is embedded from the same SKILL.md the Claude Code plugin ships,
// so there is exactly one copy to keep current.
package witr

import _ "embed"

// Markdown is the witr agent skill, served by `witr --skill`.
//
//go:embed SKILL.md
var Markdown string
