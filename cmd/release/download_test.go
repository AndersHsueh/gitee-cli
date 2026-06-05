package release

import (
	"strings"
	"testing"
)

func TestDownloadAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee release download",
		"Purpose",
		"Idempotency",
		"Atomicity",
		"Side effects",
		"Post-conditions",
		"Common errors",
		"See also",
	}
	for _, want := range required {
		if !strings.Contains(releaseDownloadAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}
