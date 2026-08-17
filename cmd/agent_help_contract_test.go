package cmd

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// This file locks the parts of the --agent-help system that AI agents
// actually consume at runtime, and that no single command's test can
// protect on its own:
//
//  1. The markdown → JSON parser (agentHelpToMap). Agents that run
//     `gitee <cmd> --agent-help -o json` depend on its exact shape:
//     a top-level "command" string plus a "sections" object whose keys
//     are normalized section titles. A regression here silently breaks
//     every downstream agent workflow.
//  2. The root coverage matrix (rootAgentHelp). It is hand-maintained
//     text; without a test it rots the moment someone adds a command
//     and forgets the matrix row.
//
// Why not "execute every command with --agent-help and assert the
// sections"? Because PrintAgentHelp calls os.Exit(0) by design (so that
// `gitee foo --agent-help` short-circuits before the command runs), which
// makes in-process execution-based coverage impractical. Per-command
// section presence is instead covered by each package's own *content_test.go
// (e.g. cmd/alias/content_test.go); this file guards the shared machine
// contract those tests cannot see.

// standardAgentDimensions are the cross-cutting concerns the root help
// promises that --agent-help adds on top of --help. If the root doc stops
// advertising one of these, the "two surfaces" contract has drifted.
var standardAgentDimensions = []string{
	"Pre-conditions",
	"Post-conditions",
	"Idempotency",
	"Atomicity",
	"Side effects",
	"error",       // "Common error codes + how to recover"
	"OAuth scope", // required scope
	"Rate limit",
	"Race conditions",
	"orchestration", // "Next-step orchestration hints"
	"Parsing hints",
}

// TestRootAgentHelpAdvertisesAllDimensions guards the headline promise of
// the dual-surface design: --agent-help is --help PLUS these dimensions.
func TestRootAgentHelpAdvertisesAllDimensions(t *testing.T) {
	for _, dim := range standardAgentDimensions {
		if !strings.Contains(rootAgentHelp, dim) {
			t.Errorf("rootAgentHelp no longer advertises the %q dimension — the dual-surface contract has drifted", dim)
		}
	}
}

// TestAgentHelpToMapShape locks the JSON object shape agents parse.
func TestAgentHelpToMapShape(t *testing.T) {
	const sample = `# gitee demo create — Agent Help

## Purpose
Create a demo resource.

## Pre-conditions
- You are authenticated.
- The repo exists.

## Idempotency
Not idempotent: re-running creates a second resource.
`
	out := agentHelpToMap(sample)

	// command name: heading text with the "— Agent Help" suffix stripped.
	if got, want := out["command"], "gitee demo create"; got != want {
		t.Errorf("command = %q, want %q", got, want)
	}

	sections, ok := out["sections"].(map[string]any)
	if !ok {
		t.Fatalf("sections is %T, want map[string]any", out["sections"])
	}

	// List body → []string with backticks stripped and "- " removed.
	pre, ok := sections["pre_conditions"].([]string)
	if !ok {
		t.Fatalf("pre_conditions is %T, want []string", sections["pre_conditions"])
	}
	if len(pre) != 2 || pre[0] != "You are authenticated." {
		t.Errorf("pre_conditions = %#v, want 2 items starting with the auth precondition", pre)
	}

	// Paragraph body → single string.
	idem, ok := sections["idempotency"].(string)
	if !ok {
		t.Fatalf("idempotency is %T, want string", sections["idempotency"])
	}
	if !strings.HasPrefix(idem, "Not idempotent") {
		t.Errorf("idempotency = %q, want it to start with 'Not idempotent'", idem)
	}
}

