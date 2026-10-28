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

var createFlags struct {
	repo       string
	tag        string
	name       string
	body       string
	target     string
	prerelease bool
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new release",
	Example: `  gitee release create --repo owner/repo --tag v1.0.0 --name "1.0.0" --target main
  gitee release create -R owner/repo -t v0.2.0 --body "Initial beta" --prerelease`,
	// PreRunE fires BEFORE ValidateRequiredFlags so --agent-help works without --tag.
	PreRunE: func(c *cobra.Command, args []string) error {
		rt := cmd.RuntimeFromCmd(c)
		if rt.AgentHelp {
			cmd.PrintAgentHelp(releaseCreateAgentHelp)
		}
		return nil
	},
	RunE: runCreate,
}

func init() {
	f := createCmd.Flags()
	f.StringVarP(&createFlags.repo, "repo", "R", "", "Repository (owner/repo)")
	f.StringVarP(&createFlags.tag, "tag", "t", "", "Release tag name (required), e.g. v1.0.0")
	f.StringVarP(&createFlags.name, "name", "n", "", "Release display name")
	f.StringVarP(&createFlags.body, "body", "b", "", "Release notes (Markdown)")
	f.StringVar(&createFlags.target, "target", "", "Target branch / commit-ish (defaults to repo's default branch)")
	f.BoolVar(&createFlags.prerelease, "prerelease", false, "Mark as a prerelease")
	_ = createCmd.MarkFlagRequired("tag")
}

func runCreate(cobraCmd *cobra.Command, args []string) error {
	rt := cmd.RuntimeFromCmd(cobraCmd)
	owner, repoName, err := resolveReleaseRepo(createFlags.repo)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClientFrom(rt)
	if err != nil {
		return err
	}

	payload := types.ReleaseCreateRequest{
		TagName:         createFlags.tag,
		Name:            createFlags.name,
		Body:            createFlags.body,
		TargetCommitish: createFlags.target,
		Prerelease:      createFlags.prerelease,
	}

	path := api.ReleaseCreate(owner, repoName)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rt.Timeout)*time.Second)
	defer cancel()

	var release types.Release
	if err := client.Post(ctx, path, payload, &release); err != nil {
		return fmt.Errorf("creating release: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	bold := color.New(color.Bold).SprintFunc()
	cyan := color.New(color.FgCyan).SprintFunc()

	fmt.Println()
	fmt.Printf("%s Release %s created\n", green("✓"), bold(release.TagName))
	if release.Name != "" {
		fmt.Printf("  Name:  %s\n", release.Name)
	}
	fmt.Printf("  URL:   %s\n", cyan(release.HTMLURL))
	fmt.Println()
	return nil
}

const releaseCreateAgentHelp = `# gitee release create — Agent Help

## Purpose
Ship a new software version: wrap an existing git tag with release notes
and optional asset metadata. The git tag must already exist on the
remote; this command only attaches the release "envelope" to it.

## Pre-conditions
- Authenticated, with write access to the target repo
- The git tag (--tag) already exists on the remote (push it first)
- --tag is non-empty (Cobra MarkFlagRequired)

## Required flags
  --tag / -t  Release tag, e.g. v1.0.0

## Optional flags
  --name / -n        Display name (defaults to tag)
  --body / -b        Release notes in Markdown
  --target           Branch or commit SHA the tag points at
  --prerelease       Mark as prerelease (default false)
  --repo / -R        owner/repo

## Idempotency
NOT idempotent. Each call creates a NEW release envelope on the tag.
Re-running with the same --tag returns 409 Conflict (release already exists).
Use ` + "`" + `gitee release edit` + "`" + ` to modify an existing one.

## Atomicity
Atomic. Either the release is created and a tag_name is returned, or no
change happens.

## Side effects
- Email/web notification to watchers of the repo
- Webhooks subscribed to "Release" events fire
- The release is publicly visible immediately

## Post-conditions (on success)
- A new release with the given tag is visible on the repo
- Stdout contains the tag name and HTML URL
- The underlying git tag is unchanged (only metadata is added)

## Output (stdout, table mode)
  ✓ Release v1.0.0 created
    Name:  1.0.0
    URL:   https://gitee.com/owner/repo/releases/tag/v1.0.0

## Common errors
- 401 Unauthorized: token expired → gitee auth login
- 403 Forbidden:    token lacks write scope
- 404 Not Found:    repo doesn't exist
- 409 Conflict:     release for this tag already exists → use release edit
- 422 Unprocessable: tag doesn't exist on remote → git push origin <tag> first

## Race condition warning ⚠️
DO NOT call ` + "`" + `gitee release view` + "`" + ` immediately after. Gitee API indexing
can lag 1-2 seconds; the new release may briefly 404. Use the tag name
from THIS output directly.

## Next steps (orchestration hints)
1. Push the git tag first: ` + "`" + `git push origin v1.0.0` + "`" + `
2. Create the release: ` + "`" + `gitee release create --tag v1.0.0 --body "..."` + "`" + `
3. Upload binaries (if any): use the web UI or Gitee API for assets
4. Announce: post the URL to chat / email

## OAuth scope required
projects (write)

## Rate limit
Gitee API v5: 1000 req/hour. Each call = 1 request.

## Example (closing the loop)
  # 1. Ensure tag exists locally and on remote
  git tag v1.0.0
  git push origin v1.0.0

  # 2. Create the release
  OUT=$(gitee release create -R owner/repo -t v1.0.0 -n "1.0.0" -b "Bug fixes and performance improvements")
  TAG=$(echo "$OUT" | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+' | head -1)

  # 3. Verify via list (after a brief wait)
  sleep 2
  gitee release list -R owner/repo | grep "$TAG"

## See also
- ` + "`" + `gitee release list` + "`" + `       — enumerate releases
- ` + "`" + `gitee release view` + "`" + `       — read one release
- ` + "`" + `gitee release edit` + "`" + `       — modify an existing release
- ` + "`" + `gitee release delete` + "`" + `     — remove the release envelope (keeps git tag)
`
