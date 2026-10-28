package cmd

import (
	"os/exec"
	"strings"
	"testing"
)

// TestAgentHelpSchemaCompleteness is the IK923X step-4 contract test.
//
// TestAgentHelpCoverageForAllCommands (agent_help_contract_test.go) only
// asserts that every leaf command answers --agent-help with EITHER the
// operational/config n/a notice OR a full doc header ("# gitee ...").
// It says nothing about what's INSIDE the full doc. This test closes
// that gap: every full doc must carry the 11 cross-cutting dimensions
// the root overview (rootAgentHelp) promises agents they will find:
//
//	Pre-conditions, Post-conditions, Idempotency, Atomicity,
//	Side effects, Common errors, OAuth scope, Rate limit,
//	Race condition, Example, and (Next steps or See also).
//
// Plus two hygiene rules that content authors routinely break:
//
//   - no "TODO(domain)" placeholder may survive into shipped docs;
//   - every "N/A" must carry a "(reason)" — a bare N/A tells an agent
//     nothing about WHY the dimension does not apply.
//
// Matching is line-prefix based on trimmed lines, so heading variants
// like "## Post-conditions (on success)", "## Race condition warning ⚠️",
// "## Example (closing the loop)" and "## Side effects (IRREVERSIBLE)"
// all satisfy their dimension. The requirement is that the dimension is
// present and findable by prefix, not that every doc uses byte-identical
// headings.
//
// Leaf enumeration: unlike the coverage test, which walks the in-process
// cobra tree via findAllLeafCommandsForCoverage(RootCommand()), this test
// derives the leaf list from the freshly-built binary's
// `gitee help reference` output. Reason: every verb except completion
// and version lives in a subpackage (cmd/repo, cmd/issue, ...) that
// imports gitee-cli/cmd, so a test inside package cmd cannot import them
// (import cycle) and the in-process tree only ever contains 5 leaves.
// The binary, built through cmd/gitee/main.go, carries the full tree;
// `help reference` prints every command path, and a path is a leaf when
// no other path extends it. The only hidden command (`gitee co`) is on
// the exemption list anyway, so nothing testable is lost.
//
// Like the coverage test, this shells out (buildThrowawayBinary from
// version_test.go) because PrintAgentHelp calls os.Exit(0) by design,
// making in-process probing impractical.
func TestAgentHelpSchemaCompleteness(t *testing.T) {
	bin := buildThrowawayBinary(t)

	// requiredSections are matched as prefixes against "## "-level
	// headings. Order follows the reading order the root doc advertises.
	requiredSections := []string{
		"## Pre-conditions",
		"## Post-conditions",
		"## Idempotency",
		"## Atomicity",
		"## Side effects",
		"## Common errors",
		"## OAuth scope",
		"## Rate limit",
		"## Race condition",
		"## Example",
	}
	// orchestrationAlternatives: at least one must be present. Docs pick
	// whichever fits; both signal "what to run after this command".
	orchestrationAlternatives := []string{"## Next steps", "## See also"}

	leafPaths := findLeafCommandPathsFromBinary(t, bin)
	if len(leafPaths) == 0 {
		t.Fatal("no leaf commands discovered from binary — `help reference` may have regressed")
	}
	t.Logf("discovered %d leaf commands to schema-check", len(leafPaths))

	for _, path := range leafPaths {
		t.Run(path, func(t *testing.T) {
			if isExemptedPathFromAgentHelpCoverage(path) {
				t.Skipf("exempted: %s (see exemptHelpPathPrefixes)", path)
			}
			args := strings.Fields(path)[1:] // drop "gitee"
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
				t.Fatalf("%s --agent-help failed: %v\noutput:\n%s", path, err, out)
			}
			combined := string(out)

			// operational/config n/a signal: no full doc to schema-check.
			if strings.Contains(combined, "is an operational/config command") &&
				strings.Contains(combined, "Use --help for usage") {
				t.Skipf("operational/config n/a signal — no full doc to schema-check")
			}
			if !strings.HasPrefix(strings.TrimLeft(combined, "\n"), "# gitee") {
				t.Fatalf("%s --agent-help is neither an n/a notice nor a full doc "+
					"(coverage contract violated). Got:\n%s", path, combined)
			}

			// Title contract: the document's first line must name the exact
			// command path the user (or agent) probed. Shared docs are
			// retitled with cmd.PrintAgentHelpForCommand; a mismatch means an
			// alias or umbrella leaked another command's title, and agents
			// would conclude they were handed the wrong command's docs.
			firstLine := ""
			for _, line := range strings.Split(combined, "\n") {
				if strings.HasPrefix(strings.TrimSpace(line), "# ") {
					firstLine = strings.TrimSpace(line)
					break
				}
			}
			if wantTitle := "# " + path + " — Agent Help"; firstLine != wantTitle {
				t.Errorf("%s agent-help title = %q, want %q", path, firstLine, wantTitle)
			}

			// Collect headings for prefix matching.
			var headings []string
			for _, line := range strings.Split(combined, "\n") {
				trimmed := strings.TrimSpace(line)
				if strings.HasPrefix(trimmed, "## ") {
					headings = append(headings, trimmed)
				}
			}
			hasSection := func(prefix string) bool {
				for _, h := range headings {
					if strings.HasPrefix(h, prefix) {
						return true
					}
				}
				return false
			}

			for _, sec := range requiredSections {
				if !hasSection(sec) {
					t.Errorf("%s agent-help missing required section %q (have headings: %s)",
						path, sec, strings.Join(headings, " | "))
				}
			}
			orchestrationFound := false
			for _, alt := range orchestrationAlternatives {
				if hasSection(alt) {
					orchestrationFound = true
					break
				}
			}
			if !orchestrationFound {
				t.Errorf("%s agent-help missing orchestration section: need %q or %q (have headings: %s)",
					path, orchestrationAlternatives[0], orchestrationAlternatives[1],
					strings.Join(headings, " | "))
			}

			// Hygiene: no leftover placeholder.
			if strings.Contains(combined, "TODO(domain)") {
				t.Errorf("%s agent-help still contains a TODO(domain) placeholder", path)
			}

			// Hygiene: bare "N/A" is forbidden — every N/A must explain
			// itself with a "(reason)" on the same line.
			for i, line := range strings.Split(combined, "\n") {
				if strings.Contains(line, "N/A") && !strings.Contains(line, "(") {
					t.Errorf("%s agent-help line %d has a bare N/A without a (reason): %q",
						path, i+1, strings.TrimSpace(line))
				}
			}
		})
	}
}

