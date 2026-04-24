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

var createFlags struct {
	name        string
	description string
	private     bool
	autoInit    bool
	gitignore   string
	license     string
	org         string
}

var createCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new repository",
	Example: `  gitee repo create --name my-repo --description "My new repo"
  gitee repo create --name my-repo --private --org my-org
  gitee repo create --name my-repo --gitignore Go --license MIT`,
	RunE: runCreate,
}

func init() {
	f := createCmd.Flags()
	f.StringVarP(&createFlags.name, "name", "n", "", "Repository name (required)")
	f.StringVarP(&createFlags.description, "description", "d", "", "Repository description")
	f.BoolVarP(&createFlags.private, "private", "p", false, "Make repository private")
	f.BoolVar(&createFlags.autoInit, "auto-init", true, "Initialize with README")
	f.StringVar(&createFlags.gitignore, "gitignore", "", "Gitignore template name")
	f.StringVar(&createFlags.license, "license", "", "License template name")
	f.StringVar(&createFlags.org, "org", "", "Organization to create repo under")
	_ = createCmd.MarkFlagRequired("name")
}

func runCreate(cobraCmd *cobra.Command, args []string) error {
	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	req := types.RepositoryCreateRequest{
		Name:              createFlags.name,
		Description:       createFlags.description,
		Private:           createFlags.private,
		AutoInit:          createFlags.autoInit,
		GitignoreTemplate:  createFlags.gitignore,
		LicenseTemplate:    createFlags.license,
		Organization:      createFlags.org,
	}

	var path string
	if createFlags.org != "" {
		path = api.ReposCreateForOrg(createFlags.org)
	} else {
		path = api.ReposCreate()
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var repo types.Repository
	if err := client.Post(ctx, path, req, &repo); err != nil {
		return fmt.Errorf("creating repository: %w", err)
	}

	bold := color.New(color.Bold).SprintFunc()
	green := color.New(color.FgGreen).SprintFunc()
	fmt.Printf("%s Repository %s created\n", green("✓"), bold(repo.FullName))
	fmt.Printf("  URL: %s\n", color.CyanString(repo.HTMLURL))
	return nil
}
