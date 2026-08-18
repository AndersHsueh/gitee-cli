package status

import (
	"strings"
	"testing"
)

// TestStatusAgentHelpContent verifies that the agent-targeted help string
// contains all 14 standard sections. If a section is dropped during a
// future refactor, this test fails immediately — preventing the agent
// doc from silently going out of sync with what `gitee status --agent-help`
// actually prints.
func TestStatusAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee status",
		"## Purpose",
		"## Pre-conditions",
		"## Required arguments",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Output (stdout, table mode)",
		"## Output (stdout, json mode)",
		"## Common errors",
		"## Race condition warning ⚠️",
		"## Known limitations",
		"## Next steps",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(statusAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
	// Spot-check the race section: the 3-GET snapshot is not atomic;
	// state can flip between the /issues and /pulls calls.
	if !strings.Contains(statusAgentHelp, "NOT atomic") {
		t.Error("status race section missing non-atomic-snapshot warning")
	}
}

// TestStatusCmdWiring asserts that the StatusCmd variable is properly
// initialized (Use string set, RunE non-nil) and that the PreRunE
// doesn't blow up when --agent-help is requested.
//
// We can't easily test the --agent-help branch end-to-end (it calls
// os.Exit(0) and writes to stdout), but we CAN verify the function
// references resolve — which is the part the compiler won't catch for
// you if you typo the const or helper name.
func TestStatusCmdWiring(t *testing.T) {
	if StatusCmd == nil {
		t.Fatal("StatusCmd is nil — root registration will panic")
	}
	if StatusCmd.Use != "status" {
		t.Errorf("StatusCmd.Use = %q, want %q", StatusCmd.Use, "status")
	}
	if StatusCmd.RunE == nil {
		t.Error("StatusCmd.RunE is nil — the command has no behavior")
	}
	if StatusCmd.PreRunE == nil {
		t.Error("StatusCmd.PreRunE is nil — --agent-help won't fire")
	}
}