// findCommandPathsFromBinary runs `gitee help reference` on the given
// binary and returns every command path it lists (parents and leaves),
// e.g. "gitee pr", "gitee pr create". `help reference` walks the same
// live cobra tree the binary executes, so this list always matches what
// users and agents can actually invoke.
//
// It fails the test if the same path is listed twice: cobra only allows
// one child per name, so a duplicate means a *cobra.Command was added to
// its parent twice (this actually happened to the `issue label` subtree,
// which printed twice in `help reference` until the duplicate AddCommand
// call was removed).
func findCommandPathsFromBinary(t *testing.T, bin string) []string {
	t.Helper()
	run := exec.Command(bin, "help", "reference")
	run.Env = []string{
		"HOME=" + t.TempDir(),
		"PATH=/usr/bin:/bin",
		"GITEE_TOKEN=",
		"GITEE_HOSTNAME=invalid.example.invalid",
	}
	out, err := run.CombinedOutput()
	if err != nil {
		t.Fatalf("gitee help reference failed: %v\noutput:\n%s", err, out)
	}

	var paths []string
	seen := make(map[string]bool)
	for i, line := range strings.Split(string(out), "\n") {
		if i == 0 || strings.TrimSpace(line) == "" {
			continue // header row ("COMMAND  SHORT") and blank lines
		}
		// Path and description are separated by a run of 2+ spaces;
		// the path itself uses single spaces between words.
		fields := strings.Fields(line)
		var words []string
		for _, f := range fields {
			words = append(words, f)
			// Stop at the first word that cannot be part of a command
			// path: descriptions start with an uppercase letter or a
			// non-command token, while every path segment is lowercase.
			if f != strings.ToLower(f) {
				words = words[:len(words)-1]
				break
			}
		}
		if len(words) < 2 || words[0] != "gitee" {
			continue
		}
		path := strings.Join(words, " ")
		if seen[path] {
			t.Errorf("`gitee help reference` lists %q twice — a command was registered on its parent more than once", path)
			continue
		}
		seen[path] = true
		paths = append(paths, path)
	}
	return paths
}

// findLeafCommandPathsFromBinary returns the subset of findCommandPathsFromBinary's
// paths that are leaves — i.e. no other listed path extends them.
func findLeafCommandPathsFromBinary(t *testing.T, bin string) []string {
	t.Helper()
	return leafCommandPaths(findCommandPathsFromBinary(t, bin))
}

// leafCommandPaths filters allPaths down to the paths no other path extends.
func leafCommandPaths(allPaths []string) []string {
	isParent := make(map[string]bool, len(allPaths))
	for _, p := range allPaths {
		for _, q := range allPaths {
			if q != p && strings.HasPrefix(q, p+" ") {
				isParent[p] = true
				break
			}
		}
	}
	var leaves []string
	for _, p := range allPaths {
		if !isParent[p] {
			leaves = append(leaves, p)
		}
	}
	return leaves
}

// isExemptedPathFromAgentHelpCoverage is the path-string counterpart of
// isExemptedFromAgentHelpCoverage (which takes a *cobra.Command we do
// not have here — see the TestAgentHelpSchemaCompleteness doc comment
// for why enumeration is binary-driven). It consults the same
// exemptHelpPathPrefixes allowlist, so both tests share one source of
// truth for exemptions.
func isExemptedPathFromAgentHelpCoverage(path string) bool {
	for _, exempt := range exemptHelpPathPrefixes {
		if path == "gitee "+exempt {
			return true
		}
	}
	return false
}
