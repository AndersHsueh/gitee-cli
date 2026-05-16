package org

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"gitee-cli/cmd"
	"gitee-cli/internal/api"
	"gitee-cli/pkg/types"
	"github.com/spf13/cobra"
)

var orgRepoFlags struct {
	org   string
	limit int
}

var repoCmd = &cobra.Command{
	Use:     "repos <org>",
	Short:   "List organization repositories",
	Example: `  gitee org repos my-org`,
	RunE:    runOrgRepos,
}

func init() {
	f := repoCmd.Flags()
	f.StringVar(&orgRepoFlags.org, "org", "", "Organization name")
	f.IntVarP(&orgRepoFlags.limit, "limit", "L", 30, "Maximum number of repos")
}

func runOrgRepos(cobraCmd *cobra.Command, args []string) error {
	org := orgRepoFlags.org
	if len(args) > 0 {
		org = args[0]
	}
	if org == "" {
		return fmt.Errorf("organization name is required")
	}

	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := fmt.Sprintf("%s?per_page=%d", api.OrgRepos(org), orgRepoFlags.limit)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var repos []types.Repository
	if err := client.Get(ctx, path, &repos); err != nil {
		return fmt.Errorf("listing org repos: %w", err)
	}

	if cmd.GlobalOutput == "json" {
		b, _ := json.MarshalIndent(repos, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	if len(repos) == 0 {
		fmt.Println("No repositories found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tVISIBILITY\tLANGUAGE\tSTARS\tFORKS")
	for _, r := range repos {
		vis := "public"
		if r.Private {
			vis = "private"
		}
		lang := r.Language
		if lang == "" {
			lang = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%d\n", r.FullName, vis, lang, r.StarsCount, r.ForksCount)
	}
	return w.Flush()
}
