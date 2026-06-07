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
	"gitee-cli/internal/utils"
	"gitee-cli/pkg/types"

	"github.com/spf13/cobra"
)

var listFlags struct {
	limit int
}

var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List your organizations",
	Example: `  gitee org list`,
	RunE:    runList,
}

func init() {
	f := listCmd.Flags()
	f.IntVarP(&listFlags.limit, "limit", "L", 30, "Maximum number of orgs")
}

func runList(cobraCmd *cobra.Command, args []string) error {
	client, err := cmd.NewAPIClient()
	if err != nil {
		return err
	}

	path := fmt.Sprintf("%s?per_page=%d", api.OrgsList(), listFlags.limit)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cmd.GlobalTimeout)*time.Second)
	defer cancel()

	var orgs []types.Organization
	if err := client.Get(ctx, path, &orgs); err != nil {
		return fmt.Errorf("listing orgs: %w", err)
	}

	if cmd.GlobalOutput == "json" {
		b, _ := json.MarshalIndent(orgs, "", "  ")
		fmt.Println(string(b))
		return nil
	}

	if len(orgs) == 0 {
		fmt.Println("No organizations found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "LOGIN\tNAME\tDESCRIPTION\tPUBLIC REPOS")
	for _, o := range orgs {
		desc := utils.Truncate(o.Description, 50)
		if desc == "" {
			desc = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\n", o.Login, o.Name, desc, o.PublicRepos)
	}
	return w.Flush()
}
