package pr

import (
	"slices"
	"strings"
	"testing"
)

func TestCheckoutAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee pr checkout",
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
		"See also",
	}
	for _, want := range required {
		if !strings.Contains(prCheckoutAgentHelp, want) {
			t.Errorf("agent help missing required section: %q", want)
		}
	}
}

func TestCheckoutHasCoAlias(t *testing.T) {
	if !slices.Contains(checkoutCmd.Aliases, "co") {
		t.Errorf("expected checkoutCmd to have alias 'co' (matches gh's `gh co` shortcut)")
	}
}