// TestAgentHelpToMapNormalizesKeys pins the title → snake_case key mapping
// that agents rely on to look up sections.
func TestAgentHelpToMapNormalizesKeys(t *testing.T) {
	cases := map[string]string{
		"Pre-conditions":         "pre_conditions",
		"Side effects":           "side_effects",
		"OAuth scope required":   "oauth_scope_required",
		"Post-conditions (on success)": "post_conditions",
		"Output (stdout, json mode)":   "output_(stdout,_json_mode)",
	}
	for title, wantKey := range cases {
		if got := normalizeSectionKey(title); got != wantKey {
			t.Errorf("normalizeSectionKey(%q) = %q, want %q", title, got, wantKey)
		}
	}
}

// TestRootAgentHelpRoundTrips ensures the root overview itself survives the
// JSON parser without losing its major sections — the root doc is what
// `gitee --agent-help -o json` returns.
func TestRootAgentHelpRoundTrips(t *testing.T) {
	out := agentHelpToMap(rootAgentHelp)
	sections, ok := out["sections"].(map[string]any)
	if !ok {
		t.Fatalf("sections is %T, want map[string]any", out["sections"])
	}
	for _, key := range []string{"what_this_is", "how_to_invoke", "design_philosophy"} {
		if _, present := sections[key]; !present {
			t.Errorf("root agent help JSON missing section %q (have %d sections)", key, len(sections))
		}
	}
}

// TestCoverageMatrixWellFormed parses every "--agent-help" line in the root
// coverage matrix and asserts each carries a recognized status marker. This
// catches a half-written or marker-less matrix row before it ships to agents.
func TestCoverageMatrixWellFormed(t *testing.T) {
	validMarkers := []string{"✅", "❌", "🆕", "n/a", "n.a."}
	seen := 0
	for _, line := range strings.Split(rootAgentHelp, "\n") {
		if !strings.Contains(line, "--agent-help") {
			continue
		}
		// Skip prose lines that merely mention the flag rather than
		// declaring a command's coverage status: anything not starting
		// with "gitee ", plus the generic usage template that uses
		// "<...>" placeholders instead of a concrete command name.
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, "gitee ") || strings.Contains(trimmed, "<") {
			continue
		}
		seen++
		ok := false
		for _, m := range validMarkers {
			if strings.Contains(line, m) {
				ok = true
				break
			}
		}
		if !ok {
			t.Errorf("coverage matrix row has no status marker (✅/❌/🆕/n/a): %q", trimmed)
		}
	}
	if seen == 0 {
		t.Fatal("coverage matrix appears empty — expected several 'gitee <cmd> --agent-help' rows")
	}
}

// exemptHelpPathPrefixes lists the cobra subcommand path prefixes whose
// leaf commands are exempt from the --agent-help contract check below.
//
//   - "help"                  — the cobra help dispatcher itself; users
//                              should run `gitee help <topic>` instead.
//   - "help environment"      — `gitee help <topic>` IS the documentation
//   - "help exit-codes"         for those topics. Their --agent-help can
//   - "help formatting"         legitimately return the topic content
//   - "help reference"          itself (matrix marks them "n/a").
//   - "co"                    — hidden alias of `gitee pr checkout`;
//                              agent-help coverage is provided by
//                              `gitee pr checkout --agent-help` so
//                              adding it here would duplicate work
//                              and confuse the matrix.
//
// New exemptions should be added deliberately and called out in the
// commit message — the whole point of this test is that any new
// leaf command without --agent-help coverage is loud, not silent.
var exemptHelpPathPrefixes = []string{
	"help",
	"help environment",
	"help exit-codes",
	"help formatting",
	"help reference",
	"co",
}

