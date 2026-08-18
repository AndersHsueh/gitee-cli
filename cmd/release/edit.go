package release

import (
	"context"
	"fmt"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var editFlags struct {
	repo       string
	tag        string
	name       string
	body       string
	prerelease bool
	clearPre   bool
}

var editCmd = &cobra.Command{
	Use:   "edit <tag>",
	Short: "Edit an existing release",
	Example: `  gitee release edit v1.0.0 --repo owner/repo --name "1.0.0 Stable"
  gitee release edit v1.0.0 -R owner/repo --body "Updated release notes"`,
	PreRunE: func(c *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(releaseEditAgentHelp)
		}
		return nil
	},
	RunE: runEdit,
}

func init() {
	f := editCmd.Flags()
	f.StringVarP(&editFlags.repo, "repo", "R", "", "Repository (owner/repo)")
	f.StringVarP(&editFlags.tag, "tag", "t", "", "Tag name (or pass as first arg)")
	f.StringVarP(&editFlags.name, "name", "n", "", "New display name")
	f.StringVarP(&editFlags.body, "body", "b", "", "New release notes (Markdown)")
	f.BoolVar(&editFlags.prerelease, "prerelease", false, "Mark as prerelease")
	f.BoolVar(&editFlags.clearPre, "no-prerelease", false, "Unset prerelease flag")
}

func runEdit(cobraCmd *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(cobraCmd)
	tag := editFlags.tag
	if tag == "" && len(args) >= 1 {
		tag = args[0]
	}
	if tag == "" {
		return fmt.Errorf("tag is required (use --tag or pass as first arg)")
	}

	owner, repoName, err := resolveReleaseRepo(editFlags.repo)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClientFrom(rt)
	if err != nil {
		return err
	}

	// Build patch payload: only include fields the user actually passed.
	// For prerelease we use a *bool so we can distinguish "set true" / "set false" / "unset".
	req := types.ReleaseEditRequest{}
	if editFlags.name != "" {
		req.Name = editFlags.name
	}
	if editFlags.body != "" {
		req.Body = editFlags.body
	}
	if editFlags.prerelease && !editFlags.clearPre {
		t := true
		req.Prerelease = &t
	} else if editFlags.clearPre {
		f := false
		req.Prerelease = &f
	}

	path := api.ReleaseEdit(owner, repoName, tag)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rt.Timeout)*time.Second)
	defer cancel()

	var release types.Release
	if err := client.Patch(ctx, path, req, &release); err != nil {
		return fmt.Errorf("editing release: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	fmt.Printf("%s Release %s updated\n", green("✓"), bold(tag))
	fmt.Printf("  Name:  %s\n", release.Name)
	fmt.Printf("  URL:   %s\n", color.YellowString(release.HTMLURL))
	return nil
}

const releaseEditAgentHelp = `# gitee release edit — Agent Help

## Purpose
Modify the metadata of an existing release (name, body, prerelease flag)
WITHOUT deleting and re-creating it. The underlying git tag is preserved.

## Pre-conditions
- Authenticated with write access
- The release for --tag already exists (returns 404 otherwise)

## Required flags
  --tag / -t  (or first positional arg)   Tag name

## Optional flags
  --repo / -R         owner/repo
  --name / -n         New display name
  --body / -b         New release notes (Markdown)
  --prerelease        Set prerelease = true
  --no-prerelease     Set prerelease = false

## Idempotency
YES. Re-running with the same values is a no-op on the server side and
returns the same release object.

## Atomicity
Atomic. Either all fields are updated together, or no change happens.

## Side effects
- Webhooks subscribed to "Release" events fire (release-updated event)
- Watchers may receive a re-announcement depending on platform settings

## Post-conditions (on success)
- Release metadata reflects the new values
- Git tag is unchanged

## Output (stdout, table mode)
  ✓ Release v1.0.0 updated
    Name:  1.0.0 Stable
    URL:   https://gitee.com/owner/repo/releases/tag/v1.0.0

## Common errors
- 404 Not Found: no release exists for that tag
- 403 Forbidden:  token lacks write scope
- 401 Unauthorized: token expired

## Race condition warning ⚠️
The shared mutable state is the release's metadata (name, body,
prerelease flag). There is no optimistic locking: two agents editing
the same release concurrently both PATCH, and the LAST write wins — the
earlier edit is silently lost. A view→edit gap has the same hazard: you
may overwrite changes made after your view. If the new content depends
on the old (e.g. appending to the body), re-read with
` + "`" + `gitee release view <tag> --output json` + "`" + ` immediately before
editing, or serialize edits across agents. Retrying with the same
values is safe (see Idempotency).

## OAuth scope required
projects (write)

## Rate limit
1000 req/hour.

## Example
  gitee release edit v1.0.0 -R owner/repo -n "1.0.0 (LTS)" -b "Long-term support release"

## See also
- ` + "`" + `gitee release view` + "`" + `    — read current state
- ` + "`" + `gitee release create` + "`" + `  — ship a new tag
- ` + "`" + `gitee release delete` + "`" + `  — remove the release envelope
`
