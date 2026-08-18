package sshkey

import (
	"strings"
	"testing"
)

func TestDeleteAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee ssh-key delete",
		"Purpose",
		"Pre-conditions",
		"Required arguments",
		"Idempotency",
		"NOT idempotent",
		"Atomicity",
		"Side effects",
		"Post-conditions",
		"Common errors",
		"Race condition warning ⚠️",
		"OAuth scope required",
		"Rate limit",
		"See also",
	}
	for _, want := range required {
		if !strings.Contains(sshKeyDeleteAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
	// Spot-check the race section: concurrent delete of the same id
	// returns 404 and must not be retried.
	if !strings.Contains(sshKeyDeleteAgentHelp, "do NOT retry on 404") {
		t.Error("ssh-key delete race section missing no-retry-on-404 guidance")
	}
}
