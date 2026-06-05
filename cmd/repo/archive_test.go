package repo

import (
	"strings"
	"testing"
)

func TestArchiveAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee repo archive",
		"Purpose",
		"Idempotency",
		"Atomicity",
		"Side effects",
		"Post-conditions",
		"Common errors",
		"OAuth scope required",
		"Rate limit",
		"See also",
	}
	for _, want := range required {
		if !strings.Contains(repoArchiveAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}
