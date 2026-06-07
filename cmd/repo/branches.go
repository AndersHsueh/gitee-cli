package repo

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"

	"github.com/spf13/cobra"
)

var branchesFlags struct {
	repo string
}

var branchesCmd = &cobra.Command{
	Use:     "branches",
	Short:   "List branches in a repository",
	Example: `  gitee repo branches --repo owner/repo`,
	RunE:    runBranches,
}

func init() {
	f := branchesCmd.Flags()
	f.StringVarP(&branchesFlags.repo, "repo", "R", "", "Repository (owner/repo)")
}

func runBranches(cobraCmd *cobra.Command, args []string) error {
	repo := branchesFlags.repo
	if repo == "" {
		repo = cmd.GlobalRepo
	}
	if repo == "" {
		return fmt.Errorf("--repo is required (e.g. --repo owner/repo)")
	}

	parts := strings.SplitN(repo, "/", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid --repo format, expected owner/repo")
	}
	owner, repoName := parts[0], parts[1]

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := api.ReposBranches(owner, repoName)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var branches []types.Branch
	if err := client.Get(ctx, path, &branches); err != nil {
		return fmt.Errorf("listing branches: %w", err)
	}

	if cmd.GlobalOutput == "json" {
		b, _ := json.MarshalIndent(branches, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	fmt.Printf("Branches in %s/%s:\n", owner, repoName)
	for _, b := range branches {
		fmt.Printf("  %s", b.Name)
		if b.Protected {
			fmt.Printf(" (protected)")
		}
		fmt.Println()
	}
	return nil
}
