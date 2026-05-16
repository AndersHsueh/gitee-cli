package repo

import (
	"context"
	"fmt"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"github.com/spf13/cobra"
)

var deleteFlags struct {
	owner string
	repo  string
	yes   bool
}

var deleteCmd = &cobra.Command{
	Use:   "delete [<owner>/<repo>]",
	Short: "Delete a repository",
	Example: `  gitee repo delete owner/repo
  gitee repo delete --repo owner/repo --yes`,
	RunE: runDelete,
}

func init() {
	f := deleteCmd.Flags()
	f.StringVar(&deleteFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&deleteFlags.repo, "repo", "R", "", "Repository name or owner/repo")
	f.BoolVar(&deleteFlags.yes, "yes", false, "Skip confirmation")
}

func runDelete(cobraCmd *cobra.Command, args []string) error {
	owner, repoName, err := resolveRepo(deleteFlags.owner, deleteFlags.repo, args)
	if err != nil {
		return err
	}

	if !deleteFlags.yes {
		fmt.Printf("Are you sure you want to delete repository %s/%s? [y/N] ", owner, repoName)
		var confirm string
		fmt.Scanln(&confirm)
		if confirm != "y" && confirm != "Y" {
			fmt.Println("Cancelled")
			return nil
		}
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := api.ReposDelete(owner, repoName)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	if err := client.Delete(ctx, path); err != nil {
		return fmt.Errorf("deleting repository: %w", err)
	}

	fmt.Printf("Repository %s/%s deleted\n", owner, repoName)
	return nil
}
