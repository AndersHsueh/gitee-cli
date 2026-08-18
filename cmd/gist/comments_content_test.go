package gist

import (
	"strings"
	"testing"
)

// TestGistCommentsAgentHelpContent verifies the plural LIST verb's
// agent-targeted help string contains all the standard sections.
func TestGistCommentsAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee gist comments",
		"## Purpose",
		"## Pre-conditions",
		"## Required arguments",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Output (stdout, table mode)",
		"## Output (stdout, json mode)",
		"## Common errors",
		"## Race condition warning ⚠️",
		"## Next steps (orchestration hints)",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(gistCommentsAgentHelp, want) {
			t.Errorf("gist comments --agent-help missing section: %q", want)
		}
	}
}

// TestGistCommentAgentHelpContent verifies the singular CREATE verb's
// agent-targeted help string contains all the standard sections.
func TestGistCommentAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee gist comment",
		"## Purpose",
		"## Pre-conditions",
		"## Required arguments",
		"## Required flags",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Common errors",
		"## Race condition warning ⚠️",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(gistCommentAgentHelp, want) {
			t.Errorf("gist comment --agent-help missing section: %q", want)
		}
	}
}

// TestGistCommentDeleteAgentHelpContent verifies the DELETE verb's
// agent-targeted help string contains all the standard sections.
func TestGistCommentDeleteAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee gist comment-delete",
		"## Purpose",
		"## Pre-conditions",
		"## Required arguments",
		"## Required flags",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Output (stdout, text mode)",
		"## Common errors",
		"## Race condition warning ⚠️",
		"## Next steps (orchestration hints)",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(gistCommentDeleteAgentHelp, want) {
			t.Errorf("gist comment-delete --agent-help missing section: %q", want)
		}
	}
}

// TestGistCommentFlagsCoveredInAgentHelp ensures the flags exposed by
// the comments / comment / comment-delete subcommands are also
// documented in their respective agent help blocks.
func TestGistCommentFlagsCoveredInAgentHelp(t *testing.T) {
	cases := []struct {
		flag     string
		inString string
	}{
		{"--limit", gistCommentsAgentHelp},
		{"-L", gistCommentsAgentHelp},
		{"--body", gistCommentAgentHelp},
		{"-b", gistCommentAgentHelp},
		{"--yes", gistCommentDeleteAgentHelp},
	}
	for _, c := range cases {
		if !strings.Contains(c.inString, c.flag) {
			t.Errorf("flag %q must be mentioned in agent help", c.flag)
		}
	}
}

// TestGistCommentRaceSections verifies each of the three comment
// subcommands has a command-specific Race condition section with no
// placeholder text left over.
func TestGistCommentRaceSections(t *testing.T) {
	cases := []struct {
		name     string
		doc      string
		wantText string
	}{
		{"comments", gistCommentsAgentHelp, "indexing lag"},
		{"comment", gistCommentAgentHelp, "posts the SAME comment twice"},
		{"comment-delete", gistCommentDeleteAgentHelp, "treat 404 as success"},
	}
	for _, c := range cases {
		if strings.Contains(c.doc, "TODO(domain)") {
			t.Errorf("gist %s --agent-help still contains TODO(domain) placeholder", c.name)
		}
		if !strings.Contains(c.doc, c.wantText) {
			t.Errorf("gist %s race section missing expected content %q", c.name, c.wantText)
		}
	}
}
