package issue

import (
	"strings"
	"testing"
)

func TestCommentsAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee issue comments",
		"Purpose",
		"Pre-conditions",
		"Required arguments",
		"Optional flags",
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
		if !strings.Contains(issueCommentsAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}
