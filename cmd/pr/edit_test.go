package pr

import (
	"strings"
	"testing"
)

func TestEditAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee pr edit",
		"Purpose",
		"Pre-conditions",
		"Required arguments",
		"Required flags",
		"Optional flags",
		"Idempotency",
		"Atomicity",
		"Side effects",
		"Post-conditions",
		"Common errors",
		"Race condition warning ⚠️",
		"Next steps",
		"OAuth scope required",
		"Rate limit",
		"Example",
		"See also",
	}
	for _, want := range required {
		if !strings.Contains(prEditAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}

// TestEditAgentHelpRaceCondition asserts the race-condition section is
// command-specific: the PATCH is last-write-wins with no optimistic
// locking, and concurrent edits silently overwrite each other.
func TestEditAgentHelpRaceCondition(t *testing.T) {
	mustMention := []string{
		"last-write-wins",
		"silently overwrites",
		"gitee pr view",
	}
	for _, want := range mustMention {
		if !strings.Contains(prEditAgentHelp, want) {
			t.Errorf("agent help race section missing: %q", want)
		}
	}
}

// TestEditIDFlagRegistered verifies that --id/-i is registered on the
// edit command. Before this fix, editFlags.id was declared and used in
// runEdit but never registered in init(), so `gitee pr edit --id 42`
// failed with "unknown flag: --id". The sibling commands (comment,
// checks, ready) all register it — edit was the outlier.
func TestEditIDFlagRegistered(t *testing.T) {
	resetEditFlags()
	defer resetEditFlags()

	if err := editCmd.ParseFlags([]string{"--id", "42"}); err != nil {
		t.Fatalf("parsing --id flag: %v", err)
	}
	if editFlags.id != 42 {
		t.Errorf("expected editFlags.id=42 after --id 42, got %d", editFlags.id)
	}

	// Reset and verify the short form -i works too.
	resetEditFlags()
	if err := editCmd.ParseFlags([]string{"-i", "7"}); err != nil {
		t.Fatalf("parsing -i flag: %v", err)
	}
	if editFlags.id != 7 {
		t.Errorf("expected editFlags.id=7 after -i 7, got %d", editFlags.id)
	}
}

// resetEditFlags restores the package-level flag state between tests.
// Cobra preserves parsed flag values after ParseFlags, so the globals
// would otherwise leak across tests.
func resetEditFlags() {
	editCmd.SetArgs(nil)
	editFlags.id = 0
	editFlags.owner = ""
	editFlags.repo = ""
	editFlags.title = ""
	editFlags.body = ""
	editFlags.state = ""
	editFlags.draft = false
}
