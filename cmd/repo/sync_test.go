package repo

import (
	"strings"
	"testing"
)

func TestSyncAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee repo sync",
		"Purpose",
		"Idempotency",
		"Atomicity",
		"Side effects",
		"Post-conditions",
		"Common errors",
		"See also",
	}
	for _, want := range required {
		if !strings.Contains(repoSyncAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}