// TestAgentHelpCoverageForAllCommands walks the live cobra command
// tree (via cmd.RootCommand()) and asserts that every leaf command,
// when probed with --agent-help, returns EITHER:
//
//  1. The operational/config n/a notice ("<prefix> is an
//     operational/config command. Use --help for usage.") for
//     verbs that don't have agent-targeted docs; OR
//  2. The full agent-help markdown ("# gitee <name> — Agent Help")
//     for verbs that do.
//
// This is the IK923X step-2 contract test: it would have caught
// `gitee completion bash --agent-help` dumping 200 lines of shell
// script, and `gitee version --agent-help` dumping the version
// string, because both fail the prefix assertion. Before this test
// landed, TestCoverageMatrixWellFormed only validated that matrix
// rows were well-formed; it never verified that the matrix
// actually covered every leaf in the tree.
//
// The check is run by shelling out to a freshly-built gitee binary
// (buildThrowawayBinary, defined in version_test.go) so we exercise
// the real Cobra flag-parsing + PreRunE pipeline — not in-process
// mocks. In-process testing of os.Exit(0) is impractical, which is
// also why this test lives at the contract level rather than inside
// each subcommand's package.
func TestAgentHelpCoverageForAllCommands(t *testing.T) {
	bin := buildThrowawayBinary(t)

	leaves := findAllLeafCommandsForCoverage(RootCommand())
	if len(leaves) == 0 {
		t.Fatal("no leaf commands discovered — registration may have regressed")
	}
	t.Logf("discovered %d leaf commands to probe", len(leaves))

	for _, leaf := range leaves {
		t.Run(leaf.CommandPath(), func(t *testing.T) {
			if isExemptedFromAgentHelpCoverage(leaf) {
				t.Skipf("exempted: %s (see exemptHelpPathPrefixes)", leaf.CommandPath())
			}
			args := strings.Fields(leaf.CommandPath())[1:] // drop "gitee"
			args = append(args, "--agent-help")
			run := exec.Command(bin, args...)
			run.Env = []string{
				"HOME=" + t.TempDir(),
				"PATH=/usr/bin:/bin",
				"GITEE_TOKEN=",
				"GITEE_HOSTNAME=invalid.example.invalid",
			}
			out, err := run.CombinedOutput()
			if err != nil {
				t.Fatalf("%s --agent-help failed: %v\noutput:\n%s",
					leaf.CommandPath(), err, out)
			}
			combined := string(out)
			// Combine all n/a variants we accept:
			//   - "X is an operational/config command. Use --help for usage." (current wording)
			//   - "X is an operational/config — use --help" (matrix shorthand; only inside rootAgentHelp)
			hasNA := strings.Contains(combined, "is an operational/config command") &&
				strings.Contains(combined, "Use --help for usage")
			hasFullDoc := strings.HasPrefix(strings.TrimLeft(combined, "\n"), "# gitee")
			if !hasNA && !hasFullDoc {
				t.Errorf("%s --agent-help output must start with the operational/config n/a notice "+
					"or a '# gitee ... — Agent Help' header. Got:\n%s",
					leaf.CommandPath(), combined)
			}
		})
	}
}

// findAllLeafCommandsForCoverage recursively walks the cobra tree and
// returns every command with no subcommands. We do not filter hidden
// commands here (the help_topics dispatcher itself is a parent and
// is naturally excluded by the "no subcommands" check; the four
// topics are leaves and are handled by the exemption list above).
func findAllLeafCommandsForCoverage(root *cobra.Command) []*cobra.Command {
	var leaves []*cobra.Command
	var walk func(c *cobra.Command)
	walk = func(c *cobra.Command) {
		children := c.Commands()
		if len(children) == 0 {
			leaves = append(leaves, c)
			return
		}
		for _, child := range children {
			walk(child)
		}
	}
	walk(root)
	return leaves
}

// isExemptedFromAgentHelpCoverage returns true if the leaf's command
// path is in the exemptHelpPathPrefixes allowlist. We compare against
// CommandPath() (e.g. "gitee help environment", "gitee co") so the
// list can be expressed in user-visible form.
func isExemptedFromAgentHelpCoverage(leaf *cobra.Command) bool {
	full := leaf.CommandPath()
	for _, exempt := range exemptHelpPathPrefixes {
		if full == "gitee "+exempt {
			return true
		}
	}
	return false
}
