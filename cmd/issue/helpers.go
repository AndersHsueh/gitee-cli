package issue

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"gitee-cli/cmd"
)

// resolveRepo returns owner and repoName from --owner/--repo flags or the
// combined --repo owner/repo flag on the root command.
func resolveRepo(owner, repo string) (string, string, error) {
	// If both are provided explicitly, use them
	if owner != "" && repo != "" {
		return owner, repo, nil
	}
	// Try combined format
	combined := repo
	if combined == "" {
		combined = cmd.GlobalRepo
	}
	if combined == "" {
		return "", "", fmt.Errorf("--repo is required (e.g. --repo owner/repo or --owner owner --repo name)")
	}
	parts := strings.SplitN(combined, "/", 2)
	if len(parts) == 2 && parts[0] != "" && parts[1] != "" {
		return parts[0], parts[1], nil
	}
	// If owner is set separately
	if owner != "" {
		return owner, combined, nil
	}
	return "", "", fmt.Errorf("invalid --repo format %q, expected owner/repo", combined)
}

// Stub command builder for not-yet-implemented sub-commands
func stubCmd(use, short string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(c *cobra.Command, args []string) error {
			return fmt.Errorf("'gitee issue %s' is not yet implemented", use)
		},
	}
}

var editCmd = stubCmd("edit <number>", "Edit an issue")
var closeCmd = stubCmd("close <number>", "Close an issue")
var reopenCmd = stubCmd("reopen <number>", "Reopen an issue")
var commentCmd = stubCmd("comment <number>", "Add a comment to an issue")

func init() {
	for _, c := range []*cobra.Command{editCmd, closeCmd, reopenCmd, commentCmd} {
		c.Flags().String("owner", "", "Repository owner")
		c.Flags().String("repo", "", "Repository name or owner/repo")
	}
}
