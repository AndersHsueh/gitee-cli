package label

import (
	"gitee-cli/cmd"
	"gitee-cli/internal/handler"

	"github.com/spf13/cobra"
)

var cloneFlags struct {
	source string
	target string
}

var cloneCmd = &cobra.Command{
	Use:   "clone",
	Short: "Copy the entire label set from one repository to another",
	Example: `  gitee label clone --source owner/template-repo --target owner/new-repo
  gitee label clone --source myorg/template --target myorg/new-app`,
	PreRunE: func(c *cobra.Command, args []string) error {
		if cmd.GlobalAgentHelp {
			cmd.PrintAgentHelp(labelCloneAgentHelp)
		}
		return nil
	},
	RunE: func(c *cobra.Command, args []string) error {
		return handler.LabelClone(cloneFlags.source, cloneFlags.target)
	},
}

func init() {
	f := cloneCmd.Flags()
	f.StringVar(&cloneFlags.source, "source", "", "Source repository (owner/repo) to copy labels FROM")
	f.StringVar(&cloneFlags.target, "target", "", "Target repository (owner/repo) to copy labels TO")
}

const labelCloneAgentHelp = `# gitee label clone — Agent Help

## Purpose
Bulk-copy the entire label set (name + color) from one repository to
another. Useful when bootstrapping a new repo with a consistent
taxonomy, or when applying a template to multiple projects.

## Pre-conditions
- Authenticated with write access to BOTH repositories
- Source repo is readable

## Required flags
  --source   owner/repo   (where to copy labels FROM)
  --target   owner/repo   (where to copy labels TO)

## Idempotency
PARTIAL. Labels that already exist on the target are SKIPPED (not
overwritten). The command prints one line per label, and the final
summary reports how many failed (likely duplicates).

## Atomicity
NOT atomic. This is a sequential loop: each label is POSTed
individually. A network failure mid-loop leaves the target in a
partially-cloned state; re-running continues from where it stopped
(because duplicates are skipped, not errored).

## Side effects
- N new labels created on the target repo (one per source label)
- Webhooks subscribed to "Label" events may fire repeatedly

## Post-conditions (on success)
- The target repo has every label that the source has, with identical
  name + color
- Stdout shows ✓ / ✗ for each label plus a final summary

## Output (stdout)
  ✓ bug
  ✓ enhancement
  ✗ wontfix
  ✓ priority/high

  Cloned 3 label(s) from owner/template to owner/new-repo (1 failed, probably duplicates)

## Common errors
- 404 Not Found: source or target repo doesn't exist
- 403 Forbidden:  write access missing on target
- Per-label 409: label already exists on target (counted as failed, does NOT abort)

## Race condition warning ⚠️
The shared mutable state is BOTH the source repo's label set (read once
up front) and the target repo's label set (written incrementally).
- Labels added to the source after the initial list are NOT cloned in
  this run — re-run the command to converge (duplicates are skipped).
- A label created on the target between the source list and its POST is
  counted in the "failed, probably duplicates" tally; read the summary
  line, not the ✗ count, to judge the outcome.
- Two agents cloning into the same target concurrently interleave POSTs:
  each reports the other's successes as failures. The final state still
  converges, but both summaries over-report failures — serialize clones
  into the same target repo.
Safe to re-run after a mid-loop network failure: already-created labels
are skipped, so the retry continues where it stopped.

## OAuth scope required
projects (read on source, write on target)

## Rate limit
1000 req/hour; cloning a 50-label repo uses ~50 requests.

## Example
  gitee label clone --source myorg/template --target myorg/new-service

## See also
- ` + "`" + `gitee label list` + "`" + `   — inspect the source set
- ` + "`" + `gitee label create` + "`" + ` — add a single label
`
