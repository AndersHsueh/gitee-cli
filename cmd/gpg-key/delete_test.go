package gpgkey

import (
	"strings"
	"testing"
)

func TestDeleteAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee gpg-key delete",
		"Purpose",
		"Pre-conditions",
		"Required arguments",
		"Idempotency",
		"NOT idempotent",
		"Atomicity",
		"Side effects",
		"Post-conditions",
		"Common errors",
		"OAuth scope required",
		"Rate limit",
		"See also",
	}
	for _, want := range required {
		if !strings.Contains(gpgKeyDeleteAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}
