package pr

import (
	"strings"
	"testing"
)

// TestCreateAgentHelpContent verifies that the agent help text contains
// every section promised in the documentation. This is the static contract
// that agents rely on: if a section is removed from the text, this test
// fails and forces the maintainer to update the docs.
//
// The PR-create help must additionally call out the "closing the loop"
// workflow because that's the #1 reason orphan issues get left in repos
// when an agent completes a fix but doesn't link it back.
//
// Note: the runtime behavior (PreRunE firing before ValidateRequiredFlags)
// is verified manually because PreRunE calls os.Exit(0) which would
// terminate the test process.
func TestCreateAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee pr create",
		"Purpose",
		"Pre-conditions",
		"Idempotency",
		"Atomicity",
		"Side effects",
		"Post-conditions",
		"Common errors",
		"OAuth scope required",
		"Rate limit",
		"Next steps",
		"closing the loop", // PR-specific: must mention issue comment
		"See also",
	}
	for _, want := range required {
		if !strings.Contains(prCreateAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}
