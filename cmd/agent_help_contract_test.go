package cmd

import (
	"strings"
	"testing"
)

// This file locks the parts of the --agent-help system that AI agents
// actually consume at runtime, and that no single command's test can
// protect on its own:
//
//  1. The markdown → JSON parser (agentHelpToMap). Agents that run
//     `gitee <cmd> --agent-help -o json` depend on its exact shape:
//     a top-level "command" string plus a "sections" object whose keys
//     are normalized section titles. A regression here silently breaks
//     every downstream agent workflow.
//  2. The root coverage matrix (rootAgentHelp). It is hand-maintained
//     text; without a test it rots the moment someone adds a command
//     and forgets the matrix row.
//
// Why not "execute every command with --agent-help and assert the
// sections"? Because PrintAgentHelp calls os.Exit(0) by design (so that
// `gitee foo --agent-help` short-circuits before the command runs), which
// makes in-process execution-based coverage impractical. Per-command
// section presence is instead covered by each package's own *content_test.go
// (e.g. cmd/alias/content_test.go); this file guards the shared machine
// contract those tests cannot see.

// standardAgentDimensions are the cross-cutting concerns the root help
// promises that --agent-help adds on top of --help. If the root doc stops
// advertising one of these, the "two surfaces" contract has drifted.
var standardAgentDimensions = []string{
	"Pre-conditions",
	"Post-conditions",
	"Idempotency",
	"Atomicity",
	"Side effects",
	"error",       // "Common error codes + how to recover"
	"OAuth scope", // required scope
	"Rate limit",
	"Race conditions",
	"orchestration", // "Next-step orchestration hints"
	"Parsing hints",
}

// TestRootAgentHelpAdvertisesAllDimensions guards the headline promise of
// the dual-surface design: --agent-help is --help PLUS these dimensions.
func TestRootAgentHelpAdvertisesAllDimensions(t *testing.T) {
	for _, dim := range standardAgentDimensions {
		if !strings.Contains(rootAgentHelp, dim) {
			t.Errorf("rootAgentHelp no longer advertises the %q dimension — the dual-surface contract has drifted", dim)
		}
	}
}

// TestAgentHelpToMapShape locks the JSON object shape agents parse.
func TestAgentHelpToMapShape(t *testing.T) {
	const sample = `# gitee demo create — Agent Help

## Purpose
Create a demo resource.

## Pre-conditions
- You are authenticated.
- The repo exists.

## Idempotency
Not idempotent: re-running creates a second resource.
`
	out := agentHelpToMap(sample)

	// command name: heading text with the "— Agent Help" suffix stripped.
	if got, want := out["command"], "gitee demo create"; got != want {
		t.Errorf("command = %q, want %q", got, want)
	}

	sections, ok := out["sections"].(map[string]any)
	if !ok {
		t.Fatalf("sections is %T, want map[string]any", out["sections"])
	}

	// List body → []string with backticks stripped and "- " removed.
	pre, ok := sections["pre_conditions"].([]string)
	if !ok {
		t.Fatalf("pre_conditions is %T, want []string", sections["pre_conditions"])
	}
	if len(pre) != 2 || pre[0] != "You are authenticated." {
		t.Errorf("pre_conditions = %#v, want 2 items starting with the auth precondition", pre)
	}

	// Paragraph body → single string.
	idem, ok := sections["idempotency"].(string)
	if !ok {
		t.Fatalf("idempotency is %T, want string", sections["idempotency"])
	}
	if !strings.HasPrefix(idem, "Not idempotent") {
		t.Errorf("idempotency = %q, want it to start with 'Not idempotent'", idem)
	}
}

// TestAgentHelpToMapNormalizesKeys pins the title → snake_case key mapping
// that agents rely on to look up sections.
func TestAgentHelpToMapNormalizesKeys(t *testing.T) {
	cases := map[string]string{
		"Pre-conditions":         "pre_conditions",
		"Side effects":           "side_effects",
		"OAuth scope required":   "oauth_scope_required",
		"Post-conditions (on success)": "post_conditions",
		"Output (stdout, json mode)":   "output_(stdout,_json_mode)",
	}
	for title, wantKey := range cases {
		if got := normalizeSectionKey(title); got != wantKey {
			t.Errorf("normalizeSectionKey(%q) = %q, want %q", title, got, wantKey)
		}
	}
}

// TestRootAgentHelpRoundTrips ensures the root overview itself survives the
// JSON parser without losing its major sections — the root doc is what
// `gitee --agent-help -o json` returns.
func TestRootAgentHelpRoundTrips(t *testing.T) {
	out := agentHelpToMap(rootAgentHelp)
	sections, ok := out["sections"].(map[string]any)
	if !ok {
		t.Fatalf("sections is %T, want map[string]any", out["sections"])
	}
	for _, key := range []string{"what_this_is", "how_to_invoke", "design_philosophy"} {
		if _, present := sections[key]; !present {
			t.Errorf("root agent help JSON missing section %q (have %d sections)", key, len(sections))
		}
	}
}

// TestCoverageMatrixWellFormed parses every "--agent-help" line in the root
// coverage matrix and asserts each carries a recognized status marker. This
// catches a half-written or marker-less matrix row before it ships to agents.
func TestCoverageMatrixWellFormed(t *testing.T) {
	validMarkers := []string{"✅", "❌", "🆕", "n/a", "n.a."}
	seen := 0
	for _, line := range strings.Split(rootAgentHelp, "\n") {
		if !strings.Contains(line, "--agent-help") {
			continue
		}
		// Skip prose lines that merely mention the flag rather than
		// declaring a command's coverage status: anything not starting
		// with "gitee ", plus the generic usage template that uses
		// "<...>" placeholders instead of a concrete command name.
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "gitee ") || strings.Contains(trimmed, "<") {
			continue
		}
		seen++
		ok := false
		for _, m := range validMarkers {
			if strings.Contains(line, m) {
				ok = true
				break
			}
		}
		if !ok {
			t.Errorf("coverage matrix row has no status marker (✅/❌/🆕/n/a): %q", trimmed)
		}
	}
	if seen == 0 {
		t.Fatal("coverage matrix appears empty — expected several 'gitee <cmd> --agent-help' rows")
	}
}
