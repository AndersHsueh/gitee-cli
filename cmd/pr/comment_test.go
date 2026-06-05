package pr

import (
	"strings"
	"testing"
)

func TestCommentAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee pr comment",
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
		"OAuth scope required",
		"Rate limit",
		"Example",
		"See also",
	}
	for _, want := range required {
		if !strings.Contains(prCommentAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}
