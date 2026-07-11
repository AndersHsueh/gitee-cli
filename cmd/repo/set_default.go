package repo

import (
	"fmt"

	"gitee-cli/cmd"
	"gitee-cli/internal/utils"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var setDefaultFlags struct {
	owner string
	repo  string
}

var setDefaultCmd = &cobra.Command{
	Use:   "set-default [owner/repo]",
	Short: "Set default repository for the current directory",
	Long: `Set the default repository used by gitee commands when none is
specified via --repo. The value is stored in the local git config
(.git/config) of the current repository.

Run with no arguments to view the current default. Use --owner and
--repo to set independently.`,
	Example: `  gitee repo set-default owner/repo
  gitee repo set-default           # show current default`,
	PreRunE: cmd.AgentHelpNA("repo set-default"),
	RunE:    runSetDefault,
}

func init() {
	f := setDefaultCmd.Flags()
	f.StringVar(&setDefaultFlags.owner, "owner", "", "Repository owner")
	f.StringVar(&setDefaultFlags.repo, "repo", "", "Repository name")

	RepoCmd.AddCommand(setDefaultCmd)
}

func runSetDefault(cobraCmd *cobra.Command, args []string) error {
	// Positional arg: owner/repo
	var targetOwner, targetRepo string
	if len(args) >= 1 {
		targetOwner, targetRepo = utils.ParseOwnerRepo(args[0])
	}
	if setDefaultFlags.owner != "" && setDefaultFlags.repo != "" {
		targetOwner, targetRepo = setDefaultFlags.owner, setDefaultFlags.repo
	}

	cyan := color.New(color.FgCyan).SprintFunc()
	dim := color.New(color.Faint).SprintFunc()

	// Read mode: no target specified
	if targetOwner == "" || targetRepo == "" {
		current, err := utils.GitConfigGet("gitee.default_repo")
		if err != nil {
			return fmt.Errorf("reading current default: %w", err)
		}
		if current == "" {
			fmt.Printf("%s No default repo set in this directory.\n", dim("note:"))
			fmt.Printf("  Set one with: %s\n", cyan("gitee repo set-default owner/repo"))
		} else {
			fmt.Printf("Default repo: %s\n", cyan(current))
		}
		return nil
	}

	// Write mode
	value := fmt.Sprintf("%s/%s", targetOwner, targetRepo)
	if err := utils.GitConfigSet("gitee.default_repo", value); err != nil {
		return fmt.Errorf("setting default repo: %w (not inside a git repo?)", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s Default repo set to %s in this directory\n", green("✓"), cyan(value))
	return nil
}
