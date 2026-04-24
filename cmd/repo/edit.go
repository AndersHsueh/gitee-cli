package repo

import (
	"context"
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"
)

var editFlags struct {
	owner        string
	repo         string
	name         string
	description  string
	privateValue bool
	defaultBranch string
}

var editCmd = &cobra.Command{
	Use:   "edit [<owner>/<repo>]",
	Short: "Edit repository settings",
	Example: `  gitee repo edit owner/repo --description "New description"
  gitee repo edit --repo owner/repo --private`,
	RunE: runEdit,
}

func init() {
	f := editCmd.Flags()
	f.StringVar(&editFlags.owner, "owner", "", "Repository owner")
	f.StringVarP(&editFlags.repo, "repo", "R", "", "Repository name or owner/repo")
	f.StringVar(&editFlags.name, "name", "", "New repository name")
	f.StringVarP(&editFlags.description, "description", "d", "", "Repository description")
	f.BoolVar(&editFlags.privateValue, "private", false, "Make repository private")
	f.StringVar(&editFlags.defaultBranch, "default-branch", "", "Default branch name")
}

func runEdit(cobraCmd *cobra.Command, args []string) error {
	owner, repoName, err := resolveRepo(editFlags.owner, editFlags.repo, args)
	if err != nil {
		return err
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	req := types.RepositoryEditRequest{
		Name:          editFlags.name,
		Description:  editFlags.description,
		DefaultBranch: editFlags.defaultBranch,
	}
	if editFlags.name != "" || editFlags.description != "" || editFlags.defaultBranch != "" {
		req.Name = editFlags.name
		req.Description = editFlags.description
		req.DefaultBranch = editFlags.defaultBranch
	}

	path := api.ReposEdit(owner, repoName)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var repo types.Repository
	if err := client.Patch(ctx, path, req, &repo); err != nil {
		return fmt.Errorf("editing repository: %w", err)
	}

	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s Repository %s updated\n", green("✓"), repo.FullName)
	return nil
}
