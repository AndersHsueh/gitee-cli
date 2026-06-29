package repo

import (
	"strings"
	"testing"
)

// Each Test<Verb>AgentHelpContent verifies the verb's agent-targeted help
// string contains all the standard sections expected by the contract
// test in cmd/agent_help_contract_test.go. If a section is dropped during
// a refactor, this test fails immediately — keeping the doc in sync with
// what `gitee repo <verb> --agent-help` actually prints.

func TestListAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee repo list",
		"## Purpose",
		"## Pre-conditions",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Output (stdout, table mode)",
		"## Output (stdout, json mode)",
		"## Common errors",
		"## Next steps (orchestration hints)",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(repoListAgentHelp, want) {
			t.Errorf("repo list --agent-help missing section: %q", want)
		}
	}
}

func TestCreateAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee repo create",
		"## Purpose",
		"## Pre-conditions",
		"## Required flags",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Common errors",
		"## Next steps (orchestration hints)",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(repoCreateAgentHelp, want) {
			t.Errorf("repo create --agent-help missing section: %q", want)
		}
	}
}

func TestViewAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee repo view",
		"## Purpose",
		"## Pre-conditions",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Output (stdout, table mode)",
		"## Output (stdout, json mode)",
		"## Common errors",
		"## Next steps (orchestration hints)",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(repoViewAgentHelp, want) {
			t.Errorf("repo view --agent-help missing section: %q", want)
		}
	}
}

func TestForkAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee repo fork",
		"## Purpose",
		"## Pre-conditions",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Common errors",
		"## Next steps (orchestration hints)",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(repoForkAgentHelp, want) {
			t.Errorf("repo fork --agent-help missing section: %q", want)
		}
	}
}

func TestEditAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee repo edit",
		"## Purpose",
		"## Pre-conditions",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Common errors",
		"## Next steps (orchestration hints)",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(repoEditAgentHelp, want) {
			t.Errorf("repo edit --agent-help missing section: %q", want)
		}
	}
}

func TestDeleteAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee repo delete",
		"## Purpose",
		"## Pre-conditions",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Common errors",
		"## Next steps (orchestration hints)",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(repoDeleteAgentHelp, want) {
			t.Errorf("repo delete --agent-help missing section: %q", want)
		}
	}
}

func TestCloneAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee repo clone",
		"## Purpose",
		"## Pre-conditions",
		"## Required arguments",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Common errors",
		"## Next steps (orchestration hints)",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(repoCloneAgentHelp, want) {
			t.Errorf("repo clone --agent-help missing section: %q", want)
		}
	}
}

func TestBrowseAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee repo browse",
		"## Purpose",
		"## Pre-conditions",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Common errors",
		"## Next steps (orchestration hints)",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(repoBrowseAgentHelp, want) {
			t.Errorf("repo browse --agent-help missing section: %q", want)
		}
	}
}

func TestBranchesAgentHelpContent(t *testing.T) {
	required := []string{
		"# gitee repo branches",
		"## Purpose",
		"## Pre-conditions",
		"## Optional flags",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Post-conditions",
		"## Output (stdout, table mode)",
		"## Output (stdout, json mode)",
		"## Common errors",
		"## Next steps (orchestration hints)",
		"## OAuth scope required",
		"## Rate limit",
		"## Example",
		"## See also",
	}
	for _, want := range required {
		if !strings.Contains(repoBranchesAgentHelp, want) {
			t.Errorf("repo branches --agent-help missing section: %q", want)
		}
	}
}

// TestRepoFlagsCoveredInAgentHelp is a "lint-level" guard ensuring every
// flag exposed by the verbs above is also documented in its agent help
// string. If a new flag is added to a cobra command without updating the
// matching const, this test fails — preventing the divergence where
// `--flag` works but `--agent-help` doesn't list it.
func TestRepoFlagsCoveredInAgentHelp(t *testing.T) {
	cases := []struct {
		flag     string
		inString string
	}{
		// list.go
		{"--owner", repoListAgentHelp},
		{"--visibility", repoListAgentHelp},
		{"--limit", repoListAgentHelp},
		{"-L", repoListAgentHelp},
		{"--sort", repoListAgentHelp},
		{"--direction", repoListAgentHelp},

		// create.go
		{"--name", repoCreateAgentHelp},
		{"-n", repoCreateAgentHelp},
		{"--description", repoCreateAgentHelp},
		{"-d", repoCreateAgentHelp},
		{"--private", repoCreateAgentHelp},
		{"-p", repoCreateAgentHelp},
		{"--auto-init", repoCreateAgentHelp},
		{"--gitignore", repoCreateAgentHelp},
		{"--license", repoCreateAgentHelp},
		{"--org", repoCreateAgentHelp},

		// view.go
		{"--owner", repoViewAgentHelp},
		{"--repo", repoViewAgentHelp},
		{"-R", repoViewAgentHelp},

		// fork.go
		{"--owner", repoForkAgentHelp},
		{"--repo", repoForkAgentHelp},
		{"-R", repoForkAgentHelp},

		// edit.go
		{"--owner", repoEditAgentHelp},
		{"--repo", repoEditAgentHelp},
		{"-R", repoEditAgentHelp},
		{"--name", repoEditAgentHelp},
		{"--description", repoEditAgentHelp},
		{"-d", repoEditAgentHelp},
		{"--private", repoEditAgentHelp},
		{"--default-branch", repoEditAgentHelp},

		// delete.go
		{"--owner", repoDeleteAgentHelp},
		{"--repo", repoDeleteAgentHelp},
		{"-R", repoDeleteAgentHelp},
		{"--yes", repoDeleteAgentHelp},

		// branches.go
		{"--repo", repoBranchesAgentHelp},
		{"-R", repoBranchesAgentHelp},
	}
	for _, c := range cases {
		if !strings.Contains(c.inString, c.flag) {
			t.Errorf("flag %q must be mentioned in agent help", c.flag)
		}
	}
}
